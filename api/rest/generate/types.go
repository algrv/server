package generate

import "github.com/algorave/server/internal/agent"

type Request struct {
	UserQuery           string          `json:"user_query" binding:"required"`
	EditorState         string          `json:"editor_state"`
	ConversationHistory []agent.Message `json:"conversation_history"`
	StrudelID           string          `json:"strudel_id,omitempty"`
	SessionID           string          `json:"session_id,omitempty"`
}

type Response struct {
	Code                string   `json:"code,omitempty"`
	DocsRetrieved       int      `json:"docs_retrieved"`
	ExamplesRetrieved   int      `json:"examples_retrieved"`
	Model               string   `json:"model"`
	IsActionable        bool     `json:"is_actionable"`
	ClarifyingQuestions []string `json:"clarifying_questions,omitempty"`
	SessionID           string   `json:"session_id,omitempty"`
}
