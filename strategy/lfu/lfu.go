package lfu

import (
	"container/list"
	"time"

	"distributed-cache/strategy"
)

type LFU struct {
	cache     map[string]*list.Element
	frequency map[int]*list.List
}

type entry struct {
	freq   int
	key    string
	value  strategy.Value
	expire time.Time
}

func NewLFU(cache map[string]*list.Element) *LFU {
	return &LFU{
		cache:     cache,
		frequency: make(map[int]*list.List),
	}
}

func (l *LFU) Get(key string) (value strategy.Value, ok bool) {
	if ele, ok := l.cache[key]; ok {
		kv := ele.Value.(*entry)
		if !kv.expire.IsZero() && kv.expire.Before(time.Now()) {
			l.removeElement(ele)
			return nil, false
		}
		kv.freq++
		if l.frequency[kv.freq] == nil {
			l.frequency[kv.freq] = list.New()
		}
		l.frequency[kv.freq].PushFront(ele)
		l.frequency[kv.freq-1].Remove(ele)
		if l.frequency[kv.freq-1].Len() == 0 {
			delete(l.frequency, kv.freq-1)
		}
		return kv.value, true
	}
	return
}

func (l *LFU) Add(key string, value strategy.Value, expire time.Time) {
	if l.cache == nil {
		l.cache = make(map[string]*list.Element)
		l.frequency = make(map[int]*list.List)
		l.frequency[1] = list.New()
	}
	if ele, ok := l.cache[key]; ok {
		kv := ele.Value.(*entry)
		kv.value = value
		kv.expire = expire
		kv.freq++
		if l.frequency[kv.freq] == nil {
			l.frequency[kv.freq] = list.New()
		}
		l.frequency[kv.freq].PushFront(ele)
		l.frequency[kv.freq-1].Remove(ele)
		if l.frequency[kv.freq-1].Len() == 0 {
			delete(l.frequency, kv.freq-1)
		}
	} else {
		if l.frequency[1] == nil {
			l.frequency[1] = list.New()
		}
		ele := l.frequency[1].PushFront(&entry{
			freq:   1,
			key:    key,
			value:  value,
			expire: expire,
		})
		l.cache[key] = ele
	}
}

func (l *LFU) removeElement(ele *list.Element) {
	kv := ele.Value.(*entry)
	delete(l.cache, kv.key)
	l.frequency[kv.freq].Remove(ele)
	if l.frequency[kv.freq].Len() == 0 {
		delete(l.frequency, kv.freq)
	}
}
