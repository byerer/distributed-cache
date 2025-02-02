package lru

import (
	"container/list"
	"time"

	"distributed-cache/strategy"
)

type LRU struct {
	ll        *list.List
	cache     map[string]*list.Element
	OnEvicted func(key string, value strategy.Value)
	remover   strategy.EntryRemover
}

type entry struct {
	key    string
	value  strategy.Value
	expire time.Time
}

type Option func(*LRU)

func WithOnEvicted(onEvicted func(string, strategy.Value)) Option {
	return func(lru *LRU) {
		lru.OnEvicted = onEvicted
	}
}

func New(option ...Option) *LRU {
	l := &LRU{
		ll:    list.New(),
		cache: make(map[string]*list.Element),
	}
	for _, opt := range option {
		opt(l)
	}
	return l
}

func (c *LRU) Get(key string) (value strategy.Value, ok bool) {
	if c.cache == nil {
		c.cache = make(map[string]*list.Element)
		c.ll = list.New()
	}
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		if !kv.expire.IsZero() && kv.expire.Before(time.Now()) {
			c.removeElement(ele)
			return nil, false
		}
		return kv.value, true
	}
	return
}

func (c *LRU) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
		if c.remover != nil {
			c.remover.OnEntryRemoved(kv.key, kv.value)
		}
	}
}

func (c *LRU) Add(key string, value strategy.Value, expire time.Time) {
	if c.cache == nil {
		c.cache = make(map[string]*list.Element)
		c.ll = list.New()
	}
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		kv.value = value
		kv.expire = expire
	} else {
		ele := c.ll.PushFront(&entry{key, value, expire})
		c.cache[key] = ele
	}
}

func (c *LRU) SetRemover(remover strategy.EntryRemover) {
	c.remover = remover
}

//func (c *LRU) Len() int {
//	return c.ll.Len()
//}

func (c *LRU) removeElement(ele *list.Element) {
	c.ll.Remove(ele)
	kv := ele.Value.(*entry)
	delete(c.cache, kv.key)
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
	if c.remover != nil {
		c.remover.OnEntryRemoved(kv.key, kv.value)
	}
}
