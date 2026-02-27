package lsm

import (
	"encoding/binary"
	"io"
	"os"
)

type SSTable struct {
	file   *os.File
	filter *BloomFilter
	index  []IndexEntry
}

type IndexEntry struct {
	Key    string
	Offset int64
}

type Entry struct {
	Key   string
	Value string
}

type SSTableIterator struct {
	file *os.File
	done bool
}

func NewSSTable(filename string) (*SSTable, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	filterPath := filename + ".filter"
	var filter *BloomFilter
	if _, err := os.Stat(filterPath); err == nil {
		filter, _ = LoadBloomFilter(filterPath)
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	var index []IndexEntry
	if stat.Size() >= 8 {
		if _, err := file.Seek(-8, io.SeekEnd); err == nil {
			var indexOffset uint64
			if err := binary.Read(file, binary.LittleEndian, &indexOffset); err == nil {
				if int64(indexOffset) < stat.Size()-8 {
					if _, err := file.Seek(int64(indexOffset), io.SeekStart); err == nil {
						index, _ = readIndex(file)
					}
				}
			}
		}
	}

	return &SSTable{file: file, filter: filter, index: index}, nil
}

func readIndex(f *os.File) ([]IndexEntry, error) {
	var count uint32
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return nil, err
	}

	index := make([]IndexEntry, count)
	for i := 0; i < int(count); i++ {
		var keyLen uint32
		if err := binary.Read(f, binary.LittleEndian, &keyLen); err != nil {
			return nil, err
		}
		keyBytes := make([]byte, keyLen)
		if _, err := io.ReadFull(f, keyBytes); err != nil {
			return nil, err
		}
		var offset uint64
		if err := binary.Read(f, binary.LittleEndian, &offset); err != nil {
			return nil, err
		}
		index[i] = IndexEntry{Key: string(keyBytes), Offset: int64(offset)}
	}
	return index, nil
}

func NewSSTableIterator(filename string) (*SSTableIterator, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return &SSTableIterator{file: file}, nil
}

func (t *SSTable) Get(key string) (string, bool) {
	if t.filter != nil && !t.filter.Check(key) {
		return "", false
	}

	startOffset := int64(0)
	indexLen := len(t.index)

	if indexLen > 0 {
		low, high := 0, indexLen-1
		idx := -1

		for low <= high {
			mid := low + (high-low)/2
			if t.index[mid].Key <= key {
				idx = mid
				low = mid + 1
			} else {
				high = mid - 1
			}
		}

		if idx >= 0 {
			startOffset = t.index[idx].Offset
		}
	}

	if _, err := t.file.Seek(startOffset, io.SeekStart); err != nil {
		return "", false
	}

	for {
		var keyLen uint32
		err := binary.Read(t.file, binary.LittleEndian, &keyLen)
		if err == io.EOF {
			return "", false
		}

		if err != nil || keyLen == 0 || keyLen > 1024*1024 {
			return "", false
		}

		keyBytes := make([]byte, keyLen)
		if _, err := io.ReadFull(t.file, keyBytes); err != nil {
			return "", false
		}

		currentKey := string(keyBytes)
		var valLen uint32
		if err := binary.Read(t.file, binary.LittleEndian, &valLen); err != nil {
			return "", false
		}

		if currentKey == key {
			valBytes := make([]byte, valLen)
			if _, err := io.ReadFull(t.file, valBytes); err != nil {
				return "", false
			}
			return string(valBytes), true
		} else {
			if currentKey > key {
				return "", false
			}
			if _, err := t.file.Seek(int64(valLen), io.SeekCurrent); err != nil {
				return "", false
			}
		}
	}
}

func (t *SSTable) Close() error {
	return t.file.Close()
}

func WriteSSTable(filename string, keys []string, data map[string]string) error {
	filter := NewBloomFilter(len(keys), 0.01)
	for _, key := range keys {
		filter.Add(key)
	}

	if err := SaveBloomFilter(filename+".filter", filter); err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var index []IndexEntry
	var currentOffset int64 = 0
	const indexInterval = 100 // Record an index entry every 100 keys

	for i, key := range keys {
		if i%indexInterval == 0 {
			index = append(index, IndexEntry{Key: key, Offset: currentOffset})
		}

		val := data[key]
		written, err := writeEntry(file, key, val)
		if err != nil {
			return err
		}
		currentOffset += int64(written)
	}

	indexOffset := currentOffset
	if err := writeIndex(file, index); err != nil {
		return err
	}

	if err := binary.Write(file, binary.LittleEndian, uint64(indexOffset)); err != nil {
		return err
	}

	return nil
}

func writeIndex(f *os.File, index []IndexEntry) error {
	if err := binary.Write(f, binary.LittleEndian, uint32(len(index))); err != nil {
		return err
	}
	for _, entry := range index {
		if err := binary.Write(f, binary.LittleEndian, uint32(len(entry.Key))); err != nil {
			return err
		}
		if _, err := f.Write([]byte(entry.Key)); err != nil {
			return err
		}
		if err := binary.Write(f, binary.LittleEndian, uint64(entry.Offset)); err != nil {
			return err
		}
	}
	return nil
}

func (t *SSTable) Keys() ([]string, error) {
	if _, err := t.file.Seek(0, 0); err != nil {
		return nil, err
	}

	var keys []string
	for {
		var keyLen uint32
		err := binary.Read(t.file, binary.LittleEndian, &keyLen)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		keyBytes := make([]byte, keyLen)
		if _, err := io.ReadFull(t.file, keyBytes); err != nil {
			return nil, err
		}
		keys = append(keys, string(keyBytes))

		var valLen uint32
		if err := binary.Read(t.file, binary.LittleEndian, &valLen); err != nil {
			return nil, err
		}

		if _, err := t.file.Seek(int64(valLen), io.SeekCurrent); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

func (it *SSTableIterator) Next() *Entry {
	if it.done {
		return nil
	}

	var keyLen uint32
	if err := binary.Read(it.file, binary.LittleEndian, &keyLen); err != nil {
		it.done = true
		return nil
	}

	keyBytes := make([]byte, keyLen)
	if _, err := io.ReadFull(it.file, keyBytes); err != nil {
		it.done = true
		return nil
	}

	var valLen uint32
	if err := binary.Read(it.file, binary.LittleEndian, &valLen); err != nil {
		it.done = true
		return nil
	}

	valBytes := make([]byte, valLen)
	if _, err := io.ReadFull(it.file, valBytes); err != nil {
		it.done = true
		return nil
	}

	return &Entry{
		Key:   string(keyBytes),
		Value: string(valBytes),
	}
}

func (it *SSTableIterator) Close() error {
	return it.file.Close()
}

func writeEntry(f *os.File, key, value string) (int, error) {
	written := 0
	if err := binary.Write(f, binary.LittleEndian, uint32(len(key))); err != nil {
		return written, err
	}
	written += 4

	n, err := f.Write([]byte(key))
	if err != nil {
		return written, err
	}
	written += n

	if err := binary.Write(f, binary.LittleEndian, uint32(len(value))); err != nil {
		return written, err
	}
	written += 4

	n, err = f.Write([]byte(value))
	if err != nil {
		return written, err
	}
	written += n

	return written, nil
}
