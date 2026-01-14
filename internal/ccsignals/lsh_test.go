package ccsignals

import (
	"sync"
	"testing"
)

func TestLSHIndex_NewLSHIndex(t *testing.T) {
	tests := []struct {
		name              string
		numBands          int
		threshold         int
		expectedBands     int
		expectedThreshold int
	}{
		{
			name:              "default values",
			numBands:          0,
			threshold:         0,
			expectedBands:     DefaultNumBands,
			expectedThreshold: DefaultSimilarityThreshold,
		},
		{
			name:              "below minimum bands",
			numBands:          2,
			threshold:         5,
			expectedBands:     DefaultNumBands,
			expectedThreshold: 5,
		},
		{
			name:              "valid bands",
			numBands:          4,
			threshold:         8,
			expectedBands:     4,
			expectedThreshold: 8,
		},
		{
			name:              "above maximum bands",
			numBands:          10,
			threshold:         5,
			expectedBands:     8,
			expectedThreshold: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := NewLSHIndex(tt.numBands, tt.threshold)
			if idx.numBands != tt.expectedBands {
				t.Errorf("numBands = %d, want %d", idx.numBands, tt.expectedBands)
			}
			if idx.threshold != tt.expectedThreshold {
				t.Errorf("threshold = %d, want %d", idx.threshold, tt.expectedThreshold)
			}
		})
	}
}

func TestLSHIndex_InsertAndQuery(t *testing.T) {
	idx := NewLSHIndex(4, 10)

	record := &FingerprintRecord{
		ID:          "record1",
		Fingerprint: 0xABCDEF1234567890,
		WorkID:      "work1",
		CreatorID:   "creator1",
		CCSignal:    SignalCredit,
	}

	idx.Insert(record)

	if idx.Size() != 1 {
		t.Errorf("Size() = %d, want 1", idx.Size())
	}

	// exact match
	results := idx.Query(0xABCDEF1234567890)
	if len(results) != 1 {
		t.Fatalf("Query returned %d results, want 1", len(results))
	}
	if results[0].Distance != 0 {
		t.Errorf("Distance = %d, want 0", results[0].Distance)
	}
}

func TestLSHIndex_QuerySimilar(t *testing.T) {
	idx := NewLSHIndex(4, 10)

	record := &FingerprintRecord{
		ID:          "record1",
		Fingerprint: 0xFFFFFFFFFFFFFFFF,
		WorkID:      "work1",
		CCSignal:    SignalNoAI,
	}

	idx.Insert(record)

	// flip 5 bits - should match (within threshold of 10)
	similar := Fingerprint(0xFFFFFFFFFFFFFFE0)
	results := idx.Query(similar)

	if len(results) != 1 {
		t.Fatalf("expected 1 result for similar fingerprint, got %d", len(results))
	}
	if results[0].Distance > 10 {
		t.Errorf("distance %d exceeds threshold", results[0].Distance)
	}
}

func TestLSHIndex_QueryNoMatch(t *testing.T) {
	idx := NewLSHIndex(4, 5)

	record := &FingerprintRecord{
		ID:          "record1",
		Fingerprint: 0xFFFFFFFFFFFFFFFF,
		WorkID:      "work1",
	}
	idx.Insert(record)

	// completely different fingerprint
	results := idx.Query(0x0000000000000000)

	if len(results) != 0 {
		t.Errorf("expected no results for dissimilar fingerprint, got %d", len(results))
	}
}

func TestLSHIndex_Remove(t *testing.T) {
	idx := NewLSHIndex(4, 10)

	record := &FingerprintRecord{
		ID:          "record1",
		Fingerprint: 0xABCDEF1234567890,
		WorkID:      "work1",
	}
	idx.Insert(record)

	idx.Remove("record1")

	if idx.Size() != 0 {
		t.Errorf("Size() = %d after removal, want 0", idx.Size())
	}

	results := idx.Query(0xABCDEF1234567890)
	if len(results) != 0 {
		t.Errorf("expected no results after removal, got %d", len(results))
	}
}

func TestLSHIndex_RemoveNonexistent(_ *testing.T) {
	idx := NewLSHIndex(4, 10)

	// should not panic
	idx.Remove("nonexistent")
}

func TestLSHIndex_QueryBest(t *testing.T) {
	idx := NewLSHIndex(4, 20)

	// insert multiple records with varying similarity
	idx.Insert(&FingerprintRecord{
		ID:          "record1",
		Fingerprint: 0xFFFFFFFFFFFFFFFF,
		WorkID:      "work1",
	})
	idx.Insert(&FingerprintRecord{
		ID:          "record2",
		Fingerprint: 0xFFFFFFFFFFFFFFF0, // 4 bits different
		WorkID:      "work2",
	})
	idx.Insert(&FingerprintRecord{
		ID:          "record3",
		Fingerprint: 0xFFFFFFFFFFFFFF00, // 8 bits different
		WorkID:      "work3",
	})

	// query with fingerprint closest to record2
	query := Fingerprint(0xFFFFFFFFFFFFFFF0)
	best := idx.QueryBest(query)

	if best == nil {
		t.Fatal("expected a best match")
	}
	if best.Record.ID != "record2" {
		t.Errorf("expected record2 as best match, got %s", best.Record.ID)
	}
	if best.Distance != 0 {
		t.Errorf("expected distance 0 for exact match, got %d", best.Distance)
	}
}

func TestLSHIndex_QueryBest_NoMatch(t *testing.T) {
	idx := NewLSHIndex(4, 5)

	idx.Insert(&FingerprintRecord{
		ID:          "record1",
		Fingerprint: 0xFFFFFFFFFFFFFFFF,
	})

	// query with very different fingerprint
	best := idx.QueryBest(0x0000000000000000)
	if best != nil {
		t.Errorf("expected no match, got %v", best)
	}
}

func TestLSHIndex_GetRecord(t *testing.T) {
	idx := NewLSHIndex(4, 10)

	record := &FingerprintRecord{
		ID:          "record1",
		Fingerprint: 0xABCDEF,
		WorkID:      "work1",
		CCSignal:    SignalNoAI,
	}
	idx.Insert(record)

	got := idx.GetRecord("record1")
	if got == nil {
		t.Fatal("expected record to be found")
	}
	if got.WorkID != "work1" {
		t.Errorf("WorkID = %s, want work1", got.WorkID)
	}

	// non-existent
	notFound := idx.GetRecord("nonexistent")
	if notFound != nil {
		t.Errorf("expected nil for nonexistent record")
	}
}

func TestLSHIndex_Concurrent(_ *testing.T) {
	idx := NewLSHIndex(4, 10)
	var wg sync.WaitGroup

	// concurrent inserts
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			record := &FingerprintRecord{
				ID:          string(rune('a' + id%26)),
				Fingerprint: Fingerprint(id * 12345),
			}
			idx.Insert(record)
		}(i)
	}

	// concurrent queries
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = idx.Query(Fingerprint(id * 12345))
		}(i)
	}

	wg.Wait()
}

func TestIndexedFingerprintStore_AddFromStrudel(t *testing.T) {
	indexed := NewInMemoryIndexedStore(4, 10, 3)

	indexed.AddFromStrudel("work1", "creator1", SignalCredit, "test content here")

	if indexed.Size() != 1 {
		t.Errorf("Size() = %d, want 1", indexed.Size())
	}
}

func TestIndexedFingerprintStore_FindSimilar(t *testing.T) {
	indexed := NewInMemoryIndexedStore(4, 10, 3)

	content := "the quick brown fox jumps over the lazy dog"
	indexed.AddFromStrudel("work1", "creator1", SignalNoAI, content)

	// search for identical content (guaranteed to match)
	results := indexed.FindSimilar(content)

	if len(results) == 0 {
		t.Error("expected to find similar content")
	}
}

func TestIndexedFingerprintStore_FindBestMatch(t *testing.T) {
	indexed := NewInMemoryIndexedStore(4, 10, 3)

	indexed.AddFromStrudel("work1", "creator1", SignalNoAI, "hello world this is a test")
	indexed.AddFromStrudel("work2", "creator2", SignalCredit, "completely different content here")

	best := indexed.FindBestMatch("hello world this is a test")
	if best == nil {
		t.Fatal("expected to find best match")
	}
	if best.Record.WorkID != "work1" {
		t.Errorf("expected work1, got %s", best.Record.WorkID)
	}
}

func TestIndexedFingerprintStore_Remove(t *testing.T) {
	indexed := NewInMemoryIndexedStore(4, 10, 3)

	indexed.AddFromStrudel("work1", "creator1", SignalCredit, "test content")
	indexed.Remove("work1")

	if indexed.Size() != 0 {
		t.Errorf("Size() = %d after removal, want 0", indexed.Size())
	}
}

func TestIndexedFingerprintStore_InsertRecord(t *testing.T) {
	indexed := NewInMemoryIndexedStore(4, 10, 3)

	// add records directly via InsertRecord
	indexed.InsertRecord(&FingerprintRecord{
		ID:          "work1",
		Fingerprint: 0xABCD,
		WorkID:      "work1",
		CCSignal:    SignalNoAI,
	})
	indexed.InsertRecord(&FingerprintRecord{
		ID:          "work2",
		Fingerprint: 0x1234,
		WorkID:      "work2",
		CCSignal:    SignalCredit,
	})

	if indexed.Size() != 2 {
		t.Errorf("Size() = %d, want 2", indexed.Size())
	}
}
