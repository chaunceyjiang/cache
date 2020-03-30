package dcache

type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	// 用于从对应的group中查找缓存值
	Get(group string, key string) ([]byte, error)
}
