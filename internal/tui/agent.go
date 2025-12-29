package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
)

// WebSocket message types (must match server)
const (
	typeAgentRequest  = "agent_request"
	typeAgentResponse = "agent_response"
	typeError         = "error"
)

// wsMessage is the WebSocket message envelope
type wsMessage struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	UserID    string          `json:"user_id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// agentRequestPayload is the payload for agent_request messages
type agentRequestPayload struct {
	UserQuery           string         `json:"user_query"`
	EditorState         string         `json:"editor_state,omitempty"`
	ConversationHistory []MessageModel `json:"conversation_history,omitempty"`
}

// agentResponsePayload is the payload for agent_response messages
type agentResponsePayload struct {
	Code                string   `json:"code,omitempty"`
	DocsRetrieved       int      `json:"docs_retrieved"`
	ExamplesRetrieved   int      `json:"examples_retrieved"`
	Model               string   `json:"model"`
	IsActionable        bool     `json:"is_actionable"`
	ClarifyingQuestions []string `json:"clarifying_questions,omitempty"`
}

// errorPayload is the payload for error messages
type errorPayload struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func sendToAgent(userQuery, editorState string, conversationHistory []MessageModel) tea.Cmd {
	return func() tea.Msg {
		endpoint := os.Getenv("ALGORAVE_WS_ENDPOINT")

		if endpoint == "" {
			endpoint = "ws://localhost:8080/api/v1/ws"
		}

		// creates anonymous session
		conn, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
		if err != nil {
			return AgentErrorMsg{userQuery: userQuery, err: fmt.Errorf("failed to connect: %w", err)}
		}

		defer conn.Close() //nolint:errcheck // closing on return, error irrelevant

		// read the first message to get session ID
		var welcomeMsg wsMessage
		if err := conn.ReadJSON(&welcomeMsg); err != nil {
			return AgentErrorMsg{userQuery: userQuery, err: fmt.Errorf("failed to read welcome: %w", err)}
		}

		sessionID := welcomeMsg.SessionID

		// build agent request payload
		payload := agentRequestPayload{
			UserQuery:           userQuery,
			EditorState:         editorState,
			ConversationHistory: conversationHistory,
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return AgentErrorMsg{userQuery: userQuery, err: fmt.Errorf("failed to marshal payload: %w", err)}
		}

		// send agent_request message
		reqMsg := wsMessage{
			Type:      typeAgentRequest,
			SessionID: sessionID,
			Timestamp: time.Now(),
			Payload:   payloadBytes,
		}

		if err := conn.WriteJSON(reqMsg); err != nil {
			return AgentErrorMsg{userQuery: userQuery, err: fmt.Errorf("failed to send request: %w", err)}
		}

		// wait for agent_response (skip agent_request broadcast)
		for {
			var respMsg wsMessage

			if err := conn.ReadJSON(&respMsg); err != nil {
				return AgentErrorMsg{userQuery: userQuery, err: fmt.Errorf("failed to read response: %w", err)}
			}

			switch respMsg.Type {
			case typeAgentResponse:
				var resp agentResponsePayload

				if err := json.Unmarshal(respMsg.Payload, &resp); err != nil {
					return AgentErrorMsg{userQuery: userQuery, err: fmt.Errorf("failed to parse response: %w", err)}
				}

				code, metadata := formatAgentResponse(resp)
				return AgentResponseMsg{
					userQuery: userQuery,
					code:      code,
					metadata:  metadata,
					questions: resp.ClarifyingQuestions,
				}

			case typeError:
				var errResp errorPayload

				if err := json.Unmarshal(respMsg.Payload, &errResp); err != nil {
					return AgentErrorMsg{userQuery: userQuery, err: fmt.Errorf("failed to parse error: %w", err)}
				}

				return AgentErrorMsg{userQuery: userQuery, err: fmt.Errorf("%s: %s", errResp.Error, errResp.Message)}

			case typeAgentRequest:
				continue

			default:
				continue
			}
		}
	}
}

func formatAgentResponse(result agentResponsePayload) (string, string) {
	code := result.Code

	metadata := fmt.Sprintf("retrieved: %d docs, %d examples | model: %s",
		result.DocsRetrieved,
		result.ExamplesRetrieved,
		result.Model)

	return code, metadata
}
