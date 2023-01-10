package kv

import (
	"fmt"
	"github.com/dop251/goja"
	"go.k6.io/k6/js/modules"
	"os"
	"strings"
	"sync"
	"time"
)

func init() {
	modules.Register("k6/x/kv", New())
}

type (
	// RootModule is the global module instance that will create module
	// instances for each VU.
	RootModule struct{}

	// ModuleInstance represents an instance of the JS module.
	ModuleInstance struct {
		// vu provides methods for accessing internal k6 objects for a VU
		vu modules.VU
		// kv is the exported type
		exports map[string]interface{}
	}
)

// Ensure the interfaces are implemented correctly.
var (
	_      modules.Instance = &ModuleInstance{}
	_      modules.Module   = &RootModule{}
	once   sync.Once
	client *Client
)

// New returns a pointer to a new RootModule instance.
func New() *RootModule {
	return &RootModule{}
}

// KV is the k6 key-value extension.
type KV struct {
	vu      modules.VU
	exports map[string]interface{}
}

type ValueTTL struct {
	value interface{}
	tm    *time.Timer
}

type Client struct {
	db map[string]*ValueTTL
	mu sync.Mutex
}

// NewModuleInstance implements the modules.Module interface returning a new instance for each VU.
func (r *RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	mi := &ModuleInstance{
		vu:      vu,
		exports: make(map[string]interface{}),
	}

	mi.exports["Client"] = mi.newClient

	return mi
}

// Exports implements the modules.Instance interface and returns the exported types for the JS module.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{
		Named:   mi.exports,
		Default: newClient(),
	}
}

func (mi *ModuleInstance) newClient(_ goja.ConstructorCall, rt *goja.Runtime) *goja.Object {
	return rt.ToValue(newClient()).ToObject(rt)
}

func newClient() *Client {
	once.Do(func() {
		db := make(map[string]*ValueTTL, 500)
		client = &Client{db: db}
	})
	return client
}

// Set the given key with the given value.
func (c *Client) Set(key string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var cv *ValueTTL
	var ok bool
	if cv, ok = c.db[key]; ok {
		if cv.tm != nil {
			stopped := cv.tm.Stop()
			if !stopped {
				_, _ = fmt.Fprintf(os.Stderr, "set: wtf how the...?\n")
				<-cv.tm.C
			}
		}
		cv.value = value
		cv.tm = nil
	} else {
		cv = &ValueTTL{
			value: value,
			tm:    nil,
		}
	}
	c.db[key] = cv
	return nil
}

// SetWithTTLInSecond the given key with the given value with TTL in second
func (c *Client) SetWithTTLInSecond(key string, value interface{}, ttl int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var cv *ValueTTL
	var ok bool
	if cv, ok = c.db[key]; ok {
		if cv.tm != nil {
			stopped := cv.tm.Stop()
			if !stopped {
				_, _ = fmt.Fprintf(os.Stderr, "setWithTTLInSecond: wtf how the...?\n")
				<-cv.tm.C
			}
		}
		cv.value = value
	} else {
		cv = &ValueTTL{
			value: value,
		}
	}
	cv.tm = time.NewTimer(time.Duration(ttl) * time.Second)
	c.db[key] = cv
	go func() {
		<-cv.tm.C
		c.mu.Lock()
		delete(c.db, key)
		c.mu.Unlock()
	}()
	return nil
}

// Get returns the value for the given key.
func (c *Client) Get(key string) (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cv, ok := c.db[key]; ok {
		return cv.value, nil
	}
	return "", fmt.Errorf("error in get value with key %s", key)
}

// ViewPrefix return all the key value pairs where the key starts with some prefix.
func (c *Client) ViewPrefix(prefix string) map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	m := make(map[string]interface{})
	for k, v := range c.db {
		if strings.HasPrefix(k, prefix) {
			m[k] = v.value
		}
	}
	return m
}

// Delete the given key
func (c *Client) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cv, ok := c.db[key]; ok {
		if cv.tm != nil {
			stopped := cv.tm.Stop()
			if !stopped {
				_, _ = fmt.Fprintf(os.Stderr, "delete: wtf how the...?\n")
				<-cv.tm.C
			}
		}
	}
	delete(c.db, key)
	return nil
}
