package retriever

import (
	"github.com/algorave/server/internal/llm"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RetrieverConfig holds configuration for the retriever client
type RetrieverConfig struct {
	DBConnString string
	TopK         int
}

type Client struct {
	pool *pgxpool.Pool
	llm  llm.LLM
	topK int
}

type SearchResult struct {
	ID           string
	PageName     string
	PageURL      string
	SectionTitle string
	Content      string
	Similarity   float32
	Metadata     map[string]interface{}
}

type ExampleResult struct {
	ID          string
	Title       string
	Description string
	Code        string
	Tags        []string
	URL         string
	Similarity  float32
}
