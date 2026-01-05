package buffer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/algrv/server/internal/logger"
)

// handles Redis-backed buffering for session data
type SessionBuffer struct {
	client       *redis.Client
	flushTimeout time.Duration
}

// creates a new session buffer with Redis connection
func NewSessionBuffer(redisURL string, flushTimeout time.Duration) (*SessionBuffer, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	client := redis.NewClient(opts)

	// test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("connected to redis")

	return &SessionBuffer{
		client:       client,
		flushTimeout: flushTimeout,
	}, nil
}

// closes the Redis connection
func (b *SessionBuffer) Close() error {
	return b.client.Close()
}

// stores the current code for a session and marks it dirty
func (b *SessionBuffer) SetCode(ctx context.Context, sessionID, code string) error {
	pipe := b.client.Pipeline()

	// set the code
	codeKey := fmt.Sprintf(keySessionCode, sessionID)
	pipe.Set(ctx, codeKey, code, 0)

	// mark session as dirty for code
	pipe.SAdd(ctx, keyDirtySessionsCode, sessionID)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set code in redis: %w", err)
	}

	return nil
}

// retrieves the current code for a session from Redis
// returns empty string if not found (caller should fall back to Postgres)
func (b *SessionBuffer) GetCode(ctx context.Context, sessionID string) (string, error) {
	codeKey := fmt.Sprintf(keySessionCode, sessionID)
	code, err := b.client.Get(ctx, codeKey).Result()

	if errors.Is(err, redis.Nil) {
		return "", nil // not in redis, caller should check postgres
	}

	if err != nil {
		return "", fmt.Errorf("failed to get code from redis: %w", err)
	}

	return code, nil
}

// appends a message to the session's message buffer
func (b *SessionBuffer) AddMessage(ctx context.Context, msg *BufferedMessage) error {
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	pipe := b.client.Pipeline()

	// append message to list
	msgKey := fmt.Sprintf(keySessionMessages, msg.SessionID)
	pipe.RPush(ctx, msgKey, msgJSON)

	// mark session as dirty for messages
	pipe.SAdd(ctx, keyDirtySessionsMessages, msg.SessionID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add message to redis: %w", err)
	}

	return nil
}

// returns all session IDs with unflushed code changes
func (b *SessionBuffer) GetDirtyCodeSessions(ctx context.Context) ([]string, error) {
	return b.client.SMembers(ctx, keyDirtySessionsCode).Result()
}

// returns all session IDs with unflushed messages
func (b *SessionBuffer) GetDirtyMessageSessions(ctx context.Context) ([]string, error) {
	return b.client.SMembers(ctx, keyDirtySessionsMessages).Result()
}

// retrieves and clears the code for a session
// returns the code and removes the session from dirty set
func (b *SessionBuffer) FlushCode(ctx context.Context, sessionID string) (string, error) {
	codeKey := fmt.Sprintf(keySessionCode, sessionID)

	// get the code
	code, err := b.client.Get(ctx, codeKey).Result()
	if errors.Is(err, redis.Nil) {
		// no code to flush, just remove from dirty set
		b.client.SRem(ctx, keyDirtySessionsCode, sessionID) //nolint:errcheck // best-effort cleanup
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get code for flush: %w", err)
	}

	// remove from dirty set (keep the code in redis for reads)
	b.client.SRem(ctx, keyDirtySessionsCode, sessionID)

	return code, nil
}

// retrieves and clears all messages for a session
// returns the messages and removes the session from dirty set
func (b *SessionBuffer) FlushMessages(ctx context.Context, sessionID string) ([]BufferedMessage, error) {
	msgKey := fmt.Sprintf(keySessionMessages, sessionID)

	// get all messages
	msgJSONs, err := b.client.LRange(ctx, msgKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get messages for flush: %w", err)
	}

	if len(msgJSONs) == 0 {
		b.client.SRem(ctx, keyDirtySessionsMessages, sessionID)
		return nil, nil
	}

	// parse messages
	messages := make([]BufferedMessage, 0, len(msgJSONs))
	for _, msgJSON := range msgJSONs {
		var msg BufferedMessage
		if err := json.Unmarshal([]byte(msgJSON), &msg); err != nil {
			logger.ErrorErr(err, "failed to unmarshal buffered message", "session_id", sessionID)
			continue
		}
		messages = append(messages, msg)
	}

	// clear the list and remove from dirty set
	pipe := b.client.Pipeline()
	pipe.Del(ctx, msgKey)
	pipe.SRem(ctx, keyDirtySessionsMessages, sessionID)
	pipe.Exec(ctx) //nolint:errcheck,gosec // best-effort cleanup, messages already retrieved

	return messages, nil
}

// removes all buffered data for a session (call after session ends)
func (b *SessionBuffer) ClearSession(ctx context.Context, sessionID string) error {
	codeKey := fmt.Sprintf(keySessionCode, sessionID)
	msgKey := fmt.Sprintf(keySessionMessages, sessionID)

	pipe := b.client.Pipeline()
	pipe.Del(ctx, codeKey)
	pipe.Del(ctx, msgKey)
	pipe.SRem(ctx, keyDirtySessionsCode, sessionID)
	pipe.SRem(ctx, keyDirtySessionsMessages, sessionID)

	_, err := pipe.Exec(ctx)
	return err
}

// returns the underlying Redis client for advanced operations
func (b *SessionBuffer) Client() *redis.Client {
	return b.client
}
