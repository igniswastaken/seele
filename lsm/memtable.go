package lsm

import (
	"sort"
	"sync"
)

type MemTable struct {
	data map[string]string
	mu   sync.RWMutex
	size int
}

func NewMemTable() *MemTable {
	return &MemTable{
		data: make(map[string]string),
	}
}

func (m *MemTable) Put(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if oldVal, exist := m.data[key]; exist {
		m.size -= len(oldVal)
	} else {
		m.size += len(key)
	}
	m.size += len(value)
	m.data[key] = value
}

func (m *MemTable) Get(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, exist := m.data[key]
	return val, exist
}

func (m *MemTable) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if oldVal, exist := m.data[key]; exist {
		m.size -= len(oldVal) + len(key)
		delete(m.data, key)
	}
}

func (m *MemTable) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.size
}

func (m *MemTable) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (m *MemTable) Flush() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]string)
	m.size = 0
}
