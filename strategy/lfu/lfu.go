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

func New(onEvicted func(string, strategy.Value)) *LFU {
	return &LFU{
		cache:     make(map[string]*list.Element),
		frequency: make(map[int]*list.List),
		OnEvicted: func(key string, value strategy.Value) {
			if onEvicted != nil {
				onEvicted(key, value)
			}
		},
	}
}

func (l *LFU) Get(key string) (value strategy.Value, ok bool) {
	if ele, ok := l.cache[key]; ok {
		kv := ele.Value.(*entry)
		l.removeElement(ele)
		if !kv.expire.IsZero() && kv.expire.Before(time.Now()) {
			return nil, false
		}
		kv.freq++
		l.addElement(kv)
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
		l.addElement(kv)
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
		}
	}
}

func (l *LFU) SetRemover(remover strategy.EntryRemover) {
	l.remover = remover
}

func (l *LFU) removeElement(ele *list.Element) {
	kv := ele.Value.(*entry)
	delete(l.cache, kv.key)
	l.frequency[kv.freq].Remove(ele)
	if l.frequency[kv.freq].Len() == 0 {
		delete(l.frequency, kv.freq)
	}
	if l.remover != nil {
		l.remover.OnEntryRemoved(kv.key, kv.value)
	}
}

func (l *LFU) addElement(kv *entry) *list.Element {
	if l.frequency[kv.freq] == nil {
		l.frequency[kv.freq] = list.New()
	}
	return l.frequency[kv.freq].PushFront(kv)
}
