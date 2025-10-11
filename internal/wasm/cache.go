package wasm

import (
	"context"

	"github.com/dgraph-io/ristretto"
)

type RuntimeCache interface {
	Get(ctx context.Context, key string) (*Runtime, bool)
	Set(ctx context.Context, key string, runtime *Runtime, cost int64) bool
	Delete(ctx context.Context, key string)
	Close()
}

type NoOpCache struct{}

func (c *NoOpCache) Get(ctx context.Context, key string) (*Runtime, bool) {
	return nil, false
}

func (c *NoOpCache) Set(ctx context.Context, key string, runtime *Runtime, cost int64) bool {
	return true
}

func (c *NoOpCache) Delete(ctx context.Context, key string) {}

func (c *NoOpCache) Close() {}

type RuntimeCacheConfig struct {
	MaxCost     int64
	NumCounters int64
	BufferItems int64
}

func NewRistrettoCache(config *RuntimeCacheConfig) RuntimeCache {
	if config == nil {
		config = &RuntimeCacheConfig{
			MaxCost:     100 << 20,
			NumCounters: 1000,
			BufferItems: 64,
		}
	}

	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.NumCounters,
		MaxCost:     config.MaxCost,
		BufferItems: config.BufferItems,
		OnEvict: func(item *ristretto.Item) {
			if runtime, ok := item.Value.(*Runtime); ok && runtime != nil {
				runtime.Close()
			}
		},
	})
	if err != nil {
		panic(err)
	}

	return &RistrettoCache{cache: cache}
}

type RistrettoCache struct {
	cache *ristretto.Cache
}

func (c *RistrettoCache) Get(ctx context.Context, key string) (*Runtime, bool) {
	if val, found := c.cache.Get(key); found {
		if runtime, ok := val.(*Runtime); ok {
			return runtime, true
		}
	}
	return nil, false
}

func (c *RistrettoCache) Set(ctx context.Context, key string, runtime *Runtime, cost int64) bool {
	return c.cache.Set(key, runtime, cost)
}

func (c *RistrettoCache) Delete(ctx context.Context, key string) {
	c.cache.Del(key)
}

func (c *RistrettoCache) Close() {
	c.cache.Close()
}
