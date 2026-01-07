package strudels

import (
	"time"

	"github.com/algrv/server/algorave/strudels"
	"github.com/algrv/server/api/rest/pagination"
)

// StrudelsListResponse wraps a list of strudels with pagination
type StrudelsListResponse struct {
	Strudels   []strudels.Strudel `json:"strudels"`
	Pagination pagination.Meta    `json:"pagination"`
}

// MessageResponse for simple success messages
type MessageResponse struct {
	Message string `json:"message"`
}

// TagsListResponse wraps a list of unique tags
type TagsListResponse struct {
	Tags []string `json:"tags"`
}

// StrudelDetailResponse includes strudel with full conversation history from strudel_messages
type StrudelDetailResponse struct {
	ID                  string                   `json:"id"`
	UserID              string                   `json:"user_id"`
	Title               string                   `json:"title"`
	Code                string                   `json:"code"`
	IsPublic            bool                     `json:"is_public"`
	CCSignal            *strudels.CCSignal       `json:"cc_signal,omitempty"`
	ForkedFrom          *string                  `json:"forked_from,omitempty"`
	ParentCCSignal      *strudels.CCSignal       `json:"parent_cc_signal,omitempty"`
	Description         string                   `json:"description,omitempty"`
	Tags                []string                 `json:"tags,omitempty"`
	Categories          []string                 `json:"categories,omitempty"`
	ConversationHistory []ConversationMessageDTO `json:"conversation_history,omitempty"`
	CreatedAt           time.Time                `json:"created_at"`
	UpdatedAt           time.Time                `json:"updated_at"`
}

// StrudelReferenceDTO for attribution display
type StrudelReferenceDTO struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	AuthorName string `json:"author_name"`
	URL        string `json:"url"`
}

// DocReferenceDTO for attribution display
type DocReferenceDTO struct {
	PageName     string `json:"page_name"`
	SectionTitle string `json:"section_title,omitempty"`
	URL          string `json:"url"`
}

// ConversationMessageDTO represents a full AI conversation message
type ConversationMessageDTO struct {
	ID                  string                `json:"id"`
	Role                string                `json:"role"`
	Content             string                `json:"content"`
	IsActionable        bool                  `json:"is_actionable"`
	IsCodeResponse      bool                  `json:"is_code_response"`
	ClarifyingQuestions []string              `json:"clarifying_questions,omitempty"`
	StrudelReferences   []StrudelReferenceDTO `json:"strudel_references,omitempty"`
	DocReferences       []DocReferenceDTO     `json:"doc_references,omitempty"`
	CreatedAt           time.Time             `json:"created_at"`
}
