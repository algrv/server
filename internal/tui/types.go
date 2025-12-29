package tui

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/gorilla/websocket"
)

// represents the current state of the TUI
type AppState int

const (
	StateWelcome AppState = iota
	StateEditor
	StateOutput
	StateLoading
)

// main TUI application model
type Model struct {
	state   AppState
	mode    string
	width   int
	height  int
	err     error
	welcome *Welcome
	editor  *EditorModel
}

// sent when an error occurs
type ErrorMsg struct {
	err error
}

// sent to transition to the editor state
type EnterEditorMsg struct{}

// represents a chat message in the conversation
type MessageModel struct {
	Role      string   `json:"role"`
	Content   string   `json:"content"`
	Metadata  string   `json:"metadata,omitempty"`
	Questions []string `json:"questions,omitempty"`
}

// code editor interface
type EditorModel struct {
	input               textinput.Model
	viewport            viewport.Model
	width               int
	height              int
	conversationHistory []MessageModel
	isFetching          bool
	spinner             spinner.Model
	glamourRenderer     *glamour.TermRenderer
	ready               bool
	shouldScrollBottom  bool
	wsClient            *WSClient
	wsConnected         bool
	wsError             error
}

// sent when the agent completes a request
type AgentResponseMsg struct {
	userQuery string
	code      string
	metadata  string
	questions []string
}

// sent when the agent encounters an error
type AgentErrorMsg struct {
	userQuery string
	err       error
}

// welcome screen model
type Welcome struct {
	mode     string
	input    string
	commands []Command
}

// represents an available TUI command
type Command struct {
	Name        string
	Description string
	Available   bool
}

// sent when the server starts
type ServerStartedMsg struct{}

// sent when the ingester completes
type IngesterCompleteMsg struct{}

// wsclient types
// manages a persistent webSocket connection
type WSClient struct {
	conn      *websocket.Conn
	sessionID string
	mu        sync.Mutex
	connected bool
	endpoint  string

	// tracks in-flight requests waiting for responses
	pending   map[string]chan wsMessage
	pendingMu sync.Mutex
}

// webSocket message envelope
type wsMessage struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	UserID    string          `json:"user_id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// payload for agent_request messages
type agentRequestPayload struct {
	UserQuery           string         `json:"user_query"`
	EditorState         string         `json:"editor_state,omitempty"`
	ConversationHistory []MessageModel `json:"conversation_history,omitempty"`
}

// payload for agent_response messages
type agentResponsePayload struct {
	Code                string   `json:"code,omitempty"`
	DocsRetrieved       int      `json:"docs_retrieved"`
	ExamplesRetrieved   int      `json:"examples_retrieved"`
	Model               string   `json:"model"`
	IsActionable        bool     `json:"is_actionable"`
	ClarifyingQuestions []string `json:"clarifying_questions,omitempty"`
}

// payload for error messages
type errorPayload struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// sent when the webSocket connection is established
type WSConnectedMsg struct {
	sessionID string
}

// sent when the webSocket connection fails
type WSConnectErrorMsg struct {
	err error
}

// sent when the webSocket connection is lost
type WSDisconnectedMsg struct{}
