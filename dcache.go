package dcache

import (
	"dcache/cachepb"
	"dcache/singleflight"
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
	pickers   PeerPicker

	loader *singleflight.Group
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
		loader:    &singleflight.Group{},
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
	// 增加保护机制
	b, err := g.loader.Do(key, func() (i interface{}, err error) {
		// 如果没有注册peer，还是调用本地缓存

		if g.pickers != nil {
			if peer, ok := g.pickers.PickPeer(key); ok {
				if value, err := g.getFromPeer(peer, key); err == nil {
					return value, err
				}
				log.Println("[cache] Failed to get from peer")
			}
		}
		return g.getLocally(key)
	})

	if err != nil {
		return ByteView{}, err
	}
	return b.(ByteView), nil
}

func (g *Group) getFromPeer(getter PeerGetter, key string) (ByteView, error) {
	//bytes, err := getter.Get(g.name, key)
	//if err != nil {
	//	return ByteView{}, err
	//}
	resp := &cachepb.Response{}
	err := getter.Get(&cachepb.Request{
		Group: g.name,
		Key:   key,
	}, resp)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: resp.Value}, nil
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

func (g *Group) RegisterPeers(picker PeerPicker) {
	if g.pickers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.pickers = picker
}
