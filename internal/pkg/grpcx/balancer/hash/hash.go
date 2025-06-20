package hash

import (
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

type HashRing struct {
	mu          sync.RWMutex
	nodes       []string          // 所有节点地址
	virtualNode int               // 每个节点的虚拟节点数
	ring        map[uint32]string // 哈希环：哈希值 -> 节点
	sortedKeys  []uint32          // 排序后的哈希环键
}

func NewHashRing(virtualNode int) *HashRing {
	return &HashRing{
		virtualNode: virtualNode,
		ring:        make(map[uint32]string),
		sortedKeys:  make([]uint32, 0),
	}
}

// AddNode 添加节点到哈希环
func (h *HashRing) AddNode(node string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 添加真实的节点
	h.nodes = append(h.nodes, node)

	// 添加虚拟节点
	for i := 0; i < h.virtualNode; i++ {
		virtualKey := node + "#" + strconv.Itoa(i)
		hash := h.hashKey(virtualKey)
		h.ring[hash] = node
		h.sortedKeys = append(h.sortedKeys, hash)
	}

	// 排序哈希键
	sort.Slice(h.sortedKeys, func(i, j int) bool {
		return h.sortedKeys[i] < h.sortedKeys[j]
	})
}

func (h *HashRing) RemoveNode(node string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 移除真实节点
	for i, n := range h.nodes {
		if n == node {
			h.nodes = append(h.nodes[:i], h.nodes[i+1:]...)
			break
		}
	}

	// 移除虚拟节点
	newRing := make(map[uint32]string)
	newKeys := make([]uint32, 0)

	for hash, n := range h.ring {
		if n != node {
			newRing[hash] = n
			newKeys = append(newKeys, hash)
		}
	}

	sort.Slice(newKeys, func(i, j int) bool {
		return newKeys[i] < newKeys[j]
	})

	h.ring = newRing
	h.sortedKeys = newKeys
}

// GetNode 根据key获取节点
func (h *HashRing) GetNode(key string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.sortedKeys) == 0 {
		return ""
	}

	hash := h.hashKey(key)

	idx := sort.Search(len(h.sortedKeys), func(i int) bool {
		return h.sortedKeys[i] >= hash
	})
	if idx == len(h.sortedKeys) {
		idx = 0
	}

	return h.ring[h.sortedKeys[idx]]
}

func (h *HashRing) NodeCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sortedKeys)
}

// hashKey 计算key的哈希值
func (h *HashRing) hashKey(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}
