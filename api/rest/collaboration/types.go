package collaboration

import (
	"time"

	"github.com/algrv/server/algorave/sessions"
	"github.com/algrv/server/api/rest/pagination"
)

// allows ending WebSocket sessions
type SessionEnder interface {
	EndSession(sessionID string, reason string)
}

type CreateSessionRequest struct {
	Title          string `json:"title" binding:"required,max=200"`
	Code           string `json:"code" binding:"max=1048576"` // 1MB limit
	IsDiscoverable *bool  `json:"is_discoverable,omitempty"`  // optional, defaults to false
}

type CreateSessionResponse struct {
	ID             string    `json:"id"`
	HostUserID     string    `json:"host_user_id"`
	Title          string    `json:"title"`
	Code           string    `json:"code"`
	IsActive       bool      `json:"is_active"`
	IsDiscoverable bool      `json:"is_discoverable"`
	CreatedAt      time.Time `json:"created_at"`
	LastActivity   time.Time `json:"last_activity"`
}

type SessionResponse struct {
	ID             string                `json:"id"`
	HostUserID     string                `json:"host_user_id"`
	Title          string                `json:"title"`
	Code           string                `json:"code"`
	IsActive       bool                  `json:"is_active"`
	IsDiscoverable bool                  `json:"is_discoverable"`
	CreatedAt      time.Time             `json:"created_at"`
	EndedAt        *time.Time            `json:"ended_at,omitempty"`
	LastActivity   time.Time             `json:"last_activity"`
	Participants   []ParticipantResponse `json:"participants,omitempty"`
}

type ParticipantResponse struct {
	ID          string     `json:"id"`
	UserID      *string    `json:"user_id,omitempty"`
	DisplayName *string    `json:"display_name,omitempty"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	JoinedAt    time.Time  `json:"joined_at"`
	LeftAt      *time.Time `json:"left_at,omitempty"`
}

type UpdateSessionCodeRequest struct {
	Code string `json:"code" binding:"required,max=1048576"` // 1MB limit
}

type CreateInviteTokenRequest struct {
	Role      string     `json:"role" binding:"required,oneof=co-author viewer"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

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

type JoinSessionRequest struct {
	InviteToken string `json:"invite_token" binding:"required"`
	DisplayName string `json:"display_name,omitempty" binding:"max=100"`
}

type JoinSessionResponse struct {
	SessionID   string `json:"session_id"`
	Role        string `json:"role"`
	DisplayName string `json:"display_name"`
}

// SessionsListResponse wraps a list of sessions
type SessionsListResponse struct {
	Sessions []SessionResponse `json:"sessions"`
}

// ParticipantsListResponse wraps a list of participants
type ParticipantsListResponse struct {
	Participants []ParticipantResponse `json:"participants"`
}

// InviteTokensListResponse wraps a list of invite tokens
type InviteTokensListResponse struct {
	Tokens []InviteTokenResponse `json:"tokens"`
}

// MessageResponse for simple success messages
type MessageResponse struct {
	Message string `json:"message"`
}

// UpdateSessionCodeResponse returned after updating session code
type UpdateSessionCodeResponse struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// UpdateRoleResponse returned after updating participant role
type UpdateRoleResponse struct {
	Message string `json:"message"`
	Role    string `json:"role"`
}

// MessagesResponse wraps chat messages
type MessagesResponse struct {
	Messages []*sessions.Message `json:"messages"`
}

// SetDiscoverableRequest for updating session discoverability
type SetDiscoverableRequest struct {
	IsDiscoverable bool `json:"is_discoverable"`
}

// LiveSessionResponse for public listing of discoverable sessions
type LiveSessionResponse struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	ParticipantCount int       `json:"participant_count"`
	IsMember         bool      `json:"is_member"`
	CreatedAt        time.Time `json:"created_at"`
	LastActivity     time.Time `json:"last_activity"`
}

// LiveSessionsListResponse wraps a list of live sessions with pagination
type LiveSessionsListResponse struct {
	Sessions   []LiveSessionResponse `json:"sessions"`
	Pagination pagination.Meta       `json:"pagination"`
}

// SoftEndSessionResponse returned after soft-ending a session
type SoftEndSessionResponse struct {
	Message            string `json:"message"`
	ParticipantsKicked int    `json:"participants_kicked"`
	InvitesRevoked     bool   `json:"invites_revoked"`
}

// IsLiveResponse for checking if a session is currently "live"
type IsLiveResponse struct {
	IsLive                bool `json:"is_live"`
	ParticipantCount      int  `json:"participant_count"`
	HasActiveInviteTokens bool `json:"has_active_invite_tokens"`
}
