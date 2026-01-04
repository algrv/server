package sessions

import (
	"context"
	"time"
)

// message type constants for session messages (must match DB check constraint)
const (
	MessageTypeUserPrompt = "user_prompt"
	MessageTypeAIResponse = "ai_response"
	MessageTypeChat       = "chat"
)

// SystemUserID is the UUID for anonymous sessions (nil UUID pattern)
const SystemUserID = "00000000-0000-0000-0000-000000000000"

// repository interface for session database operations
type Repository interface {
	// session operations
	CreateSession(ctx context.Context, req *CreateSessionRequest) (*Session, error)
	CreateAnonymousSession(ctx context.Context) (*Session, error)
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	GetUserSessions(ctx context.Context, userID string, activeOnly bool) ([]*Session, error)
	ListDiscoverableSessions(ctx context.Context, limit, offset int) ([]*Session, int, error)
	UpdateSessionCode(ctx context.Context, sessionID, code string) error
	SetDiscoverable(ctx context.Context, sessionID string, isDiscoverable bool) error
	EndSession(ctx context.Context, sessionID string) error

	// authenticated participant operations
	AddAuthenticatedParticipant(ctx context.Context, sessionID, userID, displayName, role string) (*Participant, error)
	GetAuthenticatedParticipant(ctx context.Context, sessionID, userID string) (*Participant, error)
	MarkAuthenticatedParticipantLeft(ctx context.Context, participantID string) error
	GetParticipantByID(ctx context.Context, participantID string) (*CombinedParticipant, error)
	UpdateParticipantRole(ctx context.Context, participantID, role string) error

	// anonymous participant operations
	AddAnonymousParticipant(ctx context.Context, sessionID, displayName, role string) (*AnonymousParticipant, error)

	// combined participant operations
	ListAllParticipants(ctx context.Context, sessionID string) ([]*CombinedParticipant, error)
	RemoveParticipant(ctx context.Context, participantID string) error

	// invite token operations
	CreateInviteToken(ctx context.Context, req *CreateInviteTokenRequest) (*InviteToken, error)
	ListInviteTokens(ctx context.Context, sessionID string) ([]*InviteToken, error)
	ValidateInviteToken(ctx context.Context, token string) (*InviteToken, error)
	IncrementTokenUses(ctx context.Context, tokenID string) error
	RevokeInviteToken(ctx context.Context, tokenID string) error

	// message operations
	GetMessages(ctx context.Context, sessionID string, limit int) ([]*Message, error)
	CreateMessage(ctx context.Context, sessionID string, userID *string, role, messageType, content string, isActionable bool, displayName, avatarURL *string) (*Message, error)
	AddMessage(ctx context.Context, sessionID, userID, role, messageType, content string, isActionable bool, displayName, avatarURL string) (*Message, error)
	UpdateLastActivity(ctx context.Context, sessionID string) error
}

// represents a collaborative coding session
type Session struct {
	ID             string     `json:"id"`
	HostUserID     string     `json:"host_user_id"`
	Title          string     `json:"title"`
	Code           string     `json:"code"`
	IsActive       bool       `json:"is_active"`
	IsDiscoverable bool       `json:"is_discoverable"`
	CreatedAt      time.Time  `json:"created_at"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	LastActivity   time.Time  `json:"last_activity"`
}

// represents a user in a session
type Participant struct {
	ID          string     `json:"id"`
	SessionID   string     `json:"session_id"`
	UserID      string     `json:"user_id"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	JoinedAt    time.Time  `json:"joined_at"`
	LeftAt      *time.Time `json:"left_at,omitempty"`
}

// represents an anonymous user in a session
type AnonymousParticipant struct {
	ID          string     `json:"id"`
	SessionID   string     `json:"session_id"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	JoinedAt    time.Time  `json:"joined_at"`
	LeftAt      *time.Time `json:"left_at,omitempty"`
	ExpiresAt   time.Time  `json:"expires_at"`
}

// represents either an authenticated or anonymous participant
type CombinedParticipant struct {
	ID          string     `json:"id"`
	SessionID   string     `json:"session_id"`
	UserID      *string    `json:"user_id,omitempty"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	JoinedAt    time.Time  `json:"joined_at"`
	LeftAt      *time.Time `json:"left_at,omitempty"`
}

// represents a session invite token
type InviteToken struct {
	ID        string     `json:"id"`
	SessionID string     `json:"session_id"`
	Token     string     `json:"token"`
	Role      string     `json:"role"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	UsesCount int        `json:"uses_count"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// represents a chat message in a session
type Message struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"sessionID"`
	UserID       *string   `json:"userID,omitempty"`
	Role         string    `json:"role"`        // user, assistant
	MessageType  string    `json:"messageType"` // MessageTypeUserPrompt, MessageTypeAIResponse, MessageTypeChat
	Content      string    `json:"content"`
	IsActionable bool      `json:"isActionable"`
	DisplayName  *string   `json:"displayName,omitempty"`
	AvatarURL    *string   `json:"avatarUrl,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// contains data for creating a session
type CreateSessionRequest struct {
	HostUserID string `json:"host_user_id"`
	Title      string `json:"title"`
	Code       string `json:"code"`
}

// contains data for creating an invite token
type CreateInviteTokenRequest struct {
	SessionID string     `json:"session_id"`
	Role      string     `json:"role"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}
