package strudels

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/algorave/server/internal/agent"
	"github.com/jackc/pgx/v5/pgxpool"
)

// handles strudel database operations
type Repository struct {
	db *pgxpool.Pool
}

// represents a saved strudel pattern with code and metadata
type Strudel struct {
	ID                  string              `json:"id"`
	UserID              string              `json:"user_id"`
	Title               string              `json:"title"`
	Code                string              `json:"code"`
	IsPublic            bool                `json:"is_public"`
	Description         string              `json:"description,omitempty"`
	Tags                []string            `json:"tags,omitempty"`
	Categories          []string            `json:"categories,omitempty"`
	ConversationHistory ConversationHistory `json:"conversation_history,omitempty"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
}

// represents the conversation history as a JSON array
type ConversationHistory []agent.Message

// Value implements the driver.Valuer interface for ConversationHistory
func (ch ConversationHistory) Value() (driver.Value, error) {
	if ch == nil {
		return json.Marshal([]agent.Message{})
	}

	return json.Marshal(ch)
}

// Scan implements the sql.Scanner interface for ConversationHistory
func (ch *ConversationHistory) Scan(value interface{}) error {
	if value == nil {
		*ch = []agent.Message{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, ch)
}

// contains data for creating a new strudel
type CreateStrudelRequest struct {
	Title               string              `json:"title" binding:"required,max=200"`
	Code                string              `json:"code" binding:"required,max=1048576"` // 1MB limit
	IsPublic            bool                `json:"is_public"`
	Description         string              `json:"description,omitempty" binding:"max=2000"`
	Tags                []string            `json:"tags,omitempty" binding:"max=20,dive,max=50"`       // max 20 tags, each max 50 chars
	Categories          []string            `json:"categories,omitempty" binding:"max=10,dive,max=50"` // max 10 categories, each max 50 chars
	ConversationHistory ConversationHistory `json:"conversation_history,omitempty" binding:"max=100"`  // max 100 messages
}

// contains data for updating a strudel
type UpdateStrudelRequest struct {
	Title               *string             `json:"title,omitempty" binding:"omitempty,max=200"`
	Code                *string             `json:"code,omitempty" binding:"omitempty,max=1048576"` // 1MB limit
	IsPublic            *bool               `json:"is_public,omitempty"`
	Description         *string             `json:"description,omitempty" binding:"omitempty,max=2000"`
	Tags                []string            `json:"tags,omitempty" binding:"max=20,dive,max=50"`
	Categories          []string            `json:"categories,omitempty" binding:"max=10,dive,max=50"`
	ConversationHistory ConversationHistory `json:"conversation_history,omitempty" binding:"max=100"`
}
