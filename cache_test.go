package distributed_cache

import (
	"sync"
	"testing"
	"time"

	"distributed-cache/strategy/lfu"
	"distributed-cache/strategy/lru"
)

func TestCache_Basic(t *testing.T) {
	c := DefaultCache()

	key := "key1"
	value := ByteView{b: []byte("value1")}
	c.add(key, value, time.Time{})

	if v, ok := c.get(key); !ok || string(v.b) != string(value.b) {
		t.Fatalf("cache hit key1=value1 failed")
	}

	if _, ok := c.get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

func TestCache_LRU(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := ByteView{b: []byte("value1")}, ByteView{b: []byte("value2")}, ByteView{b: []byte("value3")}

	// maxBytes设置为只能容纳两个键值对的大小
	maxBytes := int64(len(k1) + v1.Len() + len(k2) + v2.Len())
	c := NewCache(maxBytes, lru.New(nil))

	c.add(k1, v1, time.Time{})
	c.add(k2, v2, time.Time{})
	c.add(k3, v3, time.Time{})

	// k1应该被淘汰
	if _, ok := c.get(k1); ok {
		t.Fatalf("maxBytes test failed: k1 should be evicted")
	}
	// k2和k3应该存在
	if _, ok := c.get(k2); !ok {
		t.Fatalf("maxBytes test failed: k2 should exist")
	}
	if _, ok := c.get(k3); !ok {
		t.Fatalf("maxBytes test failed: k3 should exist")
	}
}

func TestCache_LFU(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := ByteView{b: []byte("value1")}, ByteView{b: []byte("value2")}, ByteView{b: []byte("value3")}

	// maxBytes设置为只能容纳两个键值对的大小
	maxBytes := int64(len(k1) + v1.Len() + len(k2) + v2.Len())
	c := NewCache(maxBytes, lfu.New(nil))

	c.add(k1, v1, time.Time{})
	c.add(k2, v2, time.Time{})
	c.add(k3, v3, time.Time{})

	// k1应该被淘汰
	if _, ok := c.get(k1); ok {
		t.Fatalf("maxBytes test failed: k1 should be evicted")
	}
	// k2和k3应该存在
	if _, ok := c.get(k2); !ok {
		t.Fatalf("maxBytes test failed: k2 should exist")
	}
	if _, ok := c.get(k3); !ok {
		t.Fatalf("maxBytes test failed: k3 should exist")
	}

}

func TestCache_Expire(t *testing.T) {
	c := &Cache{
		maxBytes: 1024,
	}

	key := "key1"
	value := ByteView{b: []byte("value1")}
	expire := time.Now().Add(50 * time.Millisecond)
	c.add(key, value, expire)

	// 未过期，应该能获取到
	if _, ok := c.get(key); !ok {
		t.Fatalf("cache get expired key failed: should not expire")
	}

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	// 已过期，应该获取不到
	if _, ok := c.get(key); ok {
		t.Fatalf("cache get expired key failed: should expire")
	}
}

func TestCache_Concurrent(t *testing.T) {
	c := &Cache{
		maxBytes: 1024,
	}

	n := 100
	wg := sync.WaitGroup{}
	wg.Add(n)

	// 并发写入
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			key := string([]byte{'k', byte(i)})
			value := ByteView{b: []byte{'v', byte(i)}}
			c.add(key, value, time.Time{})
		}(i)
	}
	wg.Wait()

	// 并发读取
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			key := string([]byte{'k', byte(i)})
			if _, ok := c.get(key); !ok {
				t.Errorf("concurrent test failed: key %s not found", key)
			}
		}(i)
	}
	wg.Wait()
}

func TestCache_NilValue(t *testing.T) {
	c := &Cache{
		maxBytes: 1024,
	}

	if _, ok := c.get("key1"); ok {
		t.Fatal("should return not found for nil cache")
	}

	c.add("key1", ByteView{b: []byte("value1")}, time.Time{})
	if _, ok := c.get("key1"); !ok {
		t.Fatal("should find key1 after add")
	}
}
