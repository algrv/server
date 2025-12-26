package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	openaiEmbeddingsURL      = "https://api.openai.com/v1/embeddings"
	defaultOpenAIModel       = "text-embedding-3-small"
	openaiEmbeddingDimension = 1536
)

// shared HTTP client for OpenAI API calls
// reuses connection pool and timeout configuration
var openaiHTTPClient = &http.Client{
	Timeout: 60 * time.Second, // total request timeout
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

type embeddingRequest struct {
	Input    []string `json:"input"`
	Model    string   `json:"model"`
	Encoding string   `json:"encoding_format"`
}

type embeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

type OpenAIConfig struct {
	APIKey string
	Model  string // e.g., "text-embedding-3-small"
}

type OpenAIEmbedder struct {
	config     OpenAIConfig
	httpClient *http.Client
}

func NewOpenAIEmbedder(config OpenAIConfig) *OpenAIEmbedder {
	if config.Model == "" {
		config.Model = defaultOpenAIModel
	}

	return &OpenAIEmbedder{
		config:     config,
		httpClient: openaiHTTPClient, // use shared client with proper timeouts and connection pooling
	}
}

func (e *OpenAIEmbedder) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embeddings[0], nil
}

func (e *OpenAIEmbedder) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	reqBody := embeddingRequest{
		Input:    texts,
		Model:    e.config.Model,
		Encoding: "float",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openaiEmbeddingsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.config.APIKey))

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var embResp embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	embeddings := make([][]float32, len(embResp.Data))
	for _, data := range embResp.Data {
		embeddings[data.Index] = data.Embedding
	}

	return embeddings, nil
}
