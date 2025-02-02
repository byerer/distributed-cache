package lfu

import (
	"testing"
	"time"
)

type value struct {
	val string
}

func (v *value) Len() int {
	return len(v.val)
}

func TestLFU_Basic(t *testing.T) {
	lfu := New(nil)
	lfu.Add("key1", &value{"value1"}, time.Time{})
	if _, ok := lfu.Get("key1"); !ok {
		t.Fatalf("lfu hit key1=value1 failed")
	}
	if _, ok := lfu.Get("key2"); ok {
		t.Fatalf("lfu miss key2 failed")
	}
}
