package stellar

import (
	"sync"
	"time"

	"github.com/switchlyprotocol/switchlynode/v3/common"
)

type StellarMetadata struct {
	SeqNumber   int64
	BlockHeight int64
	LastSync    time.Time
}

type StellarMetaDataStore struct {
	lock  *sync.Mutex
	accts map[common.PubKey]StellarMetadata
}

func NewStellarMetaDataStore() *StellarMetaDataStore {
	return &StellarMetaDataStore{
		lock:  &sync.Mutex{},
		accts: make(map[common.PubKey]StellarMetadata),
	}
}

func (s *StellarMetaDataStore) Get(pk common.PubKey) StellarMetadata {
	s.lock.Lock()
	defer s.lock.Unlock()
	if val, ok := s.accts[pk]; ok {
		return val
	}
	return StellarMetadata{}
}

func (s *StellarMetaDataStore) Set(pk common.PubKey, meta StellarMetadata) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.accts[pk] = meta
}

// GetCurrentSequence returns the current tracked sequence without incrementing
func (s *StellarMetaDataStore) GetCurrentSequence(pk common.PubKey) int64 {
	s.lock.Lock()
	defer s.lock.Unlock()
	if meta, ok := s.accts[pk]; ok {
		return meta.SeqNumber
	}
	return 0
}

// GetNextSequence returns the current sequence without incrementing (tx builder increments)
func (s *StellarMetaDataStore) GetNextSequence(pk common.PubKey) int64 {
	s.lock.Lock()
	defer s.lock.Unlock()
	meta, exists := s.accts[pk]
	if !exists {
		return 0
	}
	return meta.SeqNumber
}

// SetBaseSequence sets the initial sequence from chain (only used during sync)
func (s *StellarMetaDataStore) SetBaseSequence(pk common.PubKey, chainSeq int64, blockHeight int64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	meta := StellarMetadata{
		SeqNumber:   chainSeq,
		BlockHeight: blockHeight,
		LastSync:    time.Now(),
	}
	s.accts[pk] = meta
}

// IsStale checks if the metadata needs refreshing from chain
func (s *StellarMetaDataStore) IsStale(pk common.PubKey, currentHeight int64) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	meta, exists := s.accts[pk]
	if !exists {
		return true // No data = stale
	}

	// Consider stale if block height advanced significantly or too much time passed
	heightDiff := currentHeight - meta.BlockHeight
	timeDiff := time.Since(meta.LastSync)

	return heightDiff > 10 || timeDiff > 5*time.Minute
}

// AdvanceSequence increments the stored sequence by one after a successful broadcast
func (s *StellarMetaDataStore) AdvanceSequence(pk common.PubKey) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if meta, ok := s.accts[pk]; ok {
		meta.SeqNumber++
		meta.LastSync = time.Now()
		s.accts[pk] = meta
	}
}
