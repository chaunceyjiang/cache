package lru

import (
	"testing"
)

type String string

func (s String) Len() int {
	return len(s)
}
func TestCache_Add(t *testing.T) {
	lru := New(0, nil)
	lru.Add("key1", String("value1"))
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "value1" {
		t.Fatal("cache get key1 error")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatal("cache miss key2 failed")
	}
}

func TestCache_Remove(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value", "v2", "value3"
	total := len(k1 + k2 + v1 + v2)
	lru := New(int64(total), nil)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3)) // 超过最大容量,k2比k1更新，因此这里会删除k1

	if _, ok := lru.Get(k1); ok || lru.Len() != 2 {
		t.Fatal("Remove key1 failed")
	}
}
func TestCache_OnEvicted(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value", "v2", "value3"
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}

	total := len(k1 + k2 + v1 + v2)
	lru := New(int64(total), callback)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3)) // 超过最大容量,k2比k1更新，因此这里会删除k1

	if keys[0] != "key1" {
		t.Fatal("callback failed")
	}
}
