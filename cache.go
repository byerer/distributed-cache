package distributed_cache

import (
	"sync"
	"time"

	"distributed-cache/strategy"
	"distributed-cache/strategy/lru"
)

type Cache struct {
	mu       sync.Mutex
	maxBytes int64
	nBytes   int64
	eviction strategy.EvictionStrategy
}

func NewCache(maxBytes int64, eviction strategy.EvictionStrategy) *Cache {
	c := &Cache{
		maxBytes: maxBytes,
		eviction: eviction,
	}
	eviction.SetRemover(c)
	return c
}

func DefaultCache() *Cache {
	return NewCache(1024, lru.New(nil))
}

func (c *Cache) add(key string, value ByteView, expire time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.eviction == nil {
		c.eviction = lru.New()
		c.eviction.SetRemover(c)
	}
	c.eviction.Add(key, value, expire)
	c.nBytes += int64(len(key)) + int64(value.Len())
	for c.nBytes > c.maxBytes {
		c.eviction.RemoveOldest()
	}
}

func (c *Cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.eviction == nil {
		c.eviction = lru.New()
		c.eviction.SetRemover(c)
	}
	if v, ok := c.eviction.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}

func (c *Cache) OnEntryRemoved(key string, value strategy.Value) {
	c.nBytes -= int64(len(key)) + int64(value.Len())
}
