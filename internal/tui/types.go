package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
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
	agentClient         *AgentClient
}

// sent when the agent completes a request
type AgentResponseMsg struct {
	userQuery      string
	code           string
	metadata       string
	questions      []string
	isCodeResponse bool
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
