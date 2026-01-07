package ccsignals

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPasteLock     = "ccsignals:paste_lock:%s"
	keyPasteBaseline = "ccsignals:paste_baseline:%s"
)

// implements LockStore using Redis
type RedisLockStore struct {
	client *redis.Client
}

// creates a new Redis-backed lock store
func NewRedisLockStore(client *redis.Client) *RedisLockStore {
	return &RedisLockStore{client: client}
}

// creates a new Redis-backed lock store from a URL
func NewRedisLockStoreFromURL(redisURL string) (*RedisLockStore, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisLockStore{client: client}, nil
}

// sets a paste lock for a session with the baseline code
func (s *RedisLockStore) SetLock(ctx context.Context, sessionID, baselineCode string, ttl time.Duration) error {
	lockKey := fmt.Sprintf(keyPasteLock, sessionID)
	baselineKey := fmt.Sprintf(keyPasteBaseline, sessionID)

	pipe := s.client.Pipeline()
	pipe.Set(ctx, lockKey, "1", ttl)
	pipe.Set(ctx, baselineKey, baselineCode, ttl)

	_, err := pipe.Exec(ctx)
	return err
}

// retrieves the current lock state for a session
func (s *RedisLockStore) GetLock(ctx context.Context, sessionID string) (*LockState, error) {
	lockKey := fmt.Sprintf(keyPasteLock, sessionID)
	baselineKey := fmt.Sprintf(keyPasteBaseline, sessionID)

	// use pipeline to atomically get both values
	pipe := s.client.Pipeline()
	lockCmd := pipe.Get(ctx, lockKey)
	baselineCmd := pipe.Get(ctx, baselineKey)

	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		// check if it's just missing keys
		lockErr := lockCmd.Err()

		if errors.Is(lockErr, redis.Nil) {
			return &LockState{Locked: false}, nil
		}

		if lockErr != nil {
			return nil, lockErr
		}
	}

	// check if lock exists
	_, lockErr := lockCmd.Result()
	if errors.Is(lockErr, redis.Nil) {
		return &LockState{Locked: false}, nil
	}

	if lockErr != nil {
		return nil, lockErr
	}

	// get baseline (may be empty if expired between commands, but unlikely with pipeline)
	baseline, baselineErr := baselineCmd.Result()
	if errors.Is(baselineErr, redis.Nil) {
		// lock exists but baseline expired - treat as invalid lock
		return &LockState{Locked: false}, nil
	}

	if baselineErr != nil {
		return nil, baselineErr
	}

	// require non-empty baseline for valid lock
	if baseline == "" {
		return &LockState{Locked: false}, nil
	}

	return &LockState{
		Locked:       true,
		BaselineCode: baseline,
	}, nil
}

// removes the paste lock for a session
func (s *RedisLockStore) RemoveLock(ctx context.Context, sessionID string) error {
	lockKey := fmt.Sprintf(keyPasteLock, sessionID)
	baselineKey := fmt.Sprintf(keyPasteBaseline, sessionID)

	return s.client.Del(ctx, lockKey, baselineKey).Err()
}

// extends the lock TTL without changing other state
func (s *RedisLockStore) RefreshTTL(ctx context.Context, sessionID string, ttl time.Duration) error {
	lockKey := fmt.Sprintf(keyPasteLock, sessionID)
	baselineKey := fmt.Sprintf(keyPasteBaseline, sessionID)

	pipe := s.client.Pipeline()
	pipe.Expire(ctx, lockKey, ttl)
	pipe.Expire(ctx, baselineKey, ttl)

	_, err := pipe.Exec(ctx)
	return err
}

// closes the redis connection
func (s *RedisLockStore) Close() error {
	return s.client.Close()
}
