package lsm

import (
	"container/heap"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type heapItem struct {
	entry       *Entry
	sourceIndex int
	iterator    *SSTableIterator
}

type mergeHeap []*heapItem

func (h mergeHeap) Len() int {
	return len(h)
}

func (h mergeHeap) Less(i, j int) bool {
	if h[i].entry.Key == h[j].entry.Key {
		return h[i].sourceIndex < h[j].sourceIndex
	}
	return h[i].entry.Key < h[j].entry.Key
}

func (h mergeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *mergeHeap) Push(x any) {
	*h = append(*h, x.(*heapItem))
}

func (h *mergeHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func (l *LSMTree) Compact() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.ssTable) < 2 {
		fmt.Println("Compaction: Nothing to compact (< 2 SSTables)")
		return nil
	}

	fmt.Printf("Compaction: Merging %d SSTables...\n", len(l.ssTable))

	h := &mergeHeap{}
	heap.Init(h)

	iterators := make([]*SSTableIterator, 0, len(l.ssTable))
	for i, table := range l.ssTable {
		it, err := NewSSTableIterator(table.file.Name())
		if err != nil {
			fmt.Println("Error opening SSTable iterator:", err)
			continue
		}
		iterators = append(iterators, it)

		if entry := it.Next(); entry != nil {
			heap.Push(h, &heapItem{
				entry:       entry,
				sourceIndex: i,
				iterator:    it,
			})
		}
	}

	newFilename := filepath.Join(l.dir, fmt.Sprintf("sstable-%d.bin", time.Now().UnixNano()))
	outFile, err := os.Create(newFilename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	totalEstimatedKeys := 10000 // Default to a reasonable size if unknown
	filter := NewBloomFilter(totalEstimatedKeys, 0.01)

	var lastKey string
	activeKeys := 0

	var index []IndexEntry
	var currentOffset int64 = 0
	const indexInterval = 100

	for h.Len() > 0 {
		item := heap.Pop(h).(*heapItem)

		if next := item.iterator.Next(); next != nil {
			heap.Push(h, &heapItem{
				entry:       next,
				sourceIndex: item.sourceIndex,
				iterator:    item.iterator,
			})
		}

		if item.entry.Key == lastKey {
			continue
		}
		lastKey = item.entry.Key

		if item.entry.Value == "__deleted__" {
			continue
		}

		filter.Add(item.entry.Key)

		if activeKeys%indexInterval == 0 {
			index = append(index, IndexEntry{Key: item.entry.Key, Offset: currentOffset})
		}

		written, err := writeEntry(outFile, item.entry.Key, item.entry.Value)
		if err != nil {
			return err
		}
		currentOffset += int64(written)
		activeKeys++
	}

	indexOffset := currentOffset
	if err := writeIndex(outFile, index); err != nil {
		return err
	}
	if err := binary.Write(outFile, binary.LittleEndian, uint64(indexOffset)); err != nil {
		return err
	}

	if err := SaveBloomFilter(newFilename+".filter", filter); err != nil {
		return err
	}

	for _, it := range iterators {
		it.Close()
	}

	for _, oldTable := range l.ssTable {
		oldFile := oldTable.file.Name()
		oldTable.Close()
		if err := os.Remove(oldFile); err != nil {
			fmt.Println("Compaction: Failed to delete", oldFile, err)
		}
		os.Remove(oldFile + ".filter")
	}

	newTable, err := NewSSTable(newFilename)
	if err != nil {
		return err
	}

	l.ssTable = []*SSTable{newTable}

	fmt.Printf("Compaction: Done! %d active keys in %s\n", activeKeys, newFilename)
	return nil
}
