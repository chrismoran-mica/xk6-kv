package kv

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bool64/cache"
	"github.com/dop251/goja"
	"go.k6.io/k6/js/modules"
)

type (
	// KV is the global module instance that will create Client
	// instances for each VU.
	KV struct {
		db *cache.ShardedMapOf[interface{}]
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
		db: cache.NewShardedMapOf[interface{}](cache.Config{
			Name:                     "kv",
			TimeToLive:               1 * time.Second,
			DeleteExpiredAfter:       2 * time.Second,
			DeleteExpiredJobInterval: 1 * time.Second,
			HeapInUseSoftLimit:       64 * 1024 * 1024,
			EvictFraction:            0.2,
		}.Use),
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

	var ttl = 1 * time.Second
	if len(call.Arguments) == 1 {
		ttl = time.Duration(call.Arguments[0].ToInteger()) * time.Second
	}

	once.Do(func() {
		db := cache.NewShardedMapOf[interface{}](cache.Config{
			Name:                     "kv",
			TimeToLive:               ttl,
			DeleteExpiredAfter:       ttl * 2,
			DeleteExpiredJobInterval: ttl,
			HeapInUseSoftLimit:       64 * 1024 * 1024,
		}.Use)
		if mi.KV.db.Len() > 0 {
			mi.KV.db.DeleteAll(context.Background())
		}
		mi.KV.db = db
	})

	return rt.ToValue(mi.KV).ToObject(rt)
}

// Set the given key with the given value.
func (k *KV) Set(key string, value interface{}) error {
	k.db.Store([]byte(key), value)
	return nil
}

// SetWithTTLInSecond Sets the given key with the given value with TTL in second
func (k *KV) SetWithTTLInSecond(key string, value interface{}, ttl int) error {
	return k.db.Write(cache.WithTTL(context.Background(), time.Duration(ttl)*time.Second, true), []byte(key), value)
}

// Get returns the value for the given key.
func (k *KV) Get(key string) (interface{}, error) {
	if v, ok := k.db.Load([]byte(key)); ok {
		return v, nil
	}
	return nil, fmt.Errorf("error in get value with key %s", key)
}

// ViewPrefix return all the key value pairs where the key starts with some prefix.
func (k *KV) ViewPrefix(prefix string) map[string]interface{} {
	m := make(map[string]interface{})
	if count, err := k.db.Walk(func(e cache.EntryOf[interface{}]) error {
		if strings.HasPrefix(string(e.Key()), prefix) {
			m[string(e.Key())] = e.Value()
		}
		return nil
	}); err == nil && count > 0 {
		return m
	}

	return nil
}

func (k *KV) Len() int {
	return k.db.Len()
}

// Delete the given key
func (k *KV) Delete(key string) error {
	return k.db.Delete(context.Background(), []byte(key))
}

// Delete the given key
func (k *KV) DeleteAll() {
	k.db.DeleteAll(context.Background())
}
