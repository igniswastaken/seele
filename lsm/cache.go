package lsm

import "sync"

type cacheNode struct {
	key   string
	value string
	prev  *cacheNode
	next  *cacheNode
}

type LRUCache struct {
	capacity int
	cache    map[string]*cacheNode
	head     *cacheNode
	tail     *cacheNode
	mu       sync.Mutex
}

func NewLRUCache(capacity int) *LRUCache {
	head := &cacheNode{}
	tail := &cacheNode{}
	head.next = tail
	tail.prev = head

	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*cacheNode),
		head:     head,
		tail:     tail,
	}
}

func (c *LRUCache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.cache[key]; ok {
		c.moveToFront(node)
		return node.value, true
	}
	return "", false
}

func (c *LRUCache) Put(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.cache[key]; ok {
		node.value = value
		c.moveToFront(node)
		return
	}

	node := &cacheNode{
		key:   key,
		value: value,
	}
	c.cache[key] = node
	c.addToFront(node)

	if len(c.cache) > c.capacity {
		c.removeLRU()
	}
}

func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if node, ok := c.cache[key]; ok {
		c.removeNode(node)
		delete(c.cache, key)
	}
}

func (c *LRUCache) moveToFront(node *cacheNode) {
	c.removeNode(node)
	c.addToFront(node)
}

func (c *LRUCache) addToFront(node *cacheNode) {
	node.prev = c.head
	node.next = c.head.next
	c.head.next.prev = node
	c.head.next = node
}

func (c *LRUCache) removeNode(node *cacheNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (c *LRUCache) removeLRU() {
	lru := c.tail.prev
	if lru == c.head {
		return
	}
	c.removeNode(lru)
	delete(c.cache, lru.key)
}
