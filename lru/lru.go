package lru

import "container/list"

// Cache LRU cache
type Cache struct {
	maxBytes int64      // 允许的最大内存
	curBytes int64      //当前使用内存
	ll       *list.List // 双向链表 链表中存储entry

	cache map[string]*list.Element // 保存每个节点的地址，方便直接访问

	OnEvicted func(key string, value Value) // 某条记录被移除时的回调函数
}

// Value 存储类型
type Value interface {
	Len() int
}

type entry struct {
	key   string
	value Value
}

// New maxBytes 允许的最大值 onEvicted 某个记录被删除时的回调函数
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		curBytes:  0,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get 从map中查询对应节点，将该节点移至队首
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// 将最新访问的放在队首
		c.ll.MoveToFront(ele)
		return ele.Value.(*entry).value, ok
	}
	return
}

func (c *Cache) Remove() {
	// 获取队尾元素
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		e := ele.Value.(*entry)
		delete(c.cache, e.key)
		// 删除后，当前存储字节数也相应减少
		c.curBytes -= int64(len(e.key)) + int64(e.value.Len())

		// 如果注册了回调函数，则处理回调
		if c.OnEvicted != nil {
			c.OnEvicted(e.key, e.value)
		}
	}
}

// Add 添加记录
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		// 存在相同的key ，则直接更新值
		e := ele.Value.(*entry)
		// 同时将这个元素移至队首
		c.ll.MoveToFront(ele)
		// 更新字节大小
		c.curBytes += int64(value.Len()) - int64(e.value.Len())
		// 更新值
		e.value = value

	} else {
		e := &entry{
			key:   key,
			value: value,
		}
		// 最新元素
		c.cache[key] = c.ll.PushFront(e)
		// 更新存储大小
		c.curBytes += int64(len(key)) + int64(value.Len())
	}

	if c.maxBytes != 0 && c.curBytes > c.maxBytes {
		// 超过了最大内存设置，移除
		c.Remove()
	}
}

func (c *Cache) Len() int {
	if c.ll.Len() != len(c.cache) {
		panic("map与list大小不一致")
	}
	return c.ll.Len()
}
