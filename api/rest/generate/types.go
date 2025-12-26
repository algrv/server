package generate

import "github.com/algorave/server/internal/agent"

// Request represents the request body for code generation
type Request struct {
	UserQuery           string          `json:"user_query" binding:"required"`
	EditorState         string          `json:"editor_state"`
	ConversationHistory []agent.Message `json:"conversation_history"`
}

// Response represents the response for code generation
type Response struct {
	Code                string   `json:"code,omitempty"`
	DocsRetrieved       int      `json:"docs_retrieved"`
	ExamplesRetrieved   int      `json:"examples_retrieved"`
	Model               string   `json:"model"`
	IsActionable        bool     `json:"is_actionable"`
	ClarifyingQuestions []string `json:"clarifying_questions,omitempty"`
}
