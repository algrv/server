package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	openaiEmbeddingsURL      = "https://api.openai.com/v1/embeddings"
	openaiChatCompletionsURL = "https://api.openai.com/v1/chat/completions"
	defaultOpenAIModel       = "text-embedding-3-small"
	defaultOpenAIChatModel   = "gpt-4o"
	// openaiEmbeddingDimension = 1536
)

// shared HTTP client for OpenAI API calls
var openaiHTTPClient = &http.Client{
	Timeout: 60 * time.Second, // total request timeout
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

// rate limiter for OpenAI API calls
// limits to 50 requests/second with burst capacity of 10
var openaiRateLimiter = rate.NewLimiter(50, 10)

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

	// apply rate limiting before making the request
	if err := openaiRateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck
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

// openai chat completion types
type openaiChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openaiChatMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float32             `json:"temperature,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
}

// streaming chunk from openai
type openaiStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

type openaiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// implements TextGenerator and QueryTransformer for openai
type OpenAIGenerator struct {
	config     OpenAIConfig
	httpClient *http.Client
}

func NewOpenAIGenerator(config OpenAIConfig) *OpenAIGenerator {
	if config.Model == "" {
		config.Model = defaultOpenAIChatModel
	}

	return &OpenAIGenerator{
		config:     config,
		httpClient: openaiHTTPClient,
	}
}

func (g *OpenAIGenerator) Model() string {
	return g.config.Model
}

func (g *OpenAIGenerator) GenerateText(ctx context.Context, req TextGenerationRequest) (*TextGenerationResponse, error) {
	messages := make([]openaiChatMessage, 0, len(req.Messages)+1)

	// add system message
	if req.SystemPrompt != "" {
		messages = append(messages, openaiChatMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// add conversation messages
	for _, msg := range req.Messages {
		messages = append(messages, openaiChatMessage(msg))
	}

	reqBody := openaiChatRequest{
		Model:       g.config.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: 0.7,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openaiChatCompletionsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.config.APIKey))

	if err := openaiRateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &TextGenerationResponse{
		Text: chatResp.Choices[0].Message.Content,
		Usage: Usage{
			InputTokens:  chatResp.Usage.PromptTokens,
			OutputTokens: chatResp.Usage.CompletionTokens,
		},
	}, nil
}

func (g *OpenAIGenerator) GenerateTextStream(ctx context.Context, req TextGenerationRequest, onChunk func(chunk string) error) (*TextGenerationResponse, error) {
	messages := make([]openaiChatMessage, 0, len(req.Messages)+1)

	if req.SystemPrompt != "" {
		messages = append(messages, openaiChatMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	for _, msg := range req.Messages {
		messages = append(messages, openaiChatMessage(msg))
	}

	reqBody := openaiChatRequest{
		Model:       g.config.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: 0.7,
		Stream:      true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openaiChatCompletionsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.config.APIKey))

	if err := openaiRateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var fullText strings.Builder
	var usage Usage

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk openaiStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // skip malformed chunks
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			fullText.WriteString(content)
			if err := onChunk(content); err != nil {
				return nil, fmt.Errorf("chunk callback error: %w", err)
			}
		}

		// capture usage from final chunk if available
		if chunk.Usage != nil {
			usage.InputTokens = chunk.Usage.PromptTokens
			usage.OutputTokens = chunk.Usage.CompletionTokens
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	return &TextGenerationResponse{
		Text:  fullText.String(),
		Usage: usage,
	}, nil
}

func (g *OpenAIGenerator) TransformQuery(ctx context.Context, userQuery string) (string, error) {
	analysis, err := g.AnalyzeQuery(ctx, userQuery)
	if err != nil {
		return "", err
	}

	return userQuery + " " + analysis.TransformedQuery, nil
}

func (g *OpenAIGenerator) AnalyzeQuery(ctx context.Context, userQuery string) (*QueryAnalysis, error) {
	messages := []openaiChatMessage{
		{
			Role:    "system",
			Content: buildTransformationPrompt(),
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("User query: %s", userQuery),
		},
	}

	reqBody := openaiChatRequest{
		Model:       g.config.Model,
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   200,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openaiChatCompletionsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.config.APIKey))

	if err := openaiRateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	responseText := chatResp.Choices[0].Message.Content

	var analysis QueryAnalysis
	if err := json.Unmarshal([]byte(responseText), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse query analysis JSON: %w", err)
	}

	return &analysis, nil
}
