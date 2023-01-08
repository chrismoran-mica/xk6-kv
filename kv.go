package kv

import (
	"fmt"
	"github.com/dop251/goja"
	"os"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v3"
	"go.k6.io/k6/js/modules"
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

type Client struct {
	db *badger.DB
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
		name := os.Getenv("XK6_KV_NAME")
		if name == "" {
			name = "/tmp/badger"
		}
		var db *badger.DB
		if _, memory := os.LookupEnv("XK6_KV_MEMORY"); memory {
			db, _ = badger.Open(badger.DefaultOptions("").WithLoggingLevel(badger.ERROR).WithInMemory(true))
		} else {
			db, _ = badger.Open(badger.DefaultOptions(name).WithLoggingLevel(badger.ERROR))
		}
		client = &Client{db: db}
	})
	return client
}

// Set the given key with the given value.
func (c *Client) Set(key string, value string) error {
	err := c.db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(key), []byte(value))
		return err
	})
	return err
}

// SetWithTTLInSecond the given key with the given value with TTL in second
func (c *Client) SetWithTTLInSecond(key string, value string, ttl int) error {
	err := c.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), []byte(value)).WithTTL(time.Duration(ttl) * time.Second)
		err := txn.SetEntry(e)
		return err
	})
	return err
}

// Get returns the value for the given key.
func (c *Client) Get(key string) (string, error) {
	var valCopy []byte
	_ = c.db.View(func(txn *badger.Txn) error {
		item, _ := txn.Get([]byte(key))
		if item != nil {
			valCopy, _ = item.ValueCopy(nil)
		}
		return nil
	})
	if len(valCopy) > 0 {
		return string(valCopy), nil
	}
	return "", fmt.Errorf("error in get value with key %s", key)
}

// ViewPrefix return all the key value pairs where the key starts with some prefix.
func (c *Client) ViewPrefix(prefix string) map[string]string {
	m := make(map[string]string)
	_ = c.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(prefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				m[string(k)] = string(v)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return m
}

// Delete the given key
func (c *Client) Delete(key string) error {
	err := c.db.Update(func(txn *badger.Txn) error {
		item, _ := txn.Get([]byte(key))
		if item != nil {
			err := txn.Delete([]byte(key))
			return err
		}
		return nil
	})
	return err
}
