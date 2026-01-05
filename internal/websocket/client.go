package websocket

import (
	"encoding/json"
	"time"

	"github.com/algrv/server/algorave/users"
	"github.com/algrv/server/internal/errors"
	"github.com/algrv/server/internal/logger"
	"github.com/gorilla/websocket"
)

// creates a new webSocket client connection
func NewClient(id, sessionID, userID, displayName, role, tier, ipAddress, initialCode string, initialConversationHistory []SessionStateMessage, initialChatHistory []SessionStateChatMessage, isAuthenticated bool, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:                         id,
		SessionID:                  sessionID,
		UserID:                     userID,
		DisplayName:                displayName,
		Role:                       role,
		Tier:                       tier,
		IsAuthenticated:            isAuthenticated,
		IPAddress:                  ipAddress,
		InitialCode:                initialCode,
		InitialConversationHistory: initialConversationHistory,
		InitialChatHistory:         initialChatHistory,
		conn:                       conn,
		hub:                        hub,
		send:                       make(chan []byte, 256),
		closed:                     false,
		codeUpdateTimestamps:       make([]time.Time, 0, maxCodeUpdatesPerSecond),
		agentRequestTimestamps:     make([]time.Time, 0, maxAgentRequestsPerMinute),
		chatMessageTimestamps:      make([]time.Time, 0, maxChatMessagesPerMinute),
	}
}

// returns the per-minute agent request limit based on user tier
func (c *Client) getAgentRequestLimit() int {
	switch c.Tier {
	case "payg":
		return users.MinuteLimitPAYG
	case "byok":
		return users.MinuteLimitBYOK
	default:
		return users.MinuteLimitDefault
	}
}

// reads messages from the webSocket connection to the hub for processing
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close() //nolint:errcheck,gosec // G104: defer cleanup
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait)) //nolint:errcheck,gosec // G104: websocket setup
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait)) //nolint:errcheck,gosec // G104: pong handler
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Warn("websocket error",
					"client_id", c.ID,
					"session_id", c.SessionID,
					"error", err,
				)
			}

			break
		}

		// parse the message
		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			logger.ErrorErr(err, "failed to unmarshal message",
				"client_id", c.ID,
				"session_id", c.SessionID,
			)

			c.SendError("bad_request", "invalid message format", err.Error())
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

// writes messages from the hub to the webSocket connection for sending to the client
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.conn.Close() //nolint:errcheck,gosec // G104: defer cleanup
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint:errcheck,gosec // G104: websocket timing

			if !ok {
				// hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{}) //nolint:errcheck,gosec // G104: close message
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			w.Write(message) //nolint:errcheck,gosec // G104: websocket write

			// add queued messages to the current webSocket message
			n := len(c.send)

			for range n {
				w.Write([]byte{'\n'}) //nolint:errcheck,gosec // G104: websocket write
				w.Write(<-c.send)     //nolint:errcheck,gosec // G104: websocket write
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint:errcheck,gosec // G104: websocket ping timing

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sends a message to the client
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
		// channel is full, send error directly to websocket before closing
		c.sendBufferOverflowError()
		c.Close()
		return ErrConnectionClosed
	}
}

// sends buffer overflow error directly to websocket (bypassing the full channel)
func (c *Client) sendBufferOverflowError() {
	errorMsg, err := NewMessage(TypeError, c.SessionID, c.UserID, map[string]string{
		"error":   "buffer_overflow",
		"message": "message buffer full, connection will be closed",
		"details": "too many messages queued, please reconnect",
	})
	if err != nil {
		return
	}

	errorBytes, err := json.Marshal(errorMsg)
	if err != nil {
		return
	}

	// write directly to websocket with short deadline
	c.conn.SetWriteDeadline(time.Now().Add(2 * time.Second)) //nolint:errcheck,gosec
	c.conn.WriteMessage(websocket.TextMessage, errorBytes)   //nolint:errcheck,gosec
}

// sends an error message to the client
func (c *Client) SendError(code, message, details string) {
	// sanitize error details in production
	sanitizedDetails := details

	if details != "" {
		sanitizedDetails = sanitizeErrorString(details)
	}

	errorMsg, err := NewMessage(TypeError, c.SessionID, c.UserID, errors.ErrorResponse{
		Error:   code,
		Message: message,
		Details: sanitizedDetails,
	})
	if err != nil {
		logger.ErrorErr(err, "failed to create error message",
			"client_id", c.ID,
			"session_id", c.SessionID,
			"error_code", code,
		)
		return
	}

	c.Send(errorMsg) //nolint:errcheck,gosec // G104: best effort error notification
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

// sets the current strudel ID (thread-safe)
func (c *Client) SetCurrentStrudelID(strudelID *string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CurrentStrudelID = strudelID
}

// gets the current strudel ID (thread-safe)
func (c *Client) GetCurrentStrudelID() *string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CurrentStrudelID
}

// checks if the client can send a code update
func (c *Client) checkCodeUpdateRateLimit() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	oneSecondAgo := now.Add(-1 * time.Second)

	// remove timestamps older than 1 second
	validTimestamps := make([]time.Time, 0, maxCodeUpdatesPerSecond)

	for _, ts := range c.codeUpdateTimestamps {
		if ts.After(oneSecondAgo) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	c.codeUpdateTimestamps = validTimestamps

	// check if we've exceeded the limit
	if len(c.codeUpdateTimestamps) >= maxCodeUpdatesPerSecond {
		return false
	}

	// add current timestamp
	c.codeUpdateTimestamps = append(c.codeUpdateTimestamps, now)
	return true
}

// checks if the client can send an agent request (tier-based limits)
func (c *Client) checkAgentRequestRateLimit() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	limit := c.getAgentRequestLimit()
	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)

	// remove timestamps older than 1 minute
	validTimestamps := make([]time.Time, 0, limit)
	for _, ts := range c.agentRequestTimestamps {
		if ts.After(oneMinuteAgo) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	c.agentRequestTimestamps = validTimestamps

	// check if we've exceeded the limit
	if len(c.agentRequestTimestamps) >= limit {
		return false
	}

	// add current timestamp
	c.agentRequestTimestamps = append(c.agentRequestTimestamps, now)
	return true
}

// checks if the client can send a chat message
func (c *Client) checkChatRateLimit() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)

	// remove timestamps older than 1 minute
	validTimestamps := make([]time.Time, 0, maxChatMessagesPerMinute)
	for _, ts := range c.chatMessageTimestamps {
		if ts.After(oneMinuteAgo) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	c.chatMessageTimestamps = validTimestamps

	// check if we've exceeded the limit
	if len(c.chatMessageTimestamps) >= maxChatMessagesPerMinute {
		return false
	}

	// add current timestamp
	c.chatMessageTimestamps = append(c.chatMessageTimestamps, now)
	return true
}

// returns current rate limit status for agent requests
func (c *Client) GetAgentRateLimitStatus() *RateLimit {
	c.mu.RLock()
	defer c.mu.RUnlock()

	limit := c.getAgentRequestLimit()
	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)

	// count valid timestamps
	validCount := 0
	var oldestTimestamp time.Time

	for _, ts := range c.agentRequestTimestamps {
		if ts.After(oneMinuteAgo) {
			validCount++
			if oldestTimestamp.IsZero() || ts.Before(oldestTimestamp) {
				oldestTimestamp = ts
			}
		}
	}

	remaining := max(limit-validCount, 0)

	// calculate seconds until oldest request expires (resets quota)
	resetSeconds := 60
	if !oldestTimestamp.IsZero() {
		resetSeconds = int(oldestTimestamp.Add(time.Minute).Sub(now).Seconds())
		if resetSeconds < 0 {
			resetSeconds = 0
		}
	}

	return &RateLimit{
		RequestsRemaining: remaining,
		RequestsLimit:     limit,
		ResetSeconds:      resetSeconds,
	}
}
