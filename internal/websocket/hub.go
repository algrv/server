package websocket

import (
	"time"

	"codeberg.org/algorave/server/internal/logger"
)

func NewHub() *Hub {
	return &Hub{
		sessions:         make(map[string]map[string]*Client),
		Register:         make(chan *Client),
		Unregister:       make(chan *Client),
		Broadcast:        make(chan *Message, 256),
		handlers:         make(map[string]MessageHandler),
		running:          false,
		shutdown:         make(chan struct{}),
		userConnections:  make(map[string]int),
		ipConnections:    make(map[string]int),
		sessionSequences: make(map[string]uint64),
	}
}

// registers a handler for a specific message type
func (h *Hub) RegisterHandler(messageType string, handler MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[messageType] = handler
}

// sets callback to be called when a client disconnects
func (h *Hub) OnClientDisconnect(callback func(client *Client)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onClientDisconnect = callback
}

// sets callback to be called after a client is registered and session_state is sent
func (h *Hub) OnClientRegistered(callback func(client *Client)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onClientRegistered = callback
}

// starts the hub's main loop
func (h *Hub) Run() {
	h.running = true
	defer func() {
		h.running = false
	}()

	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case message := <-h.Broadcast:
			h.handleMessage(message)

		case <-h.shutdown:
			h.closeAllConnections()
			return
		}
	}
}

// registerClient adds a client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.sessions[client.SessionID] == nil {
		h.sessions[client.SessionID] = make(map[string]*Client)
	}

	h.sessions[client.SessionID][client.ID] = client

	if client.UserID != "" {
		h.userConnections[client.UserID]++
	}

	logger.Info("client registered",
		"client_id", client.ID,
		"session_id", client.SessionID,
		"role", client.Role,
		"display_name", client.DisplayName,
		"user_id", client.UserID,
	)

	// build participants list from connected clients (including the new client)
	participants := make([]SessionStateParticipant, 0)

	for _, c := range h.sessions[client.SessionID] {
		participants = append(participants, SessionStateParticipant{
			UserID:      c.UserID,
			DisplayName: c.DisplayName,
			Role:        c.Role,
		})
	}

	// send session_state to connecting client
	sessionStateMsg, err := NewMessage(TypeSessionState, client.SessionID, client.UserID, SessionStatePayload{
		Code:         client.InitialCode,
		YourRole:     client.Role,
		Participants: participants,
		ChatHistory:  client.InitialChatHistory,
	})
	if err == nil {
		if sendErr := client.Send(sessionStateMsg); sendErr != nil {
			logger.ErrorErr(sendErr, "failed to send session state",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
		}
	}

	// broadcast user_joined to other clients in the session
	userJoinedMsg, err := NewMessage(TypeUserJoined, client.SessionID, client.UserID, UserJoinedPayload{
		UserID:      client.UserID,
		DisplayName: client.DisplayName,
		Role:        client.Role,
	})
	if err == nil {
		h.broadcastToSession(client.SessionID, userJoinedMsg, client.ID)
	}

	// call registered callback (e.g., to send paste lock status)
	if h.onClientRegistered != nil {
		go h.onClientRegistered(client)
	}
}

// removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()

	// capture callback reference under lock
	callback := h.onClientDisconnect

	sessionClients, exists := h.sessions[client.SessionID]
	if !exists {
		h.mu.Unlock()
		return
	}

	if _, exists := sessionClients[client.ID]; !exists {
		h.mu.Unlock()
		return
	}

	delete(sessionClients, client.ID)
	client.Close()

	if client.UserID != "" {
		h.userConnections[client.UserID]--

		if h.userConnections[client.UserID] <= 0 {
			delete(h.userConnections, client.UserID)
		}
	}

	if client.IPAddress != "" {
		h.ipConnections[client.IPAddress]--

		if h.ipConnections[client.IPAddress] <= 0 {
			delete(h.ipConnections, client.IPAddress)
		}
	}

	logger.Info("client unregistered",
		"client_id", client.ID,
		"session_id", client.SessionID,
	)

	if len(sessionClients) == 0 {
		delete(h.sessions, client.SessionID)
		delete(h.sessionSequences, client.SessionID)

		logger.Info("session has no more clients, removed",
			"session_id", client.SessionID,
		)
	} else {
		userLeftMsg, err := NewMessage(TypeUserLeft, client.SessionID, client.UserID, UserLeftPayload{
			UserID:      client.UserID,
			DisplayName: client.DisplayName,
		})
		if err == nil {
			h.broadcastToSession(client.SessionID, userLeftMsg, "")
		}
	}

	h.mu.Unlock()

	// call disconnect callback outside lock (may do DB operations)
	if callback != nil {
		callback(client)
	}
}

// processes an incoming message
func (h *Hub) handleMessage(msg *Message) {
	h.mu.RLock()

	sessionClients, exists := h.sessions[msg.SessionID]
	if !exists {
		h.mu.RUnlock()
		logger.Warn("session not found for message",
			"session_id", msg.SessionID,
			"message_type", msg.Type,
		)
		return
	}

	sender, exists := sessionClients[msg.ClientID]
	h.mu.RUnlock()

	if !exists {
		logger.Warn("sender client not found for message",
			"client_id", msg.ClientID,
			"session_id", msg.SessionID,
			"message_type", msg.Type,
		)
		return
	}

	h.mu.RLock()
	handler, exists := h.handlers[msg.Type]
	h.mu.RUnlock()

	if exists {
		// run handler asynchronously to avoid blocking the hub
		go func() {
			if err := handler(h, sender, msg); err != nil {
				logger.ErrorErr(err, "handler error",
					"message_type", msg.Type,
					"client_id", sender.ID,
					"session_id", msg.SessionID,
				)

				sender.SendError("server_error", "failed to process message", err.Error())
			}
		}()
	} else {
		// reject unhandled message types
		logger.Warn("unhandled message type received",
			"message_type", msg.Type,
			"client_id", sender.ID,
			"session_id", msg.SessionID,
		)

		sender.SendError("bad_request", "unsupported message type", "message type not recognized")
	}
}

// sends a message to all clients in a session
func (h *Hub) BroadcastToSession(sessionID string, msg *Message, excludeClientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.broadcastToSession(sessionID, msg, excludeClientID)
}

// sends a message only to clients with write permissions (host and co-authors)
func (h *Hub) BroadcastToWriters(sessionID string, msg *Message, excludeClientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.broadcastToWriters(sessionID, msg, excludeClientID)
}

// the internal broadcast to writers function (must be called with lock held)
func (h *Hub) broadcastToWriters(sessionID string, msg *Message, excludeClientID string) {
	sessionClients, exists := h.sessions[sessionID]
	if !exists {
		return
	}

	// assign sequence number to message
	h.sessionSequences[sessionID]++
	msg.Sequence = h.sessionSequences[sessionID]

	for clientID, client := range sessionClients {
		if clientID == excludeClientID {
			continue
		}

		// only send to clients with write permissions
		if !client.CanWrite() {
			continue
		}

		if err := client.Send(msg); err != nil {
			logger.ErrorErr(err, "failed to send message to client",
				"client_id", clientID,
				"session_id", sessionID,
			)
		}
	}
}

// the internal broadcast function (must be called with lock held)
func (h *Hub) broadcastToSession(sessionID string, msg *Message, excludeClientID string) {
	sessionClients, exists := h.sessions[sessionID]
	if !exists {
		return
	}

	// assign sequence number to message
	h.sessionSequences[sessionID]++
	msg.Sequence = h.sessionSequences[sessionID]

	for clientID, client := range sessionClients {
		if clientID == excludeClientID {
			continue
		}

		if err := client.Send(msg); err != nil {
			logger.ErrorErr(err, "failed to send message to client",
				"client_id", clientID,
				"session_id", sessionID,
			)
		}
	}
}

// returns all clients in a session
func (h *Hub) GetSessionClients(sessionID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sessionClients, exists := h.sessions[sessionID]
	if !exists {
		return []*Client{}
	}

	clients := make([]*Client, 0, len(sessionClients))

	for _, client := range sessionClients {
		clients = append(clients, client)
	}

	return clients
}

// returns the number of clients in a session
func (h *Hub) GetClientCount(sessionID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sessionClients, exists := h.sessions[sessionID]
	if !exists {
		return 0
	}

	return len(sessionClients)
}

func (h *Hub) GetSessionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}

// IsSessionActive checks if a session has any active WebSocket connections
func (h *Hub) IsSessionActive(sessionID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	sessionClients, exists := h.sessions[sessionID]
	return exists && len(sessionClients) > 0
}

func (h *Hub) Shutdown() {
	if h.running {
		close(h.shutdown)
	}
}

func (h *Hub) closeAllConnections() {
	h.mu.Lock()

	logger.Info("notifying clients of server shutdown")

	// send shutdown notification to all clients first
	for sessionID, sessionClients := range h.sessions {
		shutdownMsg, err := NewMessage(TypeServerShutdown, sessionID, "", ServerShutdownPayload{
			Reason: "server is shutting down for maintenance",
		})
		if err != nil {
			logger.ErrorErr(err, "failed to create shutdown message")
			continue
		}

		for _, client := range sessionClients {
			if err := client.Send(shutdownMsg); err != nil {
				logger.ErrorErr(err, "failed to send shutdown notification",
					"client_id", client.ID,
					"session_id", sessionID,
				)
			}
		}
	}

	h.mu.Unlock()

	// give clients time to receive the shutdown message
	time.Sleep(500 * time.Millisecond)

	h.mu.Lock()
	defer h.mu.Unlock()

	logger.Info("closing all websocket connections")

	for sessionID, sessionClients := range h.sessions {
		for clientID, client := range sessionClients {
			client.Close()
			logger.Debug("closed client",
				"client_id", clientID,
				"session_id", sessionID,
			)
		}
	}

	// clear all sessions and connection tracking
	h.sessions = make(map[string]map[string]*Client)
	h.userConnections = make(map[string]int)
	h.ipConnections = make(map[string]int)
	h.sessionSequences = make(map[string]uint64)
}

// checks if a new connection should be allowed based on limits
func (h *Hub) CanAcceptConnection(userID, ipAddress string) (bool, string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// check per-user limit (only for authenticated users)
	if userID != "" {
		count := h.userConnections[userID]
		if count >= maxConnectionsPerUser {
			return false, "Maximum connections per user exceeded"
		}
	}

	// check per-IP limit
	count := h.ipConnections[ipAddress]
	if count >= maxConnectionsPerIP {
		return false, "Maximum connections per IP address exceeded"
	}

	return true, ""
}

// increments the connection count for an IP address
func (h *Hub) TrackIPConnection(ipAddress string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ipConnections[ipAddress]++
}

// decrements the connection count for an IP address
func (h *Hub) UntrackIPConnection(ipAddress string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ipConnections[ipAddress]--

	if h.ipConnections[ipAddress] <= 0 {
		delete(h.ipConnections, ipAddress)
	}
}

// broadcasts session_ended to all clients and closes their connections
func (h *Hub) EndSession(sessionID string, reason string) {
	h.mu.Lock()

	sessionClients, exists := h.sessions[sessionID]
	if !exists {
		h.mu.Unlock()
		return
	}

	logger.Info("ending session, notifying clients",
		"session_id", sessionID,
		"client_count", len(sessionClients),
	)

	// send session_ended notification to all clients
	sessionEndedMsg, err := NewMessage(TypeSessionEnded, sessionID, "", SessionEndedPayload{
		Reason: reason,
	})
	if err != nil {
		logger.ErrorErr(err, "failed to create session_ended message",
			"session_id", sessionID,
		)
		h.mu.Unlock()
		return
	}

	for _, client := range sessionClients {
		if err := client.Send(sessionEndedMsg); err != nil {
			logger.ErrorErr(err, "failed to send session_ended notification",
				"client_id", client.ID,
				"session_id", sessionID,
			)
		}
	}

	h.mu.Unlock()

	// give clients time to receive the message
	time.Sleep(100 * time.Millisecond)

	h.mu.Lock()
	defer h.mu.Unlock()

	// close all connections for this session
	sessionClients, exists = h.sessions[sessionID]
	if !exists {
		return
	}

	for clientID, client := range sessionClients {
		// update connection tracking
		if client.UserID != "" {
			h.userConnections[client.UserID]--
			if h.userConnections[client.UserID] <= 0 {
				delete(h.userConnections, client.UserID)
			}
		}
		if client.IPAddress != "" {
			h.ipConnections[client.IPAddress]--
			if h.ipConnections[client.IPAddress] <= 0 {
				delete(h.ipConnections, client.IPAddress)
			}
		}

		client.Close()
		logger.Debug("closed client due to session end",
			"client_id", clientID,
			"session_id", sessionID,
		)
	}

	// remove session from hub
	delete(h.sessions, sessionID)
	delete(h.sessionSequences, sessionID)

	logger.Info("session ended and removed",
		"session_id", sessionID,
	)
}
