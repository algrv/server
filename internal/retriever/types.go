package retriever

import (
	"github.com/algorave/server/internal/llm"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Client performs vector similarity search on documentation and examples
type Client struct {
	db   *pgxpool.Pool
	llm  llm.LLM
	topK int
}

// SearchResult represents a document chunk from vector search
type SearchResult struct {
	ID           string
	PageName     string
	PageURL      string
	SectionTitle string
	Content      string
	Similarity   float32
	Metadata     map[string]interface{}
}

// ExampleResult represents an example Strudel from vector search
type ExampleResult struct {
	ID          string
	Title       string
	Description string
	Code        string
	Tags        []string
	URL         string
	Similarity  float32
}
