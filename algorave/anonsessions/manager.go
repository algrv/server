package anonsessions

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/algorave/server/algorave/strudels"
)

const (
	SessionExpiryDuration = 24 * time.Hour
	CleanupInterval       = 1 * time.Hour
)

// creates a new anonymous session manager
func NewManager() *Manager {
	m := &Manager{
		sessions: make(map[string]*AnonymousSession),
		stopChan: make(chan struct{}),
	}

	// start cleanup goroutine
	go m.cleanupExpiredSessions()

	return m
}

// creates a new anonymous session
func (m *Manager) CreateSession() (*AnonymousSession, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &AnonymousSession{
		ID:                  sessionID,
		EditorState:         "",
		ConversationHistory: make(strudels.ConversationHistory, 0),
		CreatedAt:           time.Now(),
		LastActivity:        time.Now(),
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return session, nil
}

// retrieves a session by ID
func (m *Manager) GetSession(sessionID string) (*AnonymousSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// check if session has expired
	if time.Since(session.LastActivity) > SessionExpiryDuration {
		return nil, false
	}

	return session, true
}

// updates the code in a session
func (m *Manager) UpdateSessionCode(sessionID, code string) error {
	session, exists := m.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.UpdateEditorState(code)
	return nil
}

// adds a message to the session's conversation history
func (m *Manager) AddMessage(sessionID, role, content string) error {
	session, exists := m.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.AddMessage(role, content)
	return nil
}

// updates both the conversation history and editor state
func (m *Manager) UpdateSession(sessionID string, history strudels.ConversationHistory, code string) error {
	session, exists := m.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.ConversationHistory = history
	session.EditorState = code
	session.LastActivity = time.Now()

	return nil
}

// removes a session from memory
func (m *Manager) DeleteSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

// updates the last activity time for a session
func (m *Manager) TouchSession(sessionID string) error {
	session, exists := m.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.Touch()
	return nil
}

// returns the number of active sessions
func (m *Manager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// periodically removes expired sessions
func (m *Manager) cleanupExpiredSessions() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.removeExpiredSessions()
		case <-m.stopChan:
			return
		}
	}
}

// removes all expired sessions
func (m *Manager) removeExpiredSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, session := range m.sessions {
		if now.Sub(session.LastActivity) > SessionExpiryDuration {
			delete(m.sessions, id)
		}
	}
}

// stops the cleanup goroutine
func (m *Manager) Stop() {
	close(m.stopChan)
}

// generates a cryptographically secure random session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}
