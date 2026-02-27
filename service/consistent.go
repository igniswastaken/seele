package service

import (
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

type HashRing struct {
	mu       sync.RWMutex
	replicas int
	keys     []int
	hashMap  map[uint32]string
}

func NewHashRing(replicas int) *HashRing {
	return &HashRing{
		replicas: replicas,
		hashMap:  make(map[uint32]string),
	}
}

func (h *HashRing) AddNode(node string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i := 0; i < h.replicas; i++ {
		virtualKey := node + "#" + strconv.Itoa(i)
		hash := crc32.ChecksumIEEE([]byte(virtualKey))

		h.keys = append(h.keys, int(hash))
		h.hashMap[hash] = node
	}

	sort.Slice(h.keys, func(i, j int) bool {
		return h.keys[i] < h.keys[j]
	})
}

func (h *HashRing) SetNodes(nodes []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.keys = []int{}
	h.hashMap = make(map[uint32]string)

	for _, node := range nodes {
		for i := 0; i < h.replicas; i++ {
			virtualKey := node + "#" + strconv.Itoa(i)
			hash := crc32.ChecksumIEEE([]byte(virtualKey))

			h.keys = append(h.keys, int(hash))
			h.hashMap[hash] = node
		}
	}

	sort.Slice(h.keys, func(i, j int) bool {
		return h.keys[i] < h.keys[j]
	})
}

func (h *HashRing) GetNode(key string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.keys) == 0 {
		return ""
	}

	hash := crc32.ChecksumIEEE([]byte(key))

	idx := sort.Search(len(h.keys), func(i int) bool {
		return h.keys[i] >= int(hash)
	})

	if idx == len(h.keys) {
		idx = 0
	}

	return h.hashMap[uint32(h.keys[idx])]
}

func (h *HashRing) GetNodes(key string, count int) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.keys) == 0 {
		return nil
	}

	hash := crc32.ChecksumIEEE([]byte(key))
	idx := sort.Search(len(h.keys), func(i int) bool {
		return h.keys[i] >= int(hash)
	})

	if idx == len(h.keys) {
		idx = 0
	}

	var nodes []string
	seen := make(map[string]bool)

	for i := 0; i < len(h.keys); i++ {
		curIdx := (idx + i) % len(h.keys)
		node := h.hashMap[uint32(h.keys[curIdx])]
		if !seen[node] {
			seen[node] = true
			nodes = append(nodes, node)
			if len(nodes) == count {
				break
			}
		}
	}

	return nodes
}
