package llm

import (
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
	anthropicMessagesURL = "https://api.anthropic.com/v1/messages"
	anthropicVersion     = "2023-06-01"
	defaultMaxTokens     = 200
	defaultTemperature   = 0.3
)

// shared HTTP client for Anthropic API calls
var anthropicHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

// rate limiter for Anthropic API calls (50 requests/second with burst capacity of 10)
var anthropicRateLimiter = rate.NewLimiter(50, 10)

type transformRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	System      string    `json:"system,omitempty"`
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
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type AnthropicConfig struct {
	APIKey      string
	Model       string  // e.g., "claude-3-haiku-20240307"
	MaxTokens   int     // max tokens for response
	Temperature float32 // 0.0 to 1.0
}

type AnthropicTransformer struct {
	config     AnthropicConfig
	httpClient *http.Client
}

func NewAnthropicTransformer(config AnthropicConfig) *AnthropicTransformer {
	if config.MaxTokens == 0 {
		config.MaxTokens = defaultMaxTokens
	}

	if config.Temperature == 0 {
		config.Temperature = defaultTemperature
	}

	return &AnthropicTransformer{
		config:     config,
		httpClient: anthropicHTTPClient,
	}
}

func (t *AnthropicTransformer) Model() string {
	return t.config.Model
}

// analyzes the user query and returns structured actionability data
func (t *AnthropicTransformer) AnalyzeQuery(ctx context.Context, userQuery string) (*QueryAnalysis, error) {
	reqBody := transformRequest{
		Model:       t.config.Model,
		MaxTokens:   t.config.MaxTokens,
		Temperature: t.config.Temperature,
		Messages: []message{
			{
				Role:    "user",
				Content: fmt.Sprintf("%s\n\nUser query: %s", buildTransformationPrompt(), userQuery),
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicMessagesURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", t.config.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	// rate limiting
	if err := anthropicRateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var transformResp transformResponse
	if err := json.NewDecoder(resp.Body).Decode(&transformResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(transformResp.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	responseText := strings.TrimSpace(transformResp.Content[0].Text)
	var analysis QueryAnalysis

	if err := json.Unmarshal([]byte(responseText), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse query analysis JSON: %w", err)
	}

	return &analysis, nil
}

// transforms the user query for vector search (backward compatible)
func (t *AnthropicTransformer) TransformQuery(ctx context.Context, userQuery string) (string, error) {
	analysis, err := t.AnalyzeQuery(ctx, userQuery)
	if err != nil {
		return "", err
	}

	// combine original query with transformed keywords for hybrid search
	return userQuery + " " + analysis.TransformedQuery, nil
}

func (t *AnthropicTransformer) GenerateText(ctx context.Context, req TextGenerationRequest) (*TextGenerationResponse, error) {
	// build messages array from conversation history
	messages := make([]message, 0, len(req.Messages))

	for _, msg := range req.Messages {
		messages = append(messages, message(msg))
	}

	// determine max tokens (use request value or fall back to config)
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = t.config.MaxTokens
	}

	reqBody := transformRequest{
		Model:       t.config.Model,
		MaxTokens:   maxTokens,
		System:      req.SystemPrompt,
		Temperature: t.config.Temperature,
		Messages:    messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicMessagesURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", t.config.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	// rate limiting
	if err := anthropicRateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp transformResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	return &TextGenerationResponse{
		Text: strings.TrimSpace(apiResp.Content[0].Text),
		Usage: Usage{
			InputTokens:  apiResp.Usage.InputTokens,
			OutputTokens: apiResp.Usage.OutputTokens,
		},
	}, nil
}

// returns system prompt for query transformation
func buildTransformationPrompt() string {
	const prompt = `You are a query analyzer for Strudel music code generation.

Your task: Analyze the user's query and determine if it's actionable (specific enough to generate code).

Return a JSON object with this structure:
{
  "transformed_query": "3-5 technical keywords for search (comma-separated)",
  "is_actionable": true/false,
  "concrete_requests": ["list", "of", "specific", "things", "to", "do"],
  "clarifying_questions": ["list", "of", "questions", "if", "vague"]
}

ACTIONABLE queries (specific, can generate code):
- "set the bpm to 120" → actionable, clear instruction
- "add a kick drum on every beat" → actionable, specific sound + timing
- "make the hi-hats play 8 times per cycle" → actionable, clear pattern

VAGUE queries (need clarification):
- "create a house beat" → vague, no specifics about which elements
- "make it sound better" → vague, no concrete direction
- "add some drums" → vague, which drums? what pattern?

Examples:

Input: "set the bpm to 120"
{
  "transformed_query": "tempo, bpm, speed, setcpm",
  "is_actionable": true,
  "concrete_requests": ["set tempo to 120 BPM"],
  "clarifying_questions": []
}

Input: "create a house beat"
{
  "transformed_query": "house music, beat, rhythm, drums, pattern",
  "is_actionable": false,
  "concrete_requests": [],
  "clarifying_questions": ["What BPM would you like?", "Which elements should I add? (kick, hi-hat, snare, etc.)", "Any specific pattern or style in mind?"]
}

Input: "add a kick drum on every beat"
{
  "transformed_query": "kick drum, bd, bass drum, four on the floor, rhythm",
  "is_actionable": true,
  "concrete_requests": ["add kick drum pattern with hits on every beat"],
  "clarifying_questions": []
}

Return ONLY valid JSON, no markdown or explanations.`

	return prompt
}
