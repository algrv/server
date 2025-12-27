package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512 KB
)

// represents a WebSocket client connection
type Client struct {
	// unique identifier for this client
	ID string

	// session ID this client is connected to
	SessionID string

	// user ID (empty for anonymous users)
	UserID string

	// display name for this client
	DisplayName string

	// role in the session (host, co-author, viewer)
	Role string

	// whether this client has an authenticated user account
	IsAuthenticated bool

	// webSocket connection
	conn *websocket.Conn

	// hub reference for message broadcasting
	hub *Hub

	// buffered channel of outbound messages
	send chan []byte

	// mutex for thread-safe operations
	mu sync.RWMutex

	// flag indicating if client is closed
	closed bool
}

// creates a new webSocket client
func NewClient(id, sessionID, userID, displayName, role string, isAuthenticated bool, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:              id,
		SessionID:       sessionID,
		UserID:          userID,
		DisplayName:     displayName,
		Role:            role,
		IsAuthenticated: isAuthenticated,
		conn:            conn,
		hub:             hub,
		send:            make(chan []byte, 256),
		closed:          false,
	}
}

// reads messages from the webSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// parse the message
		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			c.SendError("INVALID_MESSAGE", "Invalid message format", err.Error())
			continue
		}

		// set session ID, client ID, and user ID from client
		msg.SessionID = c.SessionID
		msg.ClientID = c.ID
		msg.UserID = c.UserID
		msg.Timestamp = time.Now()

		// forward to hub for processing
		c.hub.Broadcast <- &msg
	}
}

// writes messages from the hub to the webSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// add queued messages to the current webSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sends a message to this client
func (c *Client) Send(msg *Message) (err error) {
	// recover from panic if channel is closed
	defer func() {
		if r := recover(); r != nil {
			err = ErrConnectionClosed
		}
	}()

	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrConnectionClosed
	}
	c.mu.RUnlock()

	messageBytes, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		return marshalErr
	}

	select {
	case c.send <- messageBytes:
		return nil
	default:
		// channel is full, close the client
		c.Close()
		return ErrConnectionClosed
	}
}

// sends an error message to the client
func (c *Client) SendError(code, message, details string) {
	errorMsg, err := NewMessage(TypeError, c.SessionID, c.UserID, ErrorPayload{
		Code:    code,
		Message: message,
		Details: details,
	})
	if err != nil {
		log.Printf("Failed to create error message: %v", err)
		return
	}

	c.Send(errorMsg)
}

// closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

// checks if the client is closed
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.closed
}

// checks if the client has write permissions
func (c *Client) CanWrite() bool {
	return c.Role == "host" || c.Role == "co-author"
}
