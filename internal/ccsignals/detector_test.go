package ccsignals

import (
	"context"
	"errors"
	"testing"
)

func TestDetector_IsLargeDelta(t *testing.T) {
	config := Config{
		PasteDeltaThreshold: 200,
		PasteLineThreshold:  50,
	}
	d := NewDetector(config, nil, nil)

	tests := []struct {
		name     string
		previous string
		new      string
		want     bool
	}{
		{
			name:     "no delta",
			previous: "hello",
			new:      "hello",
			want:     false,
		},
		{
			name:     "small delta",
			previous: "hello",
			new:      "hello world",
			want:     false,
		},
		{
			name:     "large char delta",
			previous: "",
			new:      string(make([]byte, 200)),
			want:     true,
		},
		{
			name:     "just under threshold",
			previous: "",
			new:      string(make([]byte, 199)),
			want:     false,
		},
		{
			name:     "large line delta",
			previous: "",
			new:      generateLines(50),
			want:     true,
		},
		{
			name:     "just under line threshold",
			previous: "",
			new:      generateLines(49),
			want:     false,
		},
		{
			name:     "negative delta ignored",
			previous: string(make([]byte, 500)),
			new:      "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.IsLargeDelta(tt.previous, tt.new)
			if got != tt.want {
				t.Errorf("IsLargeDelta() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetector_IsSignificantEdit(t *testing.T) {
	config := Config{
		UnlockThreshold: 0.30,
	}
	d := NewDetector(config, nil, nil)

	tests := []struct {
		name     string
		baseline string
		current  string
		want     bool
	}{
		{
			name:     "empty baseline",
			baseline: "",
			current:  "anything",
			want:     true,
		},
		{
			name:     "identical",
			baseline: "hello world",
			current:  "hello world",
			want:     false,
		},
		{
			name:     "small edit",
			baseline: "hello world this is a test",
			current:  "hello world this is a text",
			want:     false,
		},
		{
			name:     "30% edit",
			baseline: "1234567890",
			current:  "123",
			want:     true,
		},
		{
			name:     "complete replacement",
			baseline: "hello",
			current:  "world",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.IsSignificantEdit(tt.baseline, tt.current)
			if got != tt.want {
				t.Errorf("IsSignificantEdit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetector_DetectPaste(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()

	t.Run("no large delta returns no lock", func(t *testing.T) {
		d := NewDetector(config, nil, nil)
		result, err := d.DetectPaste(ctx, "session1", "user1", "hello", "hello world")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ShouldLock {
			t.Error("expected no lock for small delta")
		}
	})

	t.Run("large delta with no validator locks", func(t *testing.T) {
		d := NewDetector(config, nil, nil)
		largeCode := string(make([]byte, 300))
		result, err := d.DetectPaste(ctx, "session1", "user1", "", largeCode)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.ShouldLock {
			t.Error("expected lock for large delta with no validator")
		}
	})

	t.Run("user owns content no lock", func(t *testing.T) {
		validator := &mockValidator{
			ownershipMatch: &ContentMatch{Found: true, OwnerID: "user1"},
		}
		d := NewDetector(config, nil, validator)
		largeCode := string(make([]byte, 300))
		result, err := d.DetectPaste(ctx, "session1", "user1", "", largeCode)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ShouldLock {
			t.Error("expected no lock when user owns content")
		}
	})

	t.Run("public content allows AI no lock", func(t *testing.T) {
		validator := &mockValidator{
			publicMatch: &ContentMatch{Found: true, IsPublic: true, CCSignal: SignalCredit},
		}
		d := NewDetector(config, nil, validator)
		largeCode := string(make([]byte, 300))
		result, err := d.DetectPaste(ctx, "session1", "user1", "", largeCode)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ShouldLock {
			t.Error("expected no lock when public content allows AI")
		}
	})

	t.Run("public content no-ai locks", func(t *testing.T) {
		validator := &mockValidator{
			publicMatch: &ContentMatch{Found: true, IsPublic: true, CCSignal: SignalNoAI},
		}
		d := NewDetector(config, nil, validator)
		largeCode := string(make([]byte, 300))
		result, err := d.DetectPaste(ctx, "session1", "user1", "", largeCode)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.ShouldLock {
			t.Error("expected lock when public content has no-ai signal")
		}
	})

	t.Run("anonymous user large delta locks", func(t *testing.T) {
		d := NewDetector(config, nil, nil)
		largeCode := string(make([]byte, 300))
		result, err := d.DetectPaste(ctx, "session1", "", "", largeCode)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.ShouldLock {
			t.Error("expected lock for anonymous user with large delta")
		}
	})
}

func TestDetector_ProcessCodeUpdate(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()

	t.Run("nil store returns error", func(t *testing.T) {
		d := NewDetector(config, nil, nil)
		err := d.ProcessCodeUpdate(ctx, "session1", "user1", "", "code")
		if !errors.Is(err, ErrNilStore) {
			t.Errorf("expected ErrNilStore, got %v", err)
		}
	})

	t.Run("sets lock on large delta", func(t *testing.T) {
		store := NewMemoryLockStore()
		defer func() { _ = store.Close() }() //nolint:errcheck // test cleanup
		d := NewDetector(config, store, nil)

		largeCode := string(make([]byte, 300))
		err := d.ProcessCodeUpdate(ctx, "session1", "user1", "", largeCode)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		locked, err := d.IsLocked(ctx, "session1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !locked {
			t.Error("expected session to be locked")
		}
	})

	t.Run("unlocks on significant edit", func(t *testing.T) {
		store := NewMemoryLockStore()
		defer func() { _ = store.Close() }() //nolint:errcheck // test cleanup
		d := NewDetector(config, store, nil)

		// set initial lock
		largeCode := string(make([]byte, 300))
		if err := d.ProcessCodeUpdate(ctx, "session1", "user1", "", largeCode); err != nil {
			t.Fatalf("unexpected error setting lock: %v", err)
		}

		// make significant edit (delete most of it)
		err := d.ProcessCodeUpdate(ctx, "session1", "user1", largeCode, "short")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		locked, err := d.IsLocked(ctx, "session1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if locked {
			t.Error("expected session to be unlocked after significant edit")
		}
	})
}

func TestDetector_IsLocked(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()

	t.Run("nil store returns error", func(t *testing.T) {
		d := NewDetector(config, nil, nil)
		_, err := d.IsLocked(ctx, "session1")
		if !errors.Is(err, ErrNilStore) {
			t.Errorf("expected ErrNilStore, got %v", err)
		}
	})

	t.Run("unlocked session", func(t *testing.T) {
		store := NewMemoryLockStore()
		defer func() { _ = store.Close() }() //nolint:errcheck // test cleanup
		d := NewDetector(config, store, nil)

		locked, err := d.IsLocked(ctx, "session1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if locked {
			t.Error("expected session to be unlocked")
		}
	})
}

func TestDetector_WithFingerprints(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()

	t.Run("fingerprint match with no-ai locks", func(t *testing.T) {
		indexed := NewInMemoryIndexedStore(4, 15, 3)

		// add protected content with enough chars to trigger large delta (200+)
		protectedContent := "the quick brown fox jumps over the lazy dog and runs through the forest again and again until we have enough characters to trigger the paste detection threshold of two hundred characters easily done now"
		indexed.AddFromStrudel("work1", "creator1", SignalNoAI, protectedContent)

		d := NewDetector(config, nil, nil).WithFingerprints(indexed)

		// paste exact same content (guaranteed to match)
		result, err := d.DetectPaste(ctx, "session1", "user1", "", protectedContent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.ShouldLock {
			t.Error("expected lock for matching content with no-ai signal")
		}
		if result.FingerprintMatch == nil {
			t.Error("expected fingerprint match in result")
		}
	})

	t.Run("fingerprint match with allowed signal no lock", func(t *testing.T) {
		indexed := NewInMemoryIndexedStore(4, 15, 3)

		// add open content with enough chars to trigger large delta (200+)
		openContent := "the quick brown fox jumps over the lazy dog and runs through the meadow again and again until we have enough characters to trigger the paste detection threshold of two hundred characters easily done now"
		indexed.AddFromStrudel("work1", "creator1", SignalCredit, openContent)

		d := NewDetector(config, nil, nil).WithFingerprints(indexed)

		// paste exact same content (guaranteed to match)
		result, err := d.DetectPaste(ctx, "session1", "user1", "", openContent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ShouldLock {
			t.Error("expected no lock for matching content with allowed signal")
		}
	})
}

// helpers

func generateLines(n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += "\n"
	}
	return result
}

type mockValidator struct {
	ownershipMatch *ContentMatch
	ownershipErr   error
	publicMatch    *ContentMatch
	publicErr      error
}

func (m *mockValidator) ValidateOwnership(_ context.Context, _, _ string) (*ContentMatch, error) {
	if m.ownershipErr != nil {
		return nil, m.ownershipErr
	}
	if m.ownershipMatch != nil {
		return m.ownershipMatch, nil
	}
	return &ContentMatch{Found: false}, nil
}

func (m *mockValidator) ValidatePublicContent(_ context.Context, _ string) (*ContentMatch, error) {
	if m.publicErr != nil {
		return nil, m.publicErr
	}
	if m.publicMatch != nil {
		return m.publicMatch, nil
	}
	return &ContentMatch{Found: false}, nil
}

// compile-time interface check
var _ ContentValidator = (*mockValidator)(nil)
