package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type EditorModel struct {
	input               textinput.Model
	width               int
	height              int
	outputCode          string
	outputMetadata      string
	clarifyingQuestions []string
	conversationHistory []Message
	showInitialMessage  bool
	isFetching          bool
}

// returns a new code editor
func NewEditorModel() *EditorModel {
	ti := textinput.New()
	ti.Placeholder = "type your musical ideas here..."
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 80
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorLightGray)
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorWhite)

	return &EditorModel{
		input:               ti,
		conversationHistory: []Message{},
		showInitialMessage:  true,
		isFetching:          false,
	}
}

func (m *EditorModel) Update(msg tea.Msg) (*EditorModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s", "enter":
			query := m.input.Value()
			if strings.TrimSpace(query) != "" {
				m.isFetching = true
				m.input.SetValue("")

				// Append user message to history
				m.conversationHistory = append(m.conversationHistory, Message{
					Role:    "user",
					Content: query,
				})

				return m, sendToAgent(query, m.outputCode, m.conversationHistory)
			}
			return m, nil

		case "ctrl+l":
			m.input.SetValue("")
			m.outputCode = ""
			m.outputMetadata = ""
			m.clarifyingQuestions = nil
			m.conversationHistory = []Message{}
			m.showInitialMessage = true
			m.isFetching = false
			return m, nil
		}

	case AgentResponseMsg:
		m.outputCode = msg.code
		m.outputMetadata = msg.metadata
		m.clarifyingQuestions = msg.questions
		m.showInitialMessage = false
		m.isFetching = false

		// Append assistant message to history
		if msg.code != "" {
			m.conversationHistory = append(m.conversationHistory, Message{
				Role:    "assistant",
				Content: msg.code,
			})
		}

		m.input.Focus()
		return m, nil

	case AgentErrorMsg:
		m.outputCode = fmt.Sprintf("Error: %v", msg.err)
		m.outputMetadata = ""
		m.clarifyingQuestions = nil
		m.showInitialMessage = false
		m.isFetching = false
		m.input.Focus()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 10
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *EditorModel) View() string {
	var b strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorWhite).
		Render("EDITOR MODE")

	help := lipgloss.NewStyle().
		Foreground(colorGray).
		Render("[Enter/Ctrl+S: Send] [Ctrl+L: Clear] [Ctrl+C: Exit]")

	headerLine := lipgloss.JoinHorizontal(lipgloss.Left,
		header,
		strings.Repeat(" ", max(0, m.width-len("EDITOR MODE")-len(help)-2)),
		help,
	)

	b.WriteString(headerLine)
	b.WriteString("\n\n")

	// Output/Editor Box
	outputBoxContent := ""
	if m.showInitialMessage {
		outputBoxContent = lipgloss.NewStyle().
			Foreground(colorGray).
			Italic(true).
			Render("ready! type your musical ideas below and press enter to get AI assistance.")
	} else if m.outputCode != "" {
		outputBoxContent = m.outputCode
	}

	outputBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorGray).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Width(m.width - 4).
		Height(10).
		Padding(1).
		Render(outputBoxContent)

	b.WriteString(outputBox)
	b.WriteString("\n")

	// Show metadata below output box if available
	if m.outputMetadata != "" {
		b.WriteString(lipgloss.NewStyle().
			Foreground(colorGray).
			Italic(true).
			Render(m.outputMetadata))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Questions Box (if any)
	if len(m.clarifyingQuestions) > 0 {
		var questionsContent strings.Builder
		questionsContent.WriteString(lipgloss.NewStyle().
			Foreground(colorLightGray).
			Bold(true).
			Render("👾 Questions:"))
		questionsContent.WriteString("\n\n")
		for _, q := range m.clarifyingQuestions {
			questionsContent.WriteString(lipgloss.NewStyle().
				Foreground(colorWhite).
				Render("  • " + q))
			questionsContent.WriteString("\n")
		}

		questionsBox := lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorGray).
			Width(m.width - 4).
			Padding(1).
			Render(questionsContent.String())

		b.WriteString(questionsBox)
		b.WriteString("\n\n")
	}

	// Input Box
	inputBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorGray).
		Width(m.width - 4).
		Padding(0, 1).
		Render(m.input.View())

	b.WriteString(inputBox)
	b.WriteString("\n")

	// Status Line
	statusText := ""
	if m.isFetching {
		statusText = lipgloss.NewStyle().
			Foreground(colorGray).
			Italic(true).
			Render("⏳ fetching from agent...")
	}
	b.WriteString(statusText)

	return b.String()
}

func (m *EditorModel) GetCode() string {
	return m.outputCode
}

type AgentResponseMsg struct {
	code      string
	metadata  string
	questions []string
}

type AgentErrorMsg struct {
	err error
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
