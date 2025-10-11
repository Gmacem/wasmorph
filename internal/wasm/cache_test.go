package wasm

import (
	"context"
	"testing"
)

func TestNoOpCache(t *testing.T) {
	cache := &NoOpCache{}
	ctx := context.Background()

	runtime := &Runtime{}

	if _, found := cache.Get(ctx, "key"); found {
		t.Error("NoOpCache should never return cached values")
	}

	if !cache.Set(ctx, "key", runtime, 100) {
		t.Error("NoOpCache.Set should always return true")
	}

	cache.Delete(ctx, "key")
	cache.Close()
}

func TestRistrettoCache(t *testing.T) {
	cache := NewRistrettoCache(&RuntimeCacheConfig{
		MaxCost:     1024,
		NumCounters: 10,
		BufferItems: 64,
	})
	defer cache.Close()

	ctx := context.Background()
	runtime := &Runtime{}

	if _, found := cache.Get(ctx, "key"); found {
		t.Error("Cache should be empty initially")
	}

	if !cache.Set(ctx, "key", runtime, 100) {
		t.Error("Failed to set cache")
	}

	cache.(*RistrettoCache).cache.Wait()

	if cached, found := cache.Get(ctx, "key"); !found || cached != runtime {
		t.Error("Failed to get cached value")
	}

	cache.Delete(ctx, "key")

	if _, found := cache.Get(ctx, "key"); found {
		t.Error("Cache should be empty after delete")
	}
}
