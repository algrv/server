package websocket

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHubCreation(t *testing.T) {
	hub := NewHub()
	require.NotNil(t, hub)
	assert.NotNil(t, hub.Register)
	assert.NotNil(t, hub.Unregister)
	assert.NotNil(t, hub.Broadcast)
}

func TestHubRegisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	// create mock client
	client := &Client{
		ID:          "test-client-1",
		SessionID:   "test-session",
		UserID:      "test-user",
		DisplayName: "Test User",
		Role:        "host",
		hub:         hub,
		send:        make(chan []byte, 256),
	}

	// register client
	hub.Register <- client

	// wait for registration
	time.Sleep(100 * time.Millisecond)

	// verify client is registered
	clients := hub.GetSessionClients("test-session")
	assert.Len(t, clients, 1)
	assert.Equal(t, "test-client-1", clients[0].ID)
}

func TestHubUnregisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	client := &Client{
		ID:          "test-client-1",
		SessionID:   "test-session",
		UserID:      "test-user",
		DisplayName: "Test User",
		Role:        "host",
		hub:         hub,
		send:        make(chan []byte, 256),
	}

	// Register then unregister
	hub.Register <- client
	time.Sleep(100 * time.Millisecond)

	hub.Unregister <- client
	time.Sleep(100 * time.Millisecond)

	// verify client is unregistered (session should be removed)
	count := hub.GetClientCount("test-session")
	assert.Equal(t, 0, count)
}

func TestHubBroadcastToSession(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	// create two clients in same session
	client1 := &Client{
		ID:          "client-1",
		SessionID:   "session-1",
		UserID:      "user-1",
		DisplayName: "User 1",
		Role:        "host",
		hub:         hub,
		send:        make(chan []byte, 256),
	}

	client2 := &Client{
		ID:          "client-2",
		SessionID:   "session-1",
		UserID:      "user-2",
		DisplayName: "User 2",
		Role:        "co-author",
		hub:         hub,
		send:        make(chan []byte, 256),
	}

	// register both clients
	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(100 * time.Millisecond)

	// Drain "user joined" messages
	select {
	case <-client1.send:
	default:
	}

	select {
	case <-client2.send:
	default:
	}

	// create test message
	msg, err := NewMessage(TypeCodeUpdate, "session-1", "user-1", CodeUpdatePayload{
		Code:        "sound(\"bd\")",
		DisplayName: "User 1",
	})
	require.NoError(t, err)

	// broadcast to session (exclude sender)
	hub.BroadcastToSession("session-1", msg, "client-1")
	time.Sleep(100 * time.Millisecond)

	// client 1 should NOT receive (was excluded)
	select {
	case <-client1.send:
		t.Error("client-1 should not have received message (was excluded)")
	default:
		// expected
	}

	// client 2 should receive
	select {
	case received := <-client2.send:
		var receivedMsg Message
		err := json.Unmarshal(received, &receivedMsg)
		require.NoError(t, err)
		assert.Equal(t, TypeCodeUpdate, receivedMsg.Type)
	case <-time.After(1 * time.Second):
		t.Error("client-2 should have received message")
	}
}

func TestHubBroadcastToAllClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	// create clients in different sessions
	client1 := &Client{
		ID:          "client-1",
		SessionID:   "session-1",
		UserID:      "user-1",
		DisplayName: "User 1",
		Role:        "host",
		hub:         hub,
		send:        make(chan []byte, 256),
	}

	client2 := &Client{
		ID:          "client-2",
		SessionID:   "session-2",
		UserID:      "user-2",
		DisplayName: "User 2",
		Role:        "host",
		hub:         hub,
		send:        make(chan []byte, 256),
	}

	// register both clients
	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(100 * time.Millisecond)

	// create test message
	msg, err := NewMessage(TypeCodeUpdate, "session-1", "user-1", CodeUpdatePayload{
		Code:        "sound(\"bd\")",
		DisplayName: "User 1",
	})
	require.NoError(t, err)

	// broadcast to session only
	hub.BroadcastToSession("session-1", msg, "")
	time.Sleep(100 * time.Millisecond)

	// client 1 should receive
	select {
	case <-client1.send:
		// expected
	case <-time.After(1 * time.Second):
		t.Error("client-1 should have received message")
	}

	// client 2 should NOT receive
	select {
	case <-client2.send:
		t.Error("client-2 should not have received message (different session)")
	default:
		// expected
	}
}

func TestHubMessageHandler(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	// register a test handler
	handlerCalled := false
	var handlerMu sync.Mutex

	testHandler := func(hub *Hub, client *Client, msg *Message) error {
		handlerMu.Lock()
		handlerCalled = true
		handlerMu.Unlock()
		return nil
	}

	hub.RegisterHandler("test_message", testHandler)

	// create client
	client := &Client{
		ID:          "client-1",
		SessionID:   "session-1",
		UserID:      "user-1",
		DisplayName: "User 1",
		Role:        "host",
		hub:         hub,
		send:        make(chan []byte, 256),
	}

	hub.Register <- client
	time.Sleep(100 * time.Millisecond)

	// create and send test message (set ClientID for routing)
	msg, err := NewMessage("test_message", "session-1", "user-1", map[string]interface{}{
		"test": "data",
	})

	require.NoError(t, err)
	msg.ClientID = "client-1" // set ClientID so handler can find sender

	// send message through broadcast channel
	hub.Broadcast <- msg

	// Wait a bit for handler to execute
	time.Sleep(200 * time.Millisecond)

	// Verify handler was called
	handlerMu.Lock()
	assert.True(t, handlerCalled, "handler should have been called")
	handlerMu.Unlock()
}

func TestHubMultipleClientsInSession(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	sessionID := "test-session"
	numClients := 5

	clients := make([]*Client, numClients)

	for i := range numClients {
		clients[i] = &Client{
			ID:          string(rune('a' + i)),
			SessionID:   sessionID,
			UserID:      string(rune('a' + i)),
			DisplayName: string(rune('A' + i)),
			Role:        "viewer",
			hub:         hub,
			send:        make(chan []byte, 256),
		}
		hub.Register <- clients[i]
	}

	time.Sleep(200 * time.Millisecond)

	// verify all clients are registered
	count := hub.GetClientCount(sessionID)
	assert.Equal(t, numClients, count)

	// Drain "user joined" messages
	for i := range numClients {
		for {
			select {
			case <-clients[i].send:
			default:
				goto nextClient
			}
		}
	nextClient:
	}

	// broadcast message
	msg, err := NewMessage(TypeCodeUpdate, sessionID, "a", CodeUpdatePayload{
		Code:        "sound(\"bd\")",
		DisplayName: "A",
	})
	require.NoError(t, err)

	hub.BroadcastToSession(sessionID, msg, "a")
	time.Sleep(200 * time.Millisecond)

	// first client (sender) should NOT receive
	select {
	case <-clients[0].send:
		t.Error("sender should not receive broadcast")
	default:
		// expected
	}

	// all other clients SHOULD receive
	for i := 1; i < numClients; i++ {
		select {
		case <-clients[i].send:
			// expected
		case <-time.After(1 * time.Second):
			t.Errorf("client %d should have received message", i)
		}
	}
}

func TestHubSessionCleanupAfterAllClientsLeave(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	sessionID := "test-session"

	client := &Client{
		ID:          "client-1",
		SessionID:   sessionID,
		UserID:      "user-1",
		DisplayName: "User 1",
		Role:        "host",
		hub:         hub,
		send:        make(chan []byte, 256),
	}

	// register client
	hub.Register <- client
	time.Sleep(100 * time.Millisecond)

	// verify session exists
	count := hub.GetClientCount(sessionID)
	assert.Equal(t, 1, count)

	// unregister client
	hub.Unregister <- client
	time.Sleep(100 * time.Millisecond)

	// verify session is cleaned up
	count = hub.GetClientCount(sessionID)
	assert.Equal(t, 0, count)
}

func TestHubConcurrentBroadcasts(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	sessionID := "test-session"
	numClients := 10
	numMessages := 20

	// create and register clients
	clients := make([]*Client, numClients)
	for i := range numClients {
		clients[i] = &Client{
			ID:          string(rune('a' + i)),
			SessionID:   sessionID,
			UserID:      string(rune('a' + i)),
			DisplayName: string(rune('A' + i)),
			Role:        "co-author",
			hub:         hub,
			send:        make(chan []byte, 256),
		}
		hub.Register <- clients[i]
	}

	time.Sleep(200 * time.Millisecond)

	// drain any "user joined" messages that were sent during registration
	for i := range numClients {
		for {
			select {
			case <-clients[i].send:
				// drain message
			default:
				goto drained
			}
		}
	drained:
	}

	// broadcast multiple messages concurrently
	var wg sync.WaitGroup
	for i := range numMessages {
		wg.Add(1)
		go func(msgNum int) {
			defer wg.Done()
			msg, _ := NewMessage(TypeCodeUpdate, sessionID, "a", CodeUpdatePayload{
				Code:        "sound(\"bd\")",
				DisplayName: "A",
			})
			hub.BroadcastToSession(sessionID, msg, "a")
		}(i)
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	// each client (except sender) should have received all messages
	for i := 1; i < numClients; i++ {
		receivedCount := 0

		for {
			select {
			case <-clients[i].send:
				receivedCount++
			default:
				goto done
			}
		}

	done:
		assert.Equal(t, numMessages, receivedCount, "client %d should receive all messages", i)
	}
}
