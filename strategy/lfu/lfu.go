package lfu

import (
	"container/list"
	"time"

	"distributed-cache/strategy"
)

type LFU struct {
	cache     map[string]*list.Element
	frequency map[int]*list.List
	remover   strategy.EntryRemover
	OnEvicted func(key string, value strategy.Value)
}

type entry struct {
	freq   int
	key    string
	value  strategy.Value
	expire time.Time
}

type Option func(*LFU)

func WithOnEvicted(onEvicted func(string, strategy.Value)) Option {
	return func(lfu *LFU) {
		lfu.OnEvicted = onEvicted
	}
}

func New(opts ...Option) *LFU {
	l := &LFU{
		cache:     make(map[string]*list.Element),
		frequency: make(map[int]*list.List),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *LFU) Get(key string) (value strategy.Value, ok bool) {
	if ele, ok := l.cache[key]; ok {
		kv := ele.Value.(*entry)
		l.removeElement(ele)
		if !kv.expire.IsZero() && kv.expire.Before(time.Now()) {
			return nil, false
		}
		kv.freq++
		e := l.addElement(kv)
		l.cache[key] = e
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
		l.removeElement(ele)
		kv.freq++
	} else {
		e := l.addElement(&entry{
			freq:   1,
			key:    key,
			value:  value,
			expire: expire,
		})
		l.cache[key] = e
	}
}

func (l *LFU) RemoveOldest() {
	minFreq := -1
	for freq := range l.frequency {
		if minFreq == -1 || freq < minFreq {
			minFreq = freq
		}
	}

	if minFreq != -1 {
		ll := l.frequency[minFreq]
		if ll != nil && ll.Len() > 0 {
			ele := ll.Back()
			l.removeElement(ele)
			kv := ele.Value.(*entry)
			delete(l.cache, kv.key)
			if l.remover != nil {
				l.remover.OnEntryRemoved(kv.key, kv.value)
			}
			if l.OnEvicted != nil {
				l.OnEvicted(kv.key, kv.value)
			}
		}
	}
}

func (l *LFU) SetRemover(remover strategy.EntryRemover) {
	l.remover = remover
}

func (l *LFU) removeElement(ele *list.Element) {
	kv := ele.Value.(*entry)
	l.frequency[kv.freq].Remove(ele)
	if l.frequency[kv.freq].Len() == 0 {
		delete(l.frequency, kv.freq)
	}
}

func (l *LFU) addElement(kv *entry) *list.Element {
	if l.frequency[kv.freq] == nil {
		l.frequency[kv.freq] = list.New()
	}
	return l.frequency[kv.freq].PushFront(kv)
}
