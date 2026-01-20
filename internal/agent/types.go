package agent

import (
	"context"

	"codeberg.org/algopatterns/server/internal/buffer"
	"codeberg.org/algopatterns/server/internal/llm"
	"codeberg.org/algopatterns/server/internal/retriever"
	"codeberg.org/algopatterns/server/internal/strudel"
)

// document and example retrieval interface
type Retriever interface {
	HybridSearchDocs(ctx context.Context, query, editorState string, k int) ([]retriever.SearchResult, error)
	HybridSearchExamples(ctx context.Context, query, editorState string, k int) ([]retriever.ExampleResult, error)
}

// rag result caching interface
type RAGCache interface {
	GetRAGCache(ctx context.Context, sessionID string) (*buffer.CachedRAGResult, error)
	SetRAGCache(ctx context.Context, sessionID string, cache *buffer.CachedRAGResult) error
	ClearRAGCache(ctx context.Context, sessionID string) error
}

// orchestrates rag-powered code generation
type Agent struct {
	retriever Retriever
	generator llm.LLM
	validator *strudel.Validator
}

// all inputs for code generation
type GenerateRequest struct {
	UserQuery           string
	EditorState         string
	ConversationHistory []Message
	CustomGenerator     llm.TextGenerator // optional byok generator
	SessionID           string            // optional: enables rag caching for follow-up messages
	RAGCache            RAGCache          // optional: cache for rag results
}

// reference to a strudel used as context
type StrudelReference struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	AuthorName string `json:"author_name"`
	URL        string `json:"url"`
}

// reference to documentation used as context
type DocReference struct {
	PageName     string `json:"page_name"`
	SectionTitle string `json:"section_title,omitempty"`
	URL          string `json:"url"`
}

// generated code and metadata
type GenerateResponse struct {
	Code                string                    `json:"code,omitempty"`
	DocsRetrieved       int                       `json:"docs_retrieved"`
	ExamplesRetrieved   int                       `json:"examples_retrieved"`
	Examples            []retriever.ExampleResult `json:"-"` // for attribution tracking (internal)
	Docs                []retriever.SearchResult  `json:"-"` // for reference tracking (internal)
	StrudelReferences   []StrudelReference        `json:"strudel_references,omitempty"`
	DocReferences       []DocReference            `json:"doc_references,omitempty"`
	Model               string                    `json:"model"`
	IsActionable        bool                      `json:"is_actionable"`
	IsCodeResponse      bool                      `json:"is_code_response"` // true if response should update editor
	ClarifyingQuestions []string                  `json:"clarifying_questions,omitempty"`
	InputTokens         int                       `json:"input_tokens"`
	OutputTokens        int                       `json:"output_tokens"`
	DidRetry            bool                      `json:"did_retry,omitempty"`
	ValidationError     string                    `json:"validation_error,omitempty"`
}

// chunk of a streaming response
type StreamEvent struct {
	Type    string `json:"type"`              // "chunk", "refs", "done", "error"
	Content string `json:"content,omitempty"` // text chunk for type="chunk"
	Error   string `json:"error,omitempty"`   // error message for type="error"

	// final metadata sent with type="done"
	StrudelReferences []StrudelReference `json:"strudel_references,omitempty"`
	DocReferences     []DocReference     `json:"doc_references,omitempty"`
	Model             string             `json:"model,omitempty"`
	IsCodeResponse    bool               `json:"is_code_response,omitempty"`
	InputTokens       int                `json:"input_tokens,omitempty"`
	OutputTokens      int                `json:"output_tokens,omitempty"`
}

// single conversation turn
type Message struct {
	Role                string             `json:"role"`                       // "user" or "assistant"
	Content             string             `json:"content"`                    // message content
	IsActionable        bool               `json:"is_actionable,omitempty"`    // true if response can be applied
	IsCodeResponse      bool               `json:"is_code_response,omitempty"` // true if AI generated code
	ClarifyingQuestions []string           `json:"clarifying_questions,omitempty"`
	StrudelReferences   []StrudelReference `json:"strudel_references,omitempty"`
	DocReferences       []DocReference     `json:"doc_references,omitempty"`
}
