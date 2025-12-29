package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
)

// creates a new webSocket client
func NewWSClient() *WSClient {
	endpoint := os.Getenv("ALGORAVE_WS_ENDPOINT")
	if endpoint == "" {
		endpoint = "ws://localhost:8080/api/v1/ws"
	}

	return &WSClient{
		endpoint: endpoint,
		pending:  make(map[string]chan wsMessage),
	}
}

// Connect establishes the WebSocket connection
func (c *WSClient) Connect() error {
	c.mu.Lock()

	if c.connected {
		c.mu.Unlock()
		return nil
	}

	conn, _, err := websocket.DefaultDialer.Dial(c.endpoint, nil)
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn

	// set up ping/pong handlers to keep theconnection alive
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// set initial read deadline
	conn.SetReadDeadline(time.Now().Add(pongWait))

	// read the welcome message to get session ID
	var welcomeMsg wsMessage
	if err := conn.ReadJSON(&welcomeMsg); err != nil {
		conn.Close()
		c.mu.Unlock()
		return fmt.Errorf("failed to read welcome: %w", err)
	}

	c.sessionID = welcomeMsg.SessionID
	c.connected = true

	// start the read pump in a goroutine
	go c.readPump()

	// start the ping pump to keep connection alive
	go c.pingPump()

	c.mu.Unlock()
	return nil
}

// sends periodic pings to keep the connection alive
func (c *WSClient) pingPump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		<-ticker.C
		c.mu.Lock()

		if !c.connected || c.conn == nil {
			c.mu.Unlock()
			return
		}

		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			c.mu.Unlock()
			return
		}

		c.mu.Unlock()
	}
}

// continuously reads messages and routes them to pending requests
func (c *WSClient) readPump() {
	defer func() {
		c.mu.Lock()
		c.connected = false
		if c.conn != nil {
			c.conn.Close()
		}
		c.mu.Unlock()
	}()

	for {
		// reset read deadline on each successful read
		c.conn.SetReadDeadline(time.Now().Add(pongWait))

		var msg wsMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			return
		}

		// route the message based on type
		switch msg.Type {
		case typeAgentResponse, typeError:
			c.pendingMu.Lock()
			// send to all pending requests (there should only be one)
			for id, ch := range c.pending {
				select {
				case ch <- msg:
				default:
				}
				delete(c.pending, id)
			}
			c.pendingMu.Unlock()

		case typeAgentRequest:
			// ignore broadcast of own request
			continue

		default:
			continue
		}
	}
}

// returns whether the client is connected
func (c *WSClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// closes the webSocket connection
func (c *WSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connected = false
}

// sends an agent request and waits for the response
func (c *WSClient) SendAgentRequest(ctx context.Context, userQuery, editorState string, conversationHistory []MessageModel) (*AgentResponseMsg, error) {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return nil, fmt.Errorf("not connected")
	}

	conn := c.conn
	sessionID := c.sessionID
	c.mu.Unlock()

	// create response channel
	responseCh := make(chan wsMessage, 1)
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())

	c.pendingMu.Lock()
	c.pending[requestID] = responseCh
	c.pendingMu.Unlock()

	// clean up on exit
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, requestID)
		c.pendingMu.Unlock()
	}()

	// build agent request payload
	payload := agentRequestPayload{
		UserQuery:           userQuery,
		EditorState:         editorState,
		ConversationHistory: conversationHistory,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// send agent_request message
	reqMsg := wsMessage{
		Type:      typeAgentRequest,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload:   payloadBytes,
	}

	c.mu.Lock()
	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	err = conn.WriteJSON(reqMsg)
	c.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// wait for response with timeout
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case respMsg := <-responseCh:
		return c.handleResponse(userQuery, respMsg)
	}
}

// processes the response message
func (c *WSClient) handleResponse(userQuery string, respMsg wsMessage) (*AgentResponseMsg, error) {
	switch respMsg.Type {
	case typeAgentResponse:
		var resp agentResponsePayload
		if err := json.Unmarshal(respMsg.Payload, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		code, metadata := formatAgentResponse(resp)
		return &AgentResponseMsg{
			userQuery: userQuery,
			code:      code,
			metadata:  metadata,
			questions: resp.ClarifyingQuestions,
		}, nil

	case typeError:
		var errResp errorPayload
		if err := json.Unmarshal(respMsg.Payload, &errResp); err != nil {
			return nil, fmt.Errorf("failed to parse error: %w", err)
		}

		return nil, fmt.Errorf("%s: %s", errResp.Error, errResp.Message)

	default:
		return nil, fmt.Errorf("unexpected message type: %s", respMsg.Type)
	}
}

// returns a tea.Cmd that connects to the webSocket server
func (c *WSClient) ConnectCmd() tea.Cmd {
	return func() tea.Msg {
		if err := c.Connect(); err != nil {
			return WSConnectErrorMsg{err: err}
		}

		return WSConnectedMsg{sessionID: c.sessionID}
	}
}
