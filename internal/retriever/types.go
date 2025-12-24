package retriever

import (
	"net/http"

	"github.com/algorave/server/internal/embedder"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	pool           *pgxpool.Pool
	anthropicKey   string
	httpClient     *http.Client
	embedderClient *embedder.Client
	topK           int
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

type transformRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Messages    []message `json:"messages"`
	Temperature float32   `json:"temperature"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type transformResponse struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Role    string    `json:"role"`
	Content []content `json:"content"`
	Model   string    `json:"model"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
