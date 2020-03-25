package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

// Map 包含了所有的hash key
type Map struct {
	hash     Hash
	replicas int // 虚拟节点的倍数  （虚拟节点，是为就解决数据倾斜）
	keys     []int

	hashMap map[int]string // 虚拟节点与真实节点的映射
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		keys:     nil,
		hashMap:  make(map[int]string),
	}

	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE //crc 循环校验码,Range: 0 through 4294967295
	}

	return m
}

// Add 在hash环上添加节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			// 创建虚拟节点
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)

			//  映射关系
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys) // 打乱节点顺序
}

// Get 根据key 选择节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	// 计算key的hash 值
	hash := int(m.hash([]byte(key)))

	idx := sort.Search(len(m.keys), func(i int) bool {
		// 找到第一个比他大的虚拟节点
		return m.keys[i] >= hash
	})
	// 获取真实节点
	return m.hashMap[m.keys[idx%(len(m.keys))]]
}
