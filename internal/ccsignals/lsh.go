package ccsignals

import (
	"context"
	"sync"
)

const (
	// 4 bands with 16 bits each = 64 bits total
	DefaultNumBands = 4

	// 10 bits out of 64 = ~84% similarity
	DefaultSimilarityThreshold = 10

	// minimum bands to avoid uint16 overflow (64/4 = 16 bits per band)
	MinNumBands = 4
)

// stores a fingerprint with its metadata
type FingerprintRecord struct {
	ID          string
	Fingerprint Fingerprint
	WorkID      string
	CreatorID   string
	CCSignal    CCSignal
	Content     string
}

// represents a fingerprint match
type MatchResult struct {
	Record   *FingerprintRecord
	Distance int
}

// LSHIndex provides locality-sensitive hashing for efficient similarity search.
// divides fingerprints into bands and uses band values as bucket keys,
// reducing average lookup from O(n) to O(candidates in shared buckets).
type LSHIndex struct {
	mu          sync.RWMutex
	numBands    int
	bitsPerBand int
	threshold   int

	// buckets[bandIndex][bandValue] = list of record IDs
	buckets []map[uint16][]string

	// records stores all fingerprint records by ID
	records map[string]*FingerprintRecord
}

// creates a new LSH index with the given configuration
func NewLSHIndex(numBands, similarityThreshold int) *LSHIndex {
	// enforce minimum bands to prevent uint16 overflow
	if numBands < MinNumBands {
		numBands = DefaultNumBands
	}

	if numBands > 8 {
		numBands = 8
	}

	if similarityThreshold < 1 {
		similarityThreshold = DefaultSimilarityThreshold
	}

	bitsPerBand := HashBits / numBands

	index := &LSHIndex{
		numBands:    numBands,
		bitsPerBand: bitsPerBand,
		threshold:   similarityThreshold,
		buckets:     make([]map[uint16][]string, numBands),
		records:     make(map[string]*FingerprintRecord),
	}

	for i := 0; i < numBands; i++ {
		index.buckets[i] = make(map[uint16][]string)
	}

	return index
}

// adds a fingerprint record to the index
func (idx *LSHIndex) Insert(record *FingerprintRecord) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.records[record.ID] = record

	bands := idx.getBands(record.Fingerprint)
	for i, bandValue := range bands {
		idx.buckets[i][bandValue] = append(idx.buckets[i][bandValue], record.ID)
	}
}

// removes a fingerprint record from the index
func (idx *LSHIndex) Remove(id string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	record, exists := idx.records[id]
	if !exists {
		return
	}

	bands := idx.getBands(record.Fingerprint)
	for i, bandValue := range bands {
		bucket := idx.buckets[i][bandValue]
		for j, recordID := range bucket {
			if recordID == id {
				idx.buckets[i][bandValue] = append(bucket[:j], bucket[j+1:]...)
				break
			}
		}
	}

	delete(idx.records, id)
}

// finds all similar fingerprints within the similarity threshold
func (idx *LSHIndex) Query(fingerprint Fingerprint) []*MatchResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	candidateSet := make(map[string]struct{})
	bands := idx.getBands(fingerprint)

	for i, bandValue := range bands {
		for _, id := range idx.buckets[i][bandValue] {
			candidateSet[id] = struct{}{}
		}
	}

	var results []*MatchResult
	for id := range candidateSet {
		record := idx.records[id]
		if record == nil {
			continue
		}

		distance := HammingDistance(fingerprint, record.Fingerprint)
		if distance <= idx.threshold {
			results = append(results, &MatchResult{
				Record:   record,
				Distance: distance,
			})
		}
	}

	return results
}

// finds the best matching fingerprint (lowest hamming distance)
func (idx *LSHIndex) QueryBest(fingerprint Fingerprint) *MatchResult {
	results := idx.Query(fingerprint)
	if len(results) == 0 {
		return nil
	}

	best := results[0]
	for _, r := range results[1:] {
		if r.Distance < best.Distance {
			best = r
		}
	}

	return best
}

// extracts band values from a fingerprint
func (idx *LSHIndex) getBands(fp Fingerprint) []uint16 {
	bands := make([]uint16, idx.numBands)
	mask := uint64((1 << idx.bitsPerBand) - 1)

	for i := 0; i < idx.numBands; i++ {
		shift := i * idx.bitsPerBand
		bands[i] = uint16((uint64(fp) >> shift) & mask) //nolint:gosec // mask ensures value fits in uint16
	}

	return bands
}

// returns the number of records in the index
func (idx *LSHIndex) Size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.records)
}

// retrieves a record by ID
func (idx *LSHIndex) GetRecord(id string) *FingerprintRecord {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.records[id]
}

// defines the interface for persistent fingerprint storage
type FingerprintStore interface {
	Store(ctx context.Context, record *FingerprintRecord) error
	Delete(ctx context.Context, id string) error
	LoadAll(ctx context.Context) ([]*FingerprintRecord, error)
	GetByWorkID(ctx context.Context, workID string) (*FingerprintRecord, error)
}

// combines LSH index with persistent storage
type IndexedFingerprintStore struct {
	index  *LSHIndex
	store  FingerprintStore
	hasher *SimHasher
}

// creates a new indexed store
func NewIndexedFingerprintStore(store FingerprintStore, numBands, threshold, shingleSize int) *IndexedFingerprintStore {
	return &IndexedFingerprintStore{
		index:  NewLSHIndex(numBands, threshold),
		store:  store,
		hasher: NewSimHasher(shingleSize),
	}
}

// loads all records from storage into the index
func (s *IndexedFingerprintStore) Initialize(ctx context.Context) error {
	records, err := s.store.LoadAll(ctx)
	if err != nil {
		return err
	}

	for _, record := range records {
		s.index.Insert(record)
	}

	return nil
}

// creates a fingerprint for content and stores it
func (s *IndexedFingerprintStore) Add(ctx context.Context, workID, creatorID string, ccSignal CCSignal, content string) (*FingerprintRecord, error) {
	fingerprint := s.hasher.Hash(content)

	record := &FingerprintRecord{
		ID:          workID,
		Fingerprint: fingerprint,
		WorkID:      workID,
		CreatorID:   creatorID,
		CCSignal:    ccSignal,
		Content:     content,
	}

	if err := s.store.Store(ctx, record); err != nil {
		return nil, err
	}

	s.index.Insert(record)

	return record, nil
}

// removes a fingerprint by work ID
func (s *IndexedFingerprintStore) Remove(ctx context.Context, workID string) error {
	s.index.Remove(workID)
	return s.store.Delete(ctx, workID)
}

// searches for content similar to the query
func (s *IndexedFingerprintStore) FindSimilar(content string) []*MatchResult {
	fingerprint := s.hasher.Hash(content)
	return s.index.Query(fingerprint)
}

// finds the most similar content
func (s *IndexedFingerprintStore) FindBestMatch(content string) *MatchResult {
	fingerprint := s.hasher.Hash(content)
	return s.index.QueryBest(fingerprint)
}

// returns the number of indexed fingerprints
func (s *IndexedFingerprintStore) Size() int {
	return s.index.Size()
}
