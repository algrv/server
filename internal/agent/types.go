package agent

import (
	"context"

	"codeberg.org/algorave/server/internal/llm"
	"codeberg.org/algorave/server/internal/retriever"
	"codeberg.org/algorave/server/internal/strudel"
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
	validator *strudel.Validator
}

// contains all inputs for code generation
type GenerateRequest struct {
	UserQuery           string
	EditorState         string
	ConversationHistory []Message
	CustomGenerator     llm.TextGenerator // optional BYOK generator
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

// contains the generated code and metadata
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

// represents a single conversation turn
type Message struct {
	Role           string `json:"role"`                       // "user" or "assistant"
	Content        string `json:"content"`                    // message content
	IsCodeResponse bool   `json:"is_code_response,omitempty"` // true if AI generated code
}
