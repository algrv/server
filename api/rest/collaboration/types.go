package collaboration

import "time"

// CreateSessionRequest is the request to create a new session
type CreateSessionRequest struct {
	Title string `json:"title" binding:"required"`
	Code  string `json:"code"`
}

// CreateSessionResponse is the response after creating a session
type CreateSessionResponse struct {
	ID           string     `json:"id"`
	HostUserID   string     `json:"host_user_id"`
	Title        string     `json:"title"`
	Code         string     `json:"code"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	LastActivity time.Time  `json:"last_activity"`
}

// SessionResponse represents a session in API responses
type SessionResponse struct {
	ID           string              `json:"id"`
	HostUserID   string              `json:"host_user_id"`
	Title        string              `json:"title"`
	Code         string              `json:"code"`
	IsActive     bool                `json:"is_active"`
	CreatedAt    time.Time           `json:"created_at"`
	EndedAt      *time.Time          `json:"ended_at,omitempty"`
	LastActivity time.Time           `json:"last_activity"`
	Participants []ParticipantResponse `json:"participants,omitempty"`
}

// ParticipantResponse represents a participant in API responses
type ParticipantResponse struct {
	ID          string     `json:"id"`
	UserID      *string    `json:"user_id,omitempty"`
	DisplayName *string    `json:"display_name,omitempty"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	JoinedAt    time.Time  `json:"joined_at"`
	LeftAt      *time.Time `json:"left_at,omitempty"`
}

// UpdateSessionCodeRequest is the request to update session code
type UpdateSessionCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

// CreateInviteTokenRequest is the request to create an invite token
type CreateInviteTokenRequest struct {
	Role      string     `json:"role" binding:"required,oneof=co-author viewer"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// InviteTokenResponse represents an invite token in API responses
type InviteTokenResponse struct {
	ID        string     `json:"id"`
	SessionID string     `json:"session_id"`
	Token     string     `json:"token"`
	Role      string     `json:"role"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	UsesCount int        `json:"uses_count"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// JoinSessionRequest is the request to join a session via invite token
type JoinSessionRequest struct {
	InviteToken string `json:"invite_token" binding:"required"`
	DisplayName string `json:"display_name,omitempty"`
}

// JoinSessionResponse is the response after joining a session
type JoinSessionResponse struct {
	SessionID   string `json:"session_id"`
	Role        string `json:"role"`
	DisplayName string `json:"display_name"`
}
