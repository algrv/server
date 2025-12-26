package agent

import (
	"context"

	"github.com/algorave/server/internal/llm"
	"github.com/algorave/server/internal/retriever"
)

// defines the interface for document and example retrieval
type Retriever interface {
	HybridSearchDocs(ctx context.Context, query, editorState string, k int) ([]retriever.SearchResult, error)
	HybridSearchExamples(ctx context.Context, query, editorState string, k int) ([]retriever.ExampleResult, error)
}

// orchestrates RAG-powered code generation
type Agent struct {
	retriever Retriever
	generator llm.LLM
}

// contains all inputs for code generation
type GenerateRequest struct {
	UserQuery           string
	EditorState         string
	ConversationHistory []Message
}

// contains the generated code and metadata
type GenerateResponse struct {
	Code              string
	DocsRetrieved     int
	ExamplesRetrieved int
	Model             string
}

// represents a single conversation turn
type Message struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"` // message content
}
