package lsm

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const syncInterval = 10 * time.Millisecond

type WAL struct {
	file   *os.File
	mu     sync.Mutex
	stopCh chan struct{}
	dirty  bool
}

func NewWAL(filename string) (*WAL, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	w := &WAL{
		file:   file,
		stopCh: make(chan struct{}),
	}

	go w.syncLoop()

	return w, nil
}

func (w *WAL) syncLoop() {
	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.mu.Lock()
			if w.dirty {
				w.file.Sync()
				w.dirty = false
			}
			w.mu.Unlock()
		case <-w.stopCh:
			return
		}
	}
}

func (w *WAL) Write(op, key, value string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var entry string
	if op == "DELETE" {
		entry = fmt.Sprintf("DELETE|%s\n", key)
	} else {
		entry = fmt.Sprintf("PUT|%s|%s\n", key, value)
	}

	if _, err := w.file.WriteString(entry); err != nil {
		return err
	}

	w.dirty = true
	return nil
}

func (w *WAL) Recover(memTable *MemTable) error {
	if _, err := w.file.Seek(0, 0); err != nil {
		return err
	}

	scanner := bufio.NewScanner(w.file)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			continue
		}

		op := parts[0]
		key := parts[1]

		switch op {
		case "PUT":
			if len(parts) == 3 {
				memTable.Put(key, parts[2])
				count++
			}
		case "DELETE":
			memTable.Put(key, "__deleted__")
			count++
		}
	}

	if count > 0 {
		fmt.Printf("WAL Recovery: Replayed %d operations\n", count)
	}

	w.file.Seek(0, 2)
	return scanner.Err()
}

func (w *WAL) Reset() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.dirty {
		w.file.Sync()
		w.dirty = false
	}

	if err := w.file.Truncate(0); err != nil {
		return err
	}

	_, err := w.file.Seek(0, 0)
	return err
}

func (w *WAL) Close() error {
	close(w.stopCh)
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.dirty {
		w.file.Sync()
	}
	return w.file.Close()
}
