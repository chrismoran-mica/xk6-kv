package kv

import (
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/patrickmn/go-cache"
	"go.k6.io/k6/js/modules"
)

type (
	// KV is the global module instance that will create Client
	// instances for each VU.
	KV struct{}

	// ModuleInstance represents an instance of the JS module.
	ModuleInstance struct {
		vu modules.VU
		*Client
	}
)

// Ensure the interfaces are implemented correctly
var (
	_ modules.Instance = &ModuleInstance{}
	_ modules.Module   = &KV{}
)

type Client struct {
	vu modules.VU
	db *cache.Cache
}

var check = false
var client *Client

func init() {
	modules.Register("k6/x/kv", New())
}

// New returns a pointer to a new KV instance
func New() *KV {
	return &KV{}
}

// NewModuleInstance implements the modules.Module interface and returns
// a new instance for each VU.
func (*KV) NewModuleInstance(vu modules.VU) modules.Instance {
	return &ModuleInstance{vu: vu, Client: &Client{vu: vu}}
}

// Exports implements the modules.Instance interface and returns
// the exports of the JS module.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"Client": mi.NewClient,
		}}
}

// NewClient is the JS constructor for the Client
func (mi *ModuleInstance) NewClient(call goja.ConstructorCall) *goja.Object {
	rt := mi.vu.Runtime()

	var expiration = cache.NoExpiration
	var cleanup = 5 * time.Minute
	if len(call.Arguments) == 1 {
		expiration = time.Duration(call.Arguments[0].ToInteger())
	}

	if len(call.Arguments) == 2 {
		expiration = time.Duration(call.Arguments[0].ToInteger())
		cleanup = time.Duration(call.Arguments[1].ToInteger())
	}

	if check != true {
		db := cache.New(expiration, cleanup)
		client = &Client{vu: mi.vu, db: db}
		check = true
	}

	return rt.ToValue(client).ToObject(rt)
}

// Set the given key with the given value.
func (c *Client) Set(key string, value interface{}) error {
	c.db.Set(key, value, cache.DefaultExpiration)
	return nil
}

// SetWithTTLInSecond Sets the given key with the given value with TTL in second
func (c *Client) SetWithTTLInSecond(key string, value interface{}, ttl int) error {
	c.db.Set(key, value, time.Duration(ttl)*time.Second)
	return nil
}

// Get returns the value for the given key.
func (c *Client) Get(key string) (interface{}, error) {
	if v, ok := c.db.Get(key); ok {
		return v, nil
	}
	return nil, fmt.Errorf("error in get value with key %s", key)
}

// ViewPrefix return all the key value pairs where the key starts with some prefix.
func (c *Client) ViewPrefix(prefix string) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range c.db.Items() {
		if strings.HasPrefix(k, prefix) {
			m[k] = v
		}
	}
	return m
}

// Delete the given key
func (c *Client) Delete(key string) error {
	c.db.Delete(key)
	return nil
}
