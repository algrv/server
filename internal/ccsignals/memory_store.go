package ccsignals

import (
	"context"
	"sync"
	"time"
)

// MemoryLockStore implements LockStore using in-memory storage
type MemoryLockStore struct {
	mu     sync.RWMutex
	locks  map[string]*memoryLock
	done   chan struct{}
	closed bool
}

type memoryLock struct {
	baseline  string
	lockedAt  time.Time
	expiresAt time.Time
}

// creates a new in-memory lock store
func NewMemoryLockStore() *MemoryLockStore {
	store := &MemoryLockStore{
		locks: make(map[string]*memoryLock),
		done:  make(chan struct{}),
	}

	go store.cleanupLoop()

	return store
}

// sets a paste lock for a session with the baseline code
func (s *MemoryLockStore) SetLock(_ context.Context, sessionID, baselineCode string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()

	s.locks[sessionID] = &memoryLock{
		baseline:  baselineCode,
		lockedAt:  now,
		expiresAt: now.Add(ttl),
	}

	return nil
}

// retrieves the current lock state for a session
func (s *MemoryLockStore) GetLock(_ context.Context, sessionID string) (*LockState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, exists := s.locks[sessionID]
	if !exists {
		return &LockState{Locked: false}, nil
	}

	if time.Now().After(lock.expiresAt) {
		delete(s.locks, sessionID)
		return &LockState{Locked: false}, nil
	}

	return &LockState{
		Locked:       true,
		BaselineCode: lock.baseline,
		LockedAt:     lock.lockedAt,
	}, nil
}

// removes the paste lock for a session
func (s *MemoryLockStore) RemoveLock(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.locks, sessionID)
	return nil
}

// extends the lock TTL without changing other state
func (s *MemoryLockStore) RefreshTTL(_ context.Context, sessionID string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if lock, exists := s.locks[sessionID]; exists {
		lock.expiresAt = time.Now().Add(ttl)
	}

	return nil
}

// stops the cleanup goroutine
func (s *MemoryLockStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.done)
	return nil
}

func (s *MemoryLockStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

func (s *MemoryLockStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for sessionID, lock := range s.locks {
		if now.After(lock.expiresAt) {
			delete(s.locks, sessionID)
		}
	}
}
