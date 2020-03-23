package dcache

type ByteView struct {
	b []byte // 存储真实的缓存值
}

// Len 实现Value 接口
func (b ByteView) Len() int {
	return len(b.b)
}

// 拷贝原生数据，防止原始数据被修改
func (b ByteView) ByteSlice() []byte {
	c := make([]byte, len(b.b))
	copy(c, b.b)
	return c
}

