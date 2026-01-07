package ccsignals

import (
	"context"
	"sync"
	"testing"
)

func TestMemoryFingerprintStore_Store(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryFingerprintStore()

	record := &FingerprintRecord{
		ID:          "record1",
		Fingerprint: 0xABCD,
		WorkID:      "work1",
		CreatorID:   "creator1",
		CCSignal:    SignalCredit,
	}

	err := store.Store(ctx, record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// verify stored
	got, err := store.GetByWorkID(ctx, "work1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("expected record to be found")
	}

	if got.ID != "record1" {
		t.Errorf("ID = %s, want record1", got.ID)
	}
}

func TestMemoryFingerprintStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryFingerprintStore()

	record := &FingerprintRecord{
		ID:     "record1",
		WorkID: "work1",
	}

	if err := store.Store(ctx, record); err != nil {
		t.Fatalf("failed to store: %v", err)
	}

	err := store.Delete(ctx, "record1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := store.GetByWorkID(ctx, "work1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected record to be deleted")
	}
}

func TestMemoryFingerprintStore_Delete_NotExists(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryFingerprintStore()

	// should not error
	err := store.Delete(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMemoryFingerprintStore_LoadAll(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryFingerprintStore()

	for _, r := range []*FingerprintRecord{
		{ID: "r1", WorkID: "w1"},
		{ID: "r2", WorkID: "w2"},
		{ID: "r3", WorkID: "w3"},
	} {
		if err := store.Store(ctx, r); err != nil {
			t.Fatalf("failed to store: %v", err)
		}
	}

	records, err := store.LoadAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 3 {
		t.Errorf("LoadAll returned %d records, want 3", len(records))
	}
}

func TestMemoryFingerprintStore_LoadAll_Empty(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryFingerprintStore()

	records, err := store.LoadAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 0 {
		t.Errorf("LoadAll returned %d records, want 0", len(records))
	}
}

func TestMemoryFingerprintStore_GetByWorkID(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryFingerprintStore()

	if err := store.Store(ctx, &FingerprintRecord{
		ID:       "record1",
		WorkID:   "work1",
		CCSignal: SignalNoAI,
	}); err != nil {
		t.Fatalf("failed to store: %v", err)
	}

	got, err := store.GetByWorkID(ctx, "work1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("expected record to be found")
	}

	if got.CCSignal != SignalNoAI {
		t.Errorf("CCSignal = %s, want %s", got.CCSignal, SignalNoAI)
	}
}

func TestMemoryFingerprintStore_GetByWorkID_NotExists(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryFingerprintStore()

	got, err := store.GetByWorkID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent work")
	}
}

func TestMemoryFingerprintStore_Overwrite(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryFingerprintStore()

	// store initial
	if err := store.Store(ctx, &FingerprintRecord{
		ID:       "record1",
		WorkID:   "work1",
		CCSignal: SignalCredit,
	}); err != nil {
		t.Fatalf("failed to store: %v", err)
	}

	// overwrite with same ID
	if err := store.Store(ctx, &FingerprintRecord{
		ID:       "record1",
		WorkID:   "work1",
		CCSignal: SignalNoAI,
	}); err != nil {
		t.Fatalf("failed to store: %v", err)
	}

	got, err := store.GetByWorkID(ctx, "work1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.CCSignal != SignalNoAI {
		t.Errorf("CCSignal = %s, want %s (overwritten value)", got.CCSignal, SignalNoAI)
	}
}

func TestMemoryFingerprintStore_Concurrent(_ *testing.T) {
	ctx := context.Background()
	store := NewMemoryFingerprintStore()

	var wg sync.WaitGroup

	// concurrent writes - stress test, errors not checked intentionally
	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			record := &FingerprintRecord{
				ID:     string(rune('a' + id%26)),
				WorkID: string(rune('A' + id%26)),
			}
			_ = store.Store(ctx, record)                 //nolint:errcheck // stress test
			_, _ = store.GetByWorkID(ctx, record.WorkID) //nolint:errcheck // stress test
		}(i)
	}

	// concurrent reads - stress test, errors not checked intentionally
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = store.LoadAll(ctx) //nolint:errcheck // stress test
		}()
	}

	wg.Wait()
}
