package strudels

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"codeberg.org/algorave/server/internal/agent"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CCSignal represents Creative Commons Signals for AI training consent
type CCSignal string

const (
	CCSignalCredit    CCSignal = "cc-cr" // Credit: Allow AI use with attribution
	CCSignalDirect    CCSignal = "cc-dc" // Credit + Direct: Attribution + financial/in-kind support
	CCSignalEcosystem CCSignal = "cc-ec" // Credit + Ecosystem: Attribution + contribute to commons
	CCSignalOpen      CCSignal = "cc-op" // Credit + Open: Attribution + keep derivatives open
	CCSignalNoAI      CCSignal = "no-ai" // No AI: Explicitly opt-out of AI training
)

// Valid Creative Commons license values
const (
	LicenseCC0    = "CC0 1.0"
	LicenseBY     = "CC BY 4.0"
	LicenseBYSA   = "CC BY-SA 4.0"
	LicenseBYNC   = "CC BY-NC 4.0"
	LicenseBYNCSA = "CC BY-NC-SA 4.0"
	LicenseBYND   = "CC BY-ND 4.0"
	LicenseBYNCND = "CC BY-NC-ND 4.0"
)

// signalRestrictiveness defines the restrictiveness order (higher = more restrictive)
var signalRestrictiveness = map[CCSignal]int{
	"":                0, // NULL - no preference
	CCSignalCredit:    1,
	CCSignalDirect:    2,
	CCSignalEcosystem: 3,
	CCSignalOpen:      4,
	CCSignalNoAI:      5,
}

// MoreRestrictiveThan returns true if this signal is more restrictive than the other
func (s CCSignal) MoreRestrictiveThan(other CCSignal) bool {
	return signalRestrictiveness[s] > signalRestrictiveness[other]
}

// IsValid returns true if the signal is a valid CC Signal value
func (s CCSignal) IsValid() bool {
	switch s {
	case CCSignalCredit, CCSignalDirect, CCSignalEcosystem, CCSignalOpen, CCSignalNoAI:
		return true
	default:
		return false
	}
}

type Repository struct {
	db *pgxpool.Pool
}

type Strudel struct {
	ID                  string              `json:"id"`
	UserID              string              `json:"user_id"`
	Title               string              `json:"title"`
	Code                string              `json:"code"`
	IsPublic            bool                `json:"is_public"`
	License             *string             `json:"license,omitempty"`
	CCSignal            *CCSignal           `json:"cc_signal,omitempty"`
	UseInTraining       bool                `json:"-"` // admin-only, not exposed to users
	AIAssistCount       int                 `json:"ai_assist_count"`
	ForkedFrom          *string             `json:"forked_from,omitempty"`
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
	License             *string             `json:"license,omitempty"`
	CCSignal            *CCSignal           `json:"cc_signal,omitempty"`
	ForkedFrom          *string             `json:"forked_from,omitempty"`
	Description         string              `json:"description,omitempty" binding:"max=2000"`
	Tags                []string            `json:"tags,omitempty" binding:"max=20,dive,max=50"`       // max 20 tags, each max 50 chars
	Categories          []string            `json:"categories,omitempty" binding:"max=10,dive,max=50"` // max 10 categories, each max 50 chars
	ConversationHistory ConversationHistory `json:"conversation_history,omitempty" binding:"max=100"`  // max 100 messages
}

type UpdateStrudelRequest struct {
	Title               *string             `json:"title,omitempty" binding:"omitempty,max=200"`
	Code                *string             `json:"code,omitempty" binding:"omitempty,max=1048576"` // 1MB limit
	IsPublic            *bool               `json:"is_public,omitempty"`
	License             *string             `json:"license,omitempty"`
	CCSignal            *CCSignal           `json:"cc_signal,omitempty"`
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
	ID                  string             `json:"id"`
	StrudelID           string             `json:"strudel_id"`
	UserID              *string            `json:"user_id,omitempty"`
	Role                string             `json:"role"` // user, assistant
	Content             string             `json:"content"`
	IsActionable        bool               `json:"is_actionable"`
	IsCodeResponse      bool               `json:"is_code_response"`
	ClarifyingQuestions []string           `json:"clarifying_questions,omitempty"`
	StrudelReferences   []StrudelReference `json:"strudel_references,omitempty"`
	DocReferences       []DocReference     `json:"doc_references,omitempty"`
	DisplayName         *string            `json:"display_name,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
}

// reference to a strudel used as AI context
type StrudelReference struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	AuthorName string `json:"author_name"`
	URL        string `json:"url"`
}

// reference to documentation used as AI context
type DocReference struct {
	PageName     string `json:"page_name"`
	SectionTitle string `json:"section_title,omitempty"`
	URL          string `json:"url"`
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
	StrudelReferences   []StrudelReference
	DocReferences       []DocReference
	DisplayName         string
}
