package dcache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

func TestGetterFunc_Get(t *testing.T) {
	var f Getter
	// 1. 将一个匿名函数转化为一个 GetterFunc 的函数
	// 2. 又因为GetterFunc实现了 Getter 接口，因此将 GetterFunc 赋值给f
	f = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, []byte("key")) {
		t.Fatal("callback failed")
	}
}

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGroup_Get(t *testing.T) {
	// 统计次数
	loadCounts := make(map[string]int, len(db))

	gc := NewGroup("sources", 2<<10, GetterFunc(func(key string) ([]byte, error) {
		log.Println("[cache] search key", key)
		if v, ok := db[key]; ok {
			loadCounts[key]++
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s not exist", key)
	}))

	for k, v := range db {
		// 第一次访问 从回调函数中获取
		if view, err := gc.Get(k); err != nil || view.String() != v {
			t.Fatalf("get %s failed", k)
		}
		// 第二次访问，命中缓存
		if _, err := gc.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}
}
