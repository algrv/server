package llm

import "context"

// combines query transformation, embedding generation, and text generation
type LLM interface {
	QueryTransformer
	Embedder
	TextGenerator
}

// represents different LLM providers
type Provider string

// transforms user queries into technical search terms
type QueryTransformer interface {
	TransformQuery(ctx context.Context, userQuery string) (string, error)
	AnalyzeQuery(ctx context.Context, userQuery string) (*QueryAnalysis, error)
}

// QueryAnalysis contains the result of query transformation with actionability metadata
type QueryAnalysis struct {
	TransformedQuery    string   `json:"transformed_query"`
	IsActionable        bool     `json:"is_actionable"`
	ConcreteRequests    []string `json:"concrete_requests"`
	ClarifyingQuestions []string `json:"clarifying_questions"`
}

// generates embeddings from text
type Embedder interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// generates text/code from prompts
type TextGenerator interface {
	GenerateText(ctx context.Context, req TextGenerationRequest) (string, error)
	Model() string
}

// TextGenerationRequest contains inputs for text generation
type TextGenerationRequest struct {
	SystemPrompt string    // system-level instructions
	Messages     []Message // conversation history
	MaxTokens    int       // max tokens to generate
}

// Message represents a conversation turn
type Message struct {
	Role    string // "user" or "assistant"
	Content string // message content
}

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
)

// holds configuration for LLM initialization
type Config struct {
	// transformer configuration (query expansion)
	TransformerProvider Provider
	TransformerAPIKey   string
	TransformerModel    string  // e.g., "claude-3-haiku-20240307"
	TransformerMaxTokens   int     // e.g., 200
	TransformerTemperature float32 // e.g., 0.3

	// generator configuration (code generation)
	GeneratorProvider Provider
	GeneratorAPIKey   string
	GeneratorModel    string  // e.g., "claude-sonnet-4-20250514"
	GeneratorMaxTokens   int     // e.g., 4096
	GeneratorTemperature float32 // e.g., 0.7

	// embedder configuration
	EmbedderProvider Provider
	EmbedderAPIKey   string
	EmbedderModel    string // e.g., "text-embedding-3-small"
}
