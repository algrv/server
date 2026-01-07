package ccsignals

import (
	"context"
	"sync"
)

// implements FingerprintStore using in-memory storage
type MemoryFingerprintStore struct {
	mu      sync.RWMutex
	records map[string]*FingerprintRecord
	byWork  map[string]string
}

// creates a new in-memory fingerprint store
func NewMemoryFingerprintStore() *MemoryFingerprintStore {
	return &MemoryFingerprintStore{
		records: make(map[string]*FingerprintRecord),
		byWork:  make(map[string]string),
	}
}

// saves a fingerprint record
func (s *MemoryFingerprintStore) Store(_ context.Context, record *FingerprintRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[record.ID] = record
	s.byWork[record.WorkID] = record.ID
	return nil
}

// removes a fingerprint record
func (s *MemoryFingerprintStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record, exists := s.records[id]; exists {
		delete(s.byWork, record.WorkID)
	}
	delete(s.records, id)
	return nil
}

// loads all fingerprint records
func (s *MemoryFingerprintStore) LoadAll(_ context.Context) ([]*FingerprintRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*FingerprintRecord, 0, len(s.records))
	for _, record := range s.records {
		records = append(records, record)
	}
	return records, nil
}

// retrieves fingerprint by work ID
func (s *MemoryFingerprintStore) GetByWorkID(_ context.Context, workID string) (*FingerprintRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if id, exists := s.byWork[workID]; exists {
		return s.records[id], nil
	}
	return nil, nil
}
