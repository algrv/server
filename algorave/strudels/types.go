package strudels

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/algrv/server/internal/agent"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

type Strudel struct {
	ID                  string              `json:"id"`
	UserID              string              `json:"user_id"`
	Title               string              `json:"title"`
	Code                string              `json:"code"`
	IsPublic            bool                `json:"is_public"`
	UseInTraining       bool                `json:"-"` // admin-only, not exposed to users
	Description         string              `json:"description,omitempty"`
	Tags                []string            `json:"tags,omitempty"`
	Categories          []string            `json:"categories,omitempty"`
	ConversationHistory ConversationHistory `json:"conversation_history,omitempty"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
}

type ConversationHistory []agent.Message

func (ch ConversationHistory) Value() (driver.Value, error) {
	if len(ch) == 0 {
		return "[]", nil
	}

	bytes, err := json.Marshal(ch)
	if err != nil {
		return nil, err
	}

	return string(bytes), nil
}

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

type CreateStrudelRequest struct {
	Title               string              `json:"title" binding:"required,max=200"`
	Code                string              `json:"code" binding:"required,max=1048576"` // 1MB limit
	IsPublic            bool                `json:"is_public"`
	Description         string              `json:"description,omitempty" binding:"max=2000"`
	Tags                []string            `json:"tags,omitempty" binding:"max=20,dive,max=50"`       // max 20 tags, each max 50 chars
	Categories          []string            `json:"categories,omitempty" binding:"max=10,dive,max=50"` // max 10 categories, each max 50 chars
	ConversationHistory ConversationHistory `json:"conversation_history,omitempty" binding:"max=100"`  // max 100 messages
}

type UpdateStrudelRequest struct {
	Title               *string             `json:"title,omitempty" binding:"omitempty,max=200"`
	Code                *string             `json:"code,omitempty" binding:"omitempty,max=1048576"` // 1MB limit
	IsPublic            *bool               `json:"is_public,omitempty"`
	Description         *string             `json:"description,omitempty" binding:"omitempty,max=2000"`
	Tags                []string            `json:"tags,omitempty" binding:"max=20,dive,max=50"`
	Categories          []string            `json:"categories,omitempty" binding:"max=10,dive,max=50"`
	ConversationHistory ConversationHistory `json:"conversation_history,omitempty" binding:"max=100"`
}

type ListFilter struct {
	Search string   // search in title and description
	Tags   []string // filter by tags (any match)
}

// represents an AI conversation message for a saved strudel
type StrudelMessage struct {
	ID                  string    `json:"id"`
	StrudelID           string    `json:"strudel_id"`
	UserID              *string   `json:"user_id,omitempty"`
	Role                string    `json:"role"` // user, assistant
	Content             string    `json:"content"`
	IsActionable        bool      `json:"is_actionable"`
	IsCodeResponse      bool      `json:"is_code_response"`
	ClarifyingQuestions []string  `json:"clarifying_questions,omitempty"`
	DisplayName         *string   `json:"display_name,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

// contains data for adding a strudel message
type AddStrudelMessageRequest struct {
	StrudelID           string
	UserID              *string
	Role                string
	Content             string
	IsActionable        bool
	IsCodeResponse      bool
	ClarifyingQuestions []string
	DisplayName         string
}
