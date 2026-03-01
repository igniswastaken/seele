package lsm

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

type LSMTree struct {
	dir             string
	memTable        *MemTable
	wal             *WAL
	ssTable         []*SSTable
	ssTableMu       sync.RWMutex
	cache           *LRUCache
	mu              sync.RWMutex
	maxMemTableSize int
	flushCh         chan *MemTable
}

func NewLSMTree(dir string, maxMemTableSize int) (*LSMTree, error) {
	wslPath := filepath.Join(dir, "seele.wal")
	wal, err := NewWAL(wslPath)
	if err != nil {
		return nil, err
	}

	memTable := NewMemTable()

	if err := wal.Recover(memTable); err != nil {
		return nil, err
	}

	files, err := filepath.Glob(filepath.Join(dir, "sstable-*.bin"))
	if err != nil {
		return nil, err
	}

	for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
		files[i], files[j] = files[j], files[i]
	}

	tables := make([]*SSTable, 0)
	for _, f := range files {
		t, err := NewSSTable(f)
		if err == nil {
			tables = append(tables, t)
		} else {
			fmt.Println("Error opening SSTable on boot:", err)
		}
	}

	tree := &LSMTree{
		dir:             dir,
		memTable:        memTable,
		wal:             wal,
		maxMemTableSize: maxMemTableSize,
		ssTable:         tables,
		cache:           NewLRUCache(10000),
		flushCh:         make(chan *MemTable, 4),
	}

	go tree.flushWorker()

	return tree, nil
}

func (l *LSMTree) flushWorker() {
	for immutable := range l.flushCh {
		if err := l.flushToDisk(immutable); err != nil {
			fmt.Println("LSM flush error:", err)
		}
	}
}

func (l *LSMTree) Put(key, value string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.wal.Write("PUT", key, value); err != nil {
		return err
	}

	l.memTable.Put(key, value)

	if l.memTable.Size() >= l.maxMemTableSize {
		oldTable := l.memTable
		l.memTable = NewMemTable()
		l.wal.Reset()

		l.mu.Unlock()
		select {
		case l.flushCh <- oldTable:
		default:
			l.flushCh <- oldTable
		}
		l.mu.Lock()
	}
	return nil
}

func (l *LSMTree) flushToDisk(mem *MemTable) error {
	filename := filepath.Join(l.dir, fmt.Sprintf("sstable-%d.bin", time.Now().UnixNano()))

	fmt.Printf("Background Flush: Writing %d bytes to %s...\n", mem.Size(), filename)

	keys := mem.Keys()
	if err := WriteSSTable(filename, keys, mem.data); err != nil {
		return err
	}

	newTable, err := NewSSTable(filename)
	if err != nil {
		return err
	}

	l.ssTableMu.Lock()
	l.ssTable = append([]*SSTable{newTable}, l.ssTable...)
	l.ssTableMu.Unlock()

	return nil
}

func (l *LSMTree) Get(key string) (string, bool) {
	l.mu.RLock()
	val, exist := l.memTable.Get(key)
	if exist {
		l.mu.RUnlock()
		if val == "__deleted__" {
			return "", false
		}
		return val, true
	}
	l.mu.RUnlock()

	if cachedVal, found := l.cache.Get(key); found {
		if cachedVal == "__deleted__" {
			return "", false
		}
		return cachedVal, true
	}

	l.ssTableMu.RLock()
	defer l.ssTableMu.RUnlock()

	for _, table := range l.ssTable {
		val, exist := table.Get(key)

		if exist {
			l.cache.Put(key, val)
			if val == "__deleted__" {
				return "", false
			}
			return val, true
		}
	}
	return "", false
}

func (l *LSMTree) Delete(key string) error {
	return l.Put(key, "__deleted__")
}

func (l *LSMTree) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.wal.Close(); err != nil {
		fmt.Println("Error closing WAL:", err)
	}

	l.ssTableMu.Lock()
	defer l.ssTableMu.Unlock()

	for _, table := range l.ssTable {
		if err := table.Close(); err != nil {
			fmt.Println("Error closing SSTable:", err)
		}
	}

	return nil
}

func (l *LSMTree) Keys() []string {
	l.mu.RLock()
	keyMap := make(map[string]bool)
	for _, k := range l.memTable.Keys() {
		val, _ := l.memTable.Get(k)
		if val == "__deleted__" {
			keyMap[k] = false
		} else {
			keyMap[k] = true
		}
	}
	l.mu.RUnlock()

	l.ssTableMu.RLock()
	defer l.ssTableMu.RUnlock()

	for _, table := range l.ssTable {
		k, _ := table.Keys()
		for _, key := range k {
			if _, seen := keyMap[key]; !seen {
				val, found := table.Get(key)
				if found {
					if val == "__deleted__" {
						keyMap[key] = false
					} else {
						keyMap[key] = true
					}
				}
			}
		}
	}

	var activeKeys []string
	for k, active := range keyMap {
		if active {
			activeKeys = append(activeKeys, k)
		}
	}
	return activeKeys
}

func (l *LSMTree) StartCompaction(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := l.Compact(); err != nil {
				fmt.Println("Compaction error:", err)
			}
		}
	}()
}
