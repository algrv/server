package agent

import (
	"context"

	"github.com/algorave/server/internal/llm"
	"github.com/algorave/server/internal/retriever"
)

// interface for document and example retrieval
type Retriever interface {
	HybridSearchDocs(ctx context.Context, query, editorState string, k int) ([]retriever.SearchResult, error)
	HybridSearchExamples(ctx context.Context, query, editorState string, k int) ([]retriever.ExampleResult, error)
}

// orchestrates rag-powered code generation
type Agent struct {
	retriever Retriever
	generator llm.LLM
}

// contains all inputs for code generation
type GenerateRequest struct {
	UserQuery           string
	EditorState         string
	ConversationHistory []Message
	CustomGenerator     llm.TextGenerator // optional BYOK generator
}

// contains the generated code and metadata
type GenerateResponse struct {
	Code                string   `json:"code,omitempty"`
	DocsRetrieved       int      `json:"docs_retrieved"`
	ExamplesRetrieved   int      `json:"examples_retrieved"`
	Model               string   `json:"model"`
	IsActionable        bool     `json:"is_actionable"`
	ClarifyingQuestions []string `json:"clarifying_questions,omitempty"`
}

// represents a single conversation turn
type Message struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"` // message content
}
