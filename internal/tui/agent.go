package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/algorave/server/internal/agent"
	tea "github.com/charmbracelet/bubbletea"
)

func sendToAgent(userQuery, editorState string, conversationHistory []MessageModel) tea.Cmd {
	return func() tea.Msg {
		endpoint := os.Getenv("ALGORAVE_AGENT_ENDPOINT")
		if endpoint == "" {
			endpoint = "http://localhost:8080/api/v1/generate"
		}

		payload := map[string]interface{}{
			"user_query":           userQuery,
			"editor_state":         editorState,
			"conversation_history": conversationHistory,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return AgentErrorMsg{userQuery: userQuery, err: fmt.Errorf("failed to marshal request: %w", err)}
		}

		resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData)) //nolint:gosec // G107: endpoint is from config, not user input
		if err != nil {
			return AgentErrorMsg{userQuery: userQuery, err: fmt.Errorf("failed to connect to agent: %w", err)}
		}
		defer resp.Body.Close() //nolint:errcheck

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body) //nolint:errcheck
			return AgentErrorMsg{userQuery: userQuery, err: fmt.Errorf("agent returned error %d: %s", resp.StatusCode, string(body))}
		}

		var result agent.GenerateResponse

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return AgentErrorMsg{userQuery: userQuery, err: fmt.Errorf("failed to parse response: %w", err)}
		}

		code, metadata := formatAgentResponse(result)
		return AgentResponseMsg{
			userQuery: userQuery,
			code:      code,
			metadata:  metadata,
			questions: result.ClarifyingQuestions,
		}
	}
}

func formatAgentResponse(result agent.GenerateResponse) (string, string) {
	code := result.Code

	metadata := fmt.Sprintf("retrieved: %d docs, %d examples | model: %s",
		result.DocsRetrieved,
		result.ExamplesRetrieved,
		result.Model)

	return code, metadata
}
