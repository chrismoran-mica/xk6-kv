package kv

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chrismoran-mica/go-cache"
	"github.com/dop251/goja"
	"go.k6.io/k6/js/modules"
)

type (
	// KV is the global module instance that will create Client
	// instances for each VU.
	KV struct {
		db *cache.Cache[string, interface{}]
	}

	// ModuleInstance represents an instance of the JS module.
	ModuleInstance struct {
		vu modules.VU
		*KV
	}
)

// Ensure the interfaces are implemented correctly
var (
	_    modules.Instance = &ModuleInstance{}
	_    modules.Module   = &KV{}
	once sync.Once
)

func init() {
	modules.Register("k6/x/kv", New())
}

// New returns a pointer to a new KV instance
func New() *KV {
	return &KV{
		db: cache.New[string, interface{}](cache.NoExpiration, 5*time.Minute),
	}
}

// NewModuleInstance implements the modules.Module interface and returns
// a new instance for each VU.
func (k *KV) NewModuleInstance(vu modules.VU) modules.Instance {
	return &ModuleInstance{vu: vu, KV: k}
}

// Exports implements the modules.Instance interface and returns
// the exports of the JS module.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"Cache": mi.NewCache,
		}}
}

// NewCache is the JS constructor for the Cache
func (mi *ModuleInstance) NewCache(call goja.ConstructorCall) *goja.Object {
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

	once.Do(func() {
		db := cache.New[string, interface{}](expiration, cleanup)
		if len(mi.KV.db.Items()) > 0 {
			mi.KV.db.Flush()
		}
		mi.KV.db = db
	})

	return rt.ToValue(mi.KV).ToObject(rt)
}

// Add the given key with the given value only if it does not already exist in the cache
func (k *KV) Add(key string, value interface{}, ttl int) error {
	return k.db.Add(key, value, time.Duration(ttl)*time.Second)
}

// Set the given key with the given value.
func (k *KV) Set(key string, value interface{}) error {
	k.db.Set(key, value, cache.DefaultExpiration)
	return nil
}

// Replace the given key with the given value only if it already exists in the cache.
func (k *KV) Replace(key string, value interface{}, ttl int) error {
	return k.db.Replace(key, value, time.Duration(ttl)*time.Second)
}

// SetWithTTLInSecond Sets the given key with the given value with TTL in second
func (k *KV) SetWithTTLInSecond(key string, value interface{}, ttl int) error {
	k.db.Set(key, value, time.Duration(ttl)*time.Second)
	return nil
}

// Get returns the value for the given key.
func (k *KV) Get(key string) (interface{}, error) {
	if v, ok := k.db.Get(key); ok {
		return v, nil
	}
	return nil, fmt.Errorf("error in get value with key %s", key)
}

// ViewPrefix return all the key value pairs where the key starts with some prefix.
func (k *KV) ViewPrefix(prefix string) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range k.db.Items() {
		if strings.HasPrefix(k, prefix) {
			m[k] = v
		}
	}
	return m
}

// Delete the given key
func (k *KV) Delete(key string) error {
	k.db.Delete(key)
	return nil
}
