package singleflight

import "sync"

/*
  原理：

在相同key 的一个请求过程中，其他相同请求的key服用同一个
*/

type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		// 因为并发的缘故，可能会存在第一次请求还没有结束，就发起了第二次请求，因此，这里对一个请求期间的相同key的请求，复用相同结果
		g.mu.Lock()
		c.wg.Wait() // 这里等待第一次的请求完成
		return c.val, c.err
	}

	c := new(call)

	c.wg.Add(1)
	g.m[key] = c  // 将这个请求保存
	g.mu.Unlock() // 快速释放锁，减少Do 阻塞的时间

	c.val, c.err = fn() // 调用回调函数
	c.wg.Done()         // 获取到调用结束后，立即释放

	g.mu.Lock()
	delete(g.m, key) // 调用结束，释放这个请求的标记
	g.mu.Unlock()

	return c.val, c.err
}
