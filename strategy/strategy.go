package strategy

import (
	"time"
)

type EntryRemover interface {
	OnEntryRemoved(key string, value Value)
}

type EvictionStrategy interface {
	Get(key string) (Value, bool)
	RemoveOldest()
	Add(key string, value Value, expire time.Time)
	SetRemover(remover EntryRemover)
}

type Value interface {
	Len() int
}

type entry struct {
	key    string
	value  Value
	expire time.Time
}
