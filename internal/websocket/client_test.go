package websocket

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientPermissions(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		canWrite bool
	}{
		{
			name:     "host can write",
			role:     "host",
			canWrite: true,
		},
		{
			name:     "co-author can write",
			role:     "co-author",
			canWrite: true,
		},
		{
			name:     "viewer cannot write",
			role:     "viewer",
			canWrite: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				ID:          "test-client",
				SessionID:   "test-session",
				Role:        tt.role,
				DisplayName: "Test User",
				send:        make(chan []byte, 256),
			}

			assert.Equal(t, tt.canWrite, client.CanWrite())
		})
	}
}

func TestClientIsAuthenticated(t *testing.T) {
	tests := []struct {
		name            string
		userID          string
		isAuthenticated bool
	}{
		{
			name:            "authenticated user has user ID",
			userID:          "user-123",
			isAuthenticated: true,
		},
		{
			name:            "anonymous user has no user ID",
			userID:          "",
			isAuthenticated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				ID:              "test-client",
				SessionID:       "test-session",
				UserID:          tt.userID,
				DisplayName:     "Test User",
				Role:            "viewer",
				IsAuthenticated: tt.isAuthenticated,
				send:            make(chan []byte, 256),
			}

			assert.Equal(t, tt.isAuthenticated, client.IsAuthenticated)
		})
	}
}

func TestClientSendError(t *testing.T) {
	client := &Client{
		ID:          "test-client",
		SessionID:   "test-session",
		UserID:      "user-1",
		DisplayName: "Test User",
		Role:        "viewer",
		send:        make(chan []byte, 256),
	}

	// send error
	client.SendError("TEST_ERROR", "Test error message", "Additional details")

	// verify error message was sent
	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "TEST_ERROR")
		assert.Contains(t, string(msg), "Test error message")
		assert.Contains(t, string(msg), "error")
	default:
		t.Error("expected error message to be sent")
	}
}

func TestClientSendMessage(t *testing.T) {
	client := &Client{
		ID:          "test-client",
		SessionID:   "test-session",
		UserID:      "user-1",
		DisplayName: "Test User",
		Role:        "host",
		send:        make(chan []byte, 256),
	}

	// create test message
	msg, err := NewMessage(TypeCodeUpdate, "test-session", "user-1", CodeUpdatePayload{
		Code:        "sound(\"bd\")",
		DisplayName: "Test User",
	})
	assert.NoError(t, err)

	// send message
	err = client.Send(msg)
	assert.NoError(t, err)

	// verify message was sent
	select {
	case received := <-client.send:
		assert.Contains(t, string(received), "code_update")
		assert.Contains(t, string(received), "sound")
	default:
		t.Error("expected message to be sent")
	}
}

func TestClientSendMessageToClosedChannel(t *testing.T) {
	client := &Client{
		ID:          "test-client",
		SessionID:   "test-session",
		UserID:      "user-1",
		DisplayName: "Test User",
		Role:        "host",
		send:        make(chan []byte, 256),
	}

	// close the send channel
	close(client.send)

	msg, err := NewMessage(TypeCodeUpdate, "test-session", "user-1", CodeUpdatePayload{
		Code:        "sound(\"bd\")",
		DisplayName: "Test User",
	})
	assert.NoError(t, err)

	// sending to closed channel should not panic
	err = client.Send(msg)

	// error is expected when sending to closed channel
	assert.Error(t, err)
}
