package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// manages HTTP requests to the agent REST API
type AgentClient struct {
	endpoint   string
	httpClient *http.Client
}

// creates a new agent REST client
func NewAgentClient() *AgentClient {
	endpoint := os.Getenv("ALGOJAMS_API_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}

	return &AgentClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: agentRequestTimeout,
		},
	}
}

// sends a generate request to the agent REST API
func (c *AgentClient) Generate(ctx context.Context, userQuery, editorState string, conversationHistory []MessageModel) (*AgentResponseMsg, error) {
	// filter out messages with empty content (e.g., clarifying questions responses)
	// LLM APIs reject messages with empty content
	filteredHistory := make([]MessageModel, 0, len(conversationHistory))
	for _, msg := range conversationHistory {
		if msg.Content != "" {
			filteredHistory = append(filteredHistory, MessageModel{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	// build request payload
	payload := agentGenerateRequest{
		UserQuery:           userQuery,
		EditorState:         editorState,
		ConversationHistory: filteredHistory,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// create HTTP request
	url := fmt.Sprintf("%s/api/v1/agent/generate", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	// read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// handle error responses
	if resp.StatusCode != http.StatusOK {
		var errResp agentErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("%s: %s", errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// parse success response
	var result agentGenerateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	code, metadata := formatAgentResponse(result)
	return &AgentResponseMsg{
		userQuery:      userQuery,
		code:           code,
		metadata:       metadata,
		questions:      result.ClarifyingQuestions,
		isCodeResponse: result.IsCodeResponse,
	}, nil
}

// returns a tea.Cmd that sends a generate request
func (c *AgentClient) GenerateCmd(userQuery, editorState string, conversationHistory []MessageModel) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), agentRequestTimeout)
		defer cancel()

		resp, err := c.Generate(ctx, userQuery, editorState, conversationHistory)
		if err != nil {
			return AgentErrorMsg{userQuery: userQuery, err: err}
		}

		return *resp
	}
}

// REST API request/response types

type agentGenerateRequest struct {
	UserQuery           string         `json:"user_query"`
	EditorState         string         `json:"editor_state,omitempty"`
	ConversationHistory []MessageModel `json:"conversation_history,omitempty"`
}

type agentGenerateResponse struct {
	Code                string   `json:"code,omitempty"`
	DocsRetrieved       int      `json:"docs_retrieved"`
	ExamplesRetrieved   int      `json:"examples_retrieved"`
	Model               string   `json:"model"`
	IsActionable        bool     `json:"is_actionable"`
	IsCodeResponse      bool     `json:"is_code_response"`
	ClarifyingQuestions []string `json:"clarifying_questions,omitempty"`
}

type agentErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// formatAgentResponse extracts code and metadata from the response
func formatAgentResponse(result agentGenerateResponse) (string, string) {
	code := result.Code

	metadata := fmt.Sprintf("retrieved: %d docs, %d examples | model: %s",
		result.DocsRetrieved,
		result.ExamplesRetrieved,
		result.Model)

	return code, metadata
}

// timeout for agent requests
const agentRequestTimeout = 60 * time.Second
