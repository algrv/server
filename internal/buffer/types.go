package buffer

import "time"

// represents a message waiting to be flushed to Postgres
type BufferedMessage struct {
	SessionID      string    `json:"session_id"`
	UserID         string    `json:"user_id,omitempty"`
	Role           string    `json:"role"`
	MessageType    string    `json:"message_type"`
	Content        string    `json:"content"`
	IsActionable   bool      `json:"is_actionable"`
	IsCodeResponse bool      `json:"is_code_response"`
	DisplayName    string    `json:"display_name,omitempty"`
	AvatarURL      string    `json:"avatar_url,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// redis key patterns
const (
	// session:{sessionID}:code - stores current code as string
	keySessionCode = "session:%s:code"

	// session:{sessionID}:messages - stores messages as JSON list
	keySessionMessages = "session:%s:messages"

	// dirty_sessions:code - set of session IDs with unflushed code changes
	keyDirtySessionsCode = "dirty_sessions:code"

	// dirty_sessions:messages - set of session IDs with unflushed messages
	keyDirtySessionsMessages = "dirty_sessions:messages"
)
