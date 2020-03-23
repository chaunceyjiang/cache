package dcache

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
