package ccsignals

import (
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
	ID            string
	Fingerprint   Fingerprint
	WorkID        string
	CreatorID     string
	CCSignal      CCSignal
	Content       string
	ContentLength int
}

// represents a fingerprint match
type MatchResult struct {
	Record   *FingerprintRecord
	Distance int
}

// provides locality-sensitive hashing for efficient similarity search.
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

// in-memory indexed fingerprint store (no persistence, computed from strudels at startup)
type IndexedFingerprintStore struct {
	index  *LSHIndex
	hasher *SimHasher
}

// creates a new in-memory indexed store
func NewInMemoryIndexedStore(numBands, threshold, shingleSize int) *IndexedFingerprintStore {
	return &IndexedFingerprintStore{
		index:  NewLSHIndex(numBands, threshold),
		hasher: NewSimHasher(shingleSize),
	}
}

// adds a fingerprint record computed from a strudel
func (s *IndexedFingerprintStore) AddFromStrudel(workID, creatorID string, ccSignal CCSignal, content string) {
	fingerprint := s.hasher.Hash(content)

	record := &FingerprintRecord{
		ID:            workID,
		Fingerprint:   fingerprint,
		WorkID:        workID,
		CreatorID:     creatorID,
		CCSignal:      ccSignal,
		Content:       content,
		ContentLength: len(content),
	}

	s.index.Insert(record)
}

// inserts a pre-loaded record directly into the index
func (s *IndexedFingerprintStore) InsertRecord(record *FingerprintRecord) {
	s.index.Insert(record)
}

// removes a fingerprint by work ID
func (s *IndexedFingerprintStore) Remove(workID string) {
	s.index.Remove(workID)
}

// updates a fingerprint only if the content changed significantly (optimization for frequent autosaves).
// "significant" means at least one line added or removed - small edits don't meaningfully affect SimHash.
func (s *IndexedFingerprintStore) UpdateFromStrudel(workID, creatorID string, ccSignal CCSignal, content string) bool {
	// check if record exists
	s.index.mu.RLock()
	existing, exists := s.index.records[workID]
	s.index.mu.RUnlock()

	if exists {
		// skip if content identical
		if existing.Content == content {
			return false
		}

		// skip if line count unchanged (small edits within lines)
		oldLines := countLines(existing.Content)
		newLines := countLines(content)
		if oldLines == newLines {
			return false
		}
	}

	// significant change or new record - remove old and add new
	s.Remove(workID)
	s.AddFromStrudel(workID, creatorID, ccSignal, content)
	return true
}

// counts newlines in content (fast, no allocation)
func countLines(s string) int {
	count := 1
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			count++
		}
	}
	return count
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
