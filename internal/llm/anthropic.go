package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	anthropicMessagesURL = "https://api.anthropic.com/v1/messages"
	anthropicVersion     = "2023-06-01"
	defaultMaxTokens     = 200
	defaultTemperature   = 0.3
)

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
		httpClient: &http.Client{},
	}
}

func (t *AnthropicTransformer) TransformQuery(ctx context.Context, userQuery string) (string, error) {
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
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicMessagesURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", t.config.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var transformResp transformResponse
	if err := json.NewDecoder(resp.Body).Decode(&transformResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(transformResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	// extract the transformed query text
	transformed := strings.TrimSpace(transformResp.Content[0].Text)

	// combine original query with transformed keywords for hybrid search
	return userQuery + " " + transformed, nil
}

func (t *AnthropicTransformer) GenerateText(ctx context.Context, req TextGenerationRequest) (string, error) {
	// build messages array with system prompt and conversation history
	messages := make([]message, 0, len(req.Messages)+1)

	// if there's a system prompt, prepend it to the first user message
	systemPrompt := req.SystemPrompt

	for i, msg := range req.Messages {
		content := msg.Content

		// prepend system prompt to first user message
		if i == 0 && msg.Role == "user" && systemPrompt != "" {
			content = systemPrompt + "\n\n" + content
		}

		messages = append(messages, message{
			Role:    msg.Role,
			Content: content,
		})
	}

	// determine max tokens (use request value or fall back to config)
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = t.config.MaxTokens
	}

	reqBody := transformRequest{
		Model:       t.config.Model,
		MaxTokens:   maxTokens,
		Temperature: t.config.Temperature,
		Messages:    messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicMessagesURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", t.config.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp transformResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return strings.TrimSpace(apiResp.Content[0].Text), nil
}

// returns the system prompt for query transformation
func buildTransformationPrompt() string {
	const prompt = `You are a technical query expander for Strudel music documentation.
	Your task: Extract 3-5 technical keywords/concepts that would help search for relevant documentation.

	Examples:
	- "play a loud pitched sound" → "audio playback, frequency, pitch, volume, amplitude, sound synthesis"
	- "make a drum pattern" → "rhythm, beat, drum samples, percussion, pattern sequencing"
	- "add reverb effect" → "audio effects, reverb, wet/dry mix, signal processing, DSP"

	Rules:
	- Focus on technical terms found in music/audio documentation
	- Include synonyms (e.g., "loud" → "volume, amplitude")
	- Return ONLY the keywords as comma-separated text
	- Keep it concise (3-5 concepts)
	- Do not include explanations or formatting`

	return prompt
}
