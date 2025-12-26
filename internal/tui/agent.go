package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type agentResponse struct {
	Code                string   `json:"code"`
	DocsRetrieved       int      `json:"docs_retrieved"`
	ExamplesRetrieved   int      `json:"examples_retrieved"`
	Model               string   `json:"model"`
	IsActionable        bool     `json:"is_actionable"`
	ClarifyingQuestions []string `json:"clarifying_questions"`
}

func sendToAgent(userQuery, editorState string, conversationHistory []Message) tea.Cmd {
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
			return AgentErrorMsg{err: fmt.Errorf("failed to marshal request: %w", err)}
		}

		resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return AgentErrorMsg{err: fmt.Errorf("failed to connect to agent: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return AgentErrorMsg{err: fmt.Errorf("agent returned error %d: %s", resp.StatusCode, string(body))}
		}

		var result agentResponse

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return AgentErrorMsg{err: fmt.Errorf("failed to parse response: %w", err)}
		}

		code, metadata := formatAgentResponse(result)
		return AgentResponseMsg{
			code:      code,
			metadata:  metadata,
			questions: result.ClarifyingQuestions,
		}
	}
}

func formatAgentResponse(result agentResponse) (string, string) {
	code := result.Code

	metadata := fmt.Sprintf("retrieved: %d docs, %d examples | model: %s",
		result.DocsRetrieved,
		result.ExamplesRetrieved,
		result.Model)

	return code, metadata
}
