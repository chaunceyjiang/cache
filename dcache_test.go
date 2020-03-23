package dcache

import (
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
