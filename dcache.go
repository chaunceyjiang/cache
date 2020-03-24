package dcache

import (
	"errors"
	"log"
	"sync"
)

/*
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
*/

// Getter  定义一个通用获取源数据接口
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 定义了一个类型
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	// 调用该函数
	// 因此只要是 GetterFunc 类型，自动实现了 Getter 接口
	return f(key)
}

// Group 缓存的命名空间
type Group struct {
	name      string // 缓存的名字
	getter    Getter // 获取数据的回调函数
	mainCache cache  // 缓存
}

var (
	mu     sync.RWMutex              // 一个读写锁
	groups = make(map[string]*Group) // 全局缓存所有的Group
)

// NewGroup新建一个新的Group，然后放入全局缓存中
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		// 获取源数据的回调函数不能为空
		panic("nil getter")
	}

	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// GetGroup 获取一个指定的Group
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get 从缓存中获取数据
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, errors.New("key is required ")
	}
	value, ok := g.mainCache.get(key)
	//if !ok {
	//	b, err := g.getter.Get(key)
	//	if err != nil {
	//		return ByteView{}, err
	//	}
	//	value := ByteView{b: b}
	//	g.mainCache.add(key, value)
	//}

	if ok {
		log.Println("[Cache] hit")
		return value, nil
	}
	return g.load(key)
}

// load 加载数据 分别从本地，和远程加载数据
func (g *Group) load(key string) (ByteView, error) {
	return g.getLocally(key)
}

// getLocally 从本地获取数据
func (g *Group) getLocally(key string) (ByteView, error) {
	b, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	// 拷贝原始数据
	value := ByteView{b: cloeBytes(b)}
	// 缓存从源数据中获取的数据
	g.mainCache.add(key, value)
	return value, nil
}

func cloeBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
