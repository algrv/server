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
	Stream      bool      `json:"stream,omitempty"`
}

// anthropic streaming event types
type anthropicStreamEvent struct {
	Type  string          `json:"type"`
	Index int             `json:"index,omitempty"`
	Delta json.RawMessage `json:"delta,omitempty"`
	Usage *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
	Message *struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message,omitempty"`
}

type anthropicContentDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
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

func (t *AnthropicTransformer) GenerateTextStream(ctx context.Context, req TextGenerationRequest, onChunk func(chunk string) error) (*TextGenerationResponse, error) {
	messages := make([]message, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, message(msg))
	}

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
		Stream:      true,
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

	var fullText strings.Builder
	var usage Usage

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Anthropic sends "event: <type>" followed by "data: <json>"
		if strings.HasPrefix(line, "event:") {
			continue // skip event type lines, we get type from data
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue // skip malformed events
		}

		switch event.Type {
		case "content_block_delta":
			var delta anthropicContentDelta
			if err := json.Unmarshal(event.Delta, &delta); err == nil && delta.Text != "" {
				fullText.WriteString(delta.Text)
				if err := onChunk(delta.Text); err != nil {
					return nil, fmt.Errorf("chunk callback error: %w", err)
				}
			}
		case "message_start":
			if event.Message != nil {
				usage.InputTokens = event.Message.Usage.InputTokens
			}
		case "message_delta":
			if event.Usage != nil {
				usage.OutputTokens = event.Usage.OutputTokens
			}
		case "message_stop":
			// end of message
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	return &TextGenerationResponse{
		Text:  strings.TrimSpace(fullText.String()),
		Usage: usage,
	}, nil
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

Your task: Analyze the user's query and determine:
1. Is it actionable (specific enough to proceed)?
2. Is it a code request (wants code generated) or a question (wants information/explanation)?

Return a JSON object with this structure:
{
  "transformed_query": "3-5 technical keywords for search (comma-separated)",
  "is_actionable": true/false,
  "is_code_request": true/false,
  "concrete_requests": ["list", "of", "specific", "things", "to", "do"],
  "clarifying_questions": ["list", "of", "questions", "if", "vague"]
}

CLASSIFICATION RULES:

1. CODE REQUESTS (is_code_request: true) - User wants code generated:
   - "set the bpm to 120" → wants code
   - "add a kick drum on every beat" → wants code
   - "make the hi-hats faster" → wants code modification
   - "create a bassline" → wants code

2. QUESTIONS (is_code_request: false) - User wants information/explanation:
   - "how do I use lpf?" → asking for explanation
   - "what does the note function do?" → asking for information
   - "what key is good for house music?" → asking for advice
   - "can you explain scales?" → asking for explanation
   - "what's the difference between sound() and note()?" → asking for comparison

3. ACTIONABLE vs VAGUE:
   - Actionable: Specific enough to proceed (either generate code or answer question)
   - Vague: Needs clarification before proceeding

EXAMPLES:

Input: "set the bpm to 120"
{
  "transformed_query": "tempo, bpm, speed, setcpm",
  "is_actionable": true,
  "is_code_request": true,
  "concrete_requests": ["set tempo to 120 BPM"],
  "clarifying_questions": []
}

Input: "create a house beat"
{
  "transformed_query": "house music, beat, rhythm, drums, pattern",
  "is_actionable": false,
  "is_code_request": true,
  "concrete_requests": [],
  "clarifying_questions": ["What BPM would you like?", "Which elements should I add? (kick, hi-hat, snare, etc.)", "Any specific pattern or style in mind?"]
}

Input: "add a kick drum on every beat"
{
  "transformed_query": "kick drum, bd, bass drum, four on the floor, rhythm",
  "is_actionable": true,
  "is_code_request": true,
  "concrete_requests": ["add kick drum pattern with hits on every beat"],
  "clarifying_questions": []
}

Input: "how do I use the lpf filter?"
{
  "transformed_query": "lpf, low pass filter, cutoff, frequency",
  "is_actionable": true,
  "is_code_request": false,
  "concrete_requests": [],
  "clarifying_questions": []
}

Input: "what key works well for techno?"
{
  "transformed_query": "key, scale, techno, music theory",
  "is_actionable": true,
  "is_code_request": false,
  "concrete_requests": [],
  "clarifying_questions": []
}

Return ONLY valid JSON, no markdown or explanations.`

	return prompt
}
