package websocket

import (
	"testing"
	"time"
)

// test code update rate limiting (10/second)
func TestCodeUpdateRateLimit(t *testing.T) {
	client := &Client{
		codeUpdateTimestamps: make([]time.Time, 0, maxCodeUpdatesPerSecond),
	}

	// first 10 updates should pass
	for i := 0; i < maxCodeUpdatesPerSecond; i++ {
		if !client.checkCodeUpdateRateLimit() {
			t.Errorf("Code update %d should have been allowed, but was rate limited", i+1)
		}
	}

	// 11th update should be rate limited
	if client.checkCodeUpdateRateLimit() {
		t.Error("11th code update should have been rate limited, but was allowed")
	}

	if len(client.codeUpdateTimestamps) != maxCodeUpdatesPerSecond {
		t.Errorf("Expected %d timestamps, got %d", maxCodeUpdatesPerSecond, len(client.codeUpdateTimestamps))
	}
}

// test code update rate limit window expiration (1 second window)
func TestCodeUpdateRateLimitWindowExpiration(t *testing.T) {
	client := &Client{
		codeUpdateTimestamps: make([]time.Time, 0, maxCodeUpdatesPerSecond),
	}

	// simulate 10 updates from 2 seconds ago (should be expired)
	twoSecondsAgo := time.Now().Add(-2 * time.Second)
	for i := 0; i < maxCodeUpdatesPerSecond; i++ {
		client.codeUpdateTimestamps = append(client.codeUpdateTimestamps, twoSecondsAgo)
	}

	// next update should pass because old timestamps are expired
	if !client.checkCodeUpdateRateLimit() {
		t.Error("Code update should have been allowed after old timestamps expired")
	}

	// old timestamps should be cleaned up
	if len(client.codeUpdateTimestamps) != 1 {
		t.Errorf("Expected 1 timestamp after cleanup, got %d", len(client.codeUpdateTimestamps))
	}
}

// test chat message rate limiting (20/minute)
func TestChatRateLimit(t *testing.T) {
	client := &Client{
		chatMessageTimestamps: make([]time.Time, 0, maxChatMessagesPerMinute),
	}

	// first 20 messages should pass
	for i := 0; i < maxChatMessagesPerMinute; i++ {
		if !client.checkChatRateLimit() {
			t.Errorf("Chat message %d should have been allowed, but was rate limited", i+1)
		}
	}

	// 21st message should be rate limited
	if client.checkChatRateLimit() {
		t.Error("21st chat message should have been rate limited, but was allowed")
	}

	if len(client.chatMessageTimestamps) != maxChatMessagesPerMinute {
		t.Errorf("Expected %d timestamps, got %d", maxChatMessagesPerMinute, len(client.chatMessageTimestamps))
	}
}

// test chat rate limit window expiration
func TestChatRateLimitWindowExpiration(t *testing.T) {
	client := &Client{
		chatMessageTimestamps: make([]time.Time, 0, maxChatMessagesPerMinute),
	}

	// simulate 20 messages from 2 minutes ago (should be expired)
	twoMinutesAgo := time.Now().Add(-2 * time.Minute)
	for i := 0; i < maxChatMessagesPerMinute; i++ {
		client.chatMessageTimestamps = append(client.chatMessageTimestamps, twoMinutesAgo)
	}

	// next message should pass because old timestamps are expired
	if !client.checkChatRateLimit() {
		t.Error("Chat message should have been allowed after old timestamps expired")
	}

	// old timestamps should be cleaned up
	if len(client.chatMessageTimestamps) != 1 {
		t.Errorf("Expected 1 timestamp after cleanup, got %d", len(client.chatMessageTimestamps))
	}
}

// test CanWrite permission check
func TestCanWrite(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{"host", true},
		{"co-author", true},
		{"viewer", false},
		{"", false},
	}

	for _, tt := range tests {
		client := &Client{Role: tt.role}
		got := client.CanWrite()
		if got != tt.expected {
			t.Errorf("CanWrite() for role %q = %v, want %v", tt.role, got, tt.expected)
		}
	}
}
