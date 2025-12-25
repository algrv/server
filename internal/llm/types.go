package llm

import "context"

// combines query transformation and embedding generation
type LLM interface {
	QueryTransformer
	Embedder
}

// represents different LLM providers
type Provider string

// transforms user queries into technical search terms
type QueryTransformer interface {
	TransformQuery(ctx context.Context, userQuery string) (string, error)
}

// generates embeddings from text
type Embedder interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
)

// holds configuration for LLM initialization
type Config struct {
	// transformer configuration
	TransformerProvider Provider
	TransformerAPIKey   string
	TransformerModel    string // e.g., "claude-3-haiku-20240307"

	// embedder configuration
	EmbedderProvider Provider
	EmbedderAPIKey   string
	EmbedderModel    string // e.g., "text-embedding-3-small"

	// optional parameters
	MaxTokens   int     // for transformer
	Temperature float32 // for transformer
}
