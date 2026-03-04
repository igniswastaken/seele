package service

import (
	"sync"
	"time"

	"github.com/zerothy/seele/lsm"
)

type Store struct {
	engine     *lsm.LSMTree
	merkle     *MerkleTree
	merkleLock sync.RWMutex
	isDirty    bool
	cas        sync.Mutex
}

const tombstone = "__deleted__"
const memtableLimit = 1024 * 1024

func NewStore(dir string) (*Store, error) {
	engine, err := lsm.NewLSMTree(dir, memtableLimit)
	if err != nil {
		return nil, err
	}

	store := &Store{engine: engine, isDirty: true}

	go store.merkleWorker()

	return store, nil
}

func (s *Store) Set(key, value string) error {
	err := s.engine.Put(key, value)
	if err == nil {
		s.markDirty()
	}
	return err
}

func (s *Store) Get(key string) (string, bool) {
	return s.engine.Get(key)
}

func (s *Store) Delete(key string) error {
	err := s.engine.Delete(key)
	if err == nil {
		s.markDirty()
	}
	return err
}

func (s *Store) SetIfNotExists(key, value string) (bool, error) {
	s.cas.Lock()
	defer s.cas.Unlock()

	if _, exists := s.engine.Get(key); exists {
		return false, nil
	}
	err := s.engine.Put(key, value)
	if err == nil {
		s.markDirty()
	}
	return err == nil, err
}

func (s *Store) SetIfExists(key, value string) (bool, error) {
	s.cas.Lock()
	defer s.cas.Unlock()

	if _, exists := s.engine.Get(key); !exists {
		return false, nil
	}
	err := s.engine.Put(key, value)
	if err == nil {
		s.markDirty()
	}
	return err == nil, err
}

func (s *Store) Keys() []string {
	return s.engine.Keys()
}

func (s *Store) Close() error {
	return s.engine.Close()
}

func (s *Store) markDirty() {
	s.merkleLock.Lock()
	s.isDirty = true
	s.merkleLock.Unlock()
}

func (s *Store) merkleWorker() {
	for {
		time.Sleep(3 * time.Second)

		s.merkleLock.RLock()
		needsRebuild := s.isDirty
		s.merkleLock.RUnlock()

		if needsRebuild {
			s.rebuildMerkle()
		}
	}
}

func (s *Store) rebuildMerkle() {
	keys := s.engine.Keys()

	newTree := NewMerkleTree(keys)

	s.merkleLock.Lock()
	s.merkle = newTree
	s.isDirty = false
	s.merkleLock.Unlock()
}

func (s *Store) GetMerkleRoot() string {
	s.merkleLock.RLock()
	defer s.merkleLock.RUnlock()

	if s.merkle == nil || s.merkle.Root == nil {
		return ""
	}
	return s.merkle.Root.Hash
}
