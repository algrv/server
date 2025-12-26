package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

type MessageModel struct {
	Role      string   `json:"role"`
	Content   string   `json:"content"`
	Metadata  string   `json:"metadata,omitempty"`
	Questions []string `json:"questions,omitempty"`
}

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
}

// returns a new code editor
func NewEditor() *EditorModel {
	ti := textinput.New()
	ti.Placeholder = "type your strudel ideas and press enter to get AI assistance..."
	ti.Focus()
	ti.CharLimit = 0
	ti.Prompt = "> "
	ti.PromptStyle.Foreground(colorGreen)
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorLightGray)
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorWhite)

	physicalWidth, physicalHeight, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		physicalWidth = 80
		physicalHeight = 24
	}

	ti.Width = physicalWidth - 4

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorLightGray)

	// create glamour renderer for markdown
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(physicalWidth-8),
	)
	if err != nil {
		// fallback to auto style
		renderer, _ = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(physicalWidth-8),
		)
	}

	vp := viewport.New(physicalWidth, physicalHeight-6)
	vp.YPosition = 0

	return &EditorModel{
		input:               ti,
		viewport:            vp,
		conversationHistory: []MessageModel{},
		isFetching:          false,
		spinner:             s,
		glamourRenderer:     renderer,
		width:               physicalWidth,
		height:              physicalHeight,
		ready:               true,
		shouldScrollBottom:  false,
	}
}

func (m *EditorModel) Update(msg tea.Msg) (*EditorModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s", "enter":
			if m.isFetching {
				return m, nil
			}

			query := m.input.Value()

			if strings.TrimSpace(query) != "" {
				m.isFetching = true
				m.input.SetValue("")

				// get current code from last assistant message
				currentCode := ""
				for i := len(m.conversationHistory) - 1; i >= 0; i-- {
					if m.conversationHistory[i].Role == "assistant" {
						currentCode = m.conversationHistory[i].Content
						break
					}
				}

				// send to agent (don't add user message yet - agent will add it)
				return m, tea.Batch(
					m.spinner.Tick,
					sendToAgent(query, currentCode, m.conversationHistory),
				)
			}

			return m, nil

		case "ctrl+l":
			m.input.SetValue("")
			m.conversationHistory = []MessageModel{}
			m.isFetching = false
			return m, nil

		case "pgup", "pgdown":
			// page up/down for viewport scrolling
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)

		case "ctrl+up", "ctrl+down":
			// ctrl+arrow keys for line-by-line viewport scrolling
			var scrollMsg tea.Msg
			if msg.String() == "ctrl+up" {
				scrollMsg = tea.KeyMsg{Type: tea.KeyUp}
			} else {
				scrollMsg = tea.KeyMsg{Type: tea.KeyDown}
			}
			m.viewport, cmd = m.viewport.Update(scrollMsg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)

		default:
			// pass other keys to input (including regular arrow keys for cursor movement)
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

	case AgentResponseMsg:
		m.isFetching = false

		// append both user query and assistant response to history
		m.conversationHistory = append(m.conversationHistory,
			MessageModel{
				Role:    "user",
				Content: msg.userQuery,
			},
			MessageModel{
				Role:      "assistant",
				Content:   msg.code,
				Metadata:  msg.metadata,
				Questions: msg.questions,
			},
		)

		// scroll to bottom to show new message
		m.shouldScrollBottom = true

		// refocus input
		m.input.Focus()

	case AgentErrorMsg:
		m.isFetching = false

		// append user query and error to history
		m.conversationHistory = append(m.conversationHistory,
			MessageModel{
				Role:    "user",
				Content: msg.userQuery,
			},
			MessageModel{
				Role:    "assistant",
				Content: fmt.Sprintf("Error: %v", msg.err),
			},
		)

		// scroll to bottom to show error
		m.shouldScrollBottom = true

		m.input.Focus()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 4

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-6)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 6
		}

		// update glamour renderer width
		renderer, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(msg.Width-8),
		)

		if err != nil {
			renderer, _ = glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(msg.Width-8),
			)
		}

		m.glamourRenderer = renderer

		return m, nil

	case spinner.TickMsg:
		if m.isFetching {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.MouseMsg:
		// enable mouse wheel scrolling
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	// update viewport for other messages
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *EditorModel) View() string {
	var b strings.Builder

	// render chat history
	chatContent := m.renderChatHistory()
	m.viewport.SetContent(chatContent)

	// scroll to bottom if needed
	if m.shouldScrollBottom {
		m.viewport.GotoBottom()
		m.shouldScrollBottom = false
	}

	// header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPurple).
		Render("")

	headerLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		header,
	)

	b.WriteString(headerLine)
	b.WriteString("\n\n")

	// viewport with chat history
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// input prompt
	if m.isFetching {
		fetchingMsg := lipgloss.NewStyle().
			Foreground(colorGray).
			Italic(true).
			Render(m.spinner.View() + " thinking...")
		b.WriteString(fetchingMsg)
		b.WriteString("\n\n")
	} else {
		b.WriteString(m.input.View())
		b.WriteString("\n\n")
	}

	help := lipgloss.NewStyle().
		Foreground(colorGray).
		Render("send: enter | scroll: pgup/pgdn, ctrl+↑/↓ | clear: ctrl+l | exit: ctrl+c")

	helpLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		lipgloss.NewStyle().Foreground(colorYellow).Render("? "),
		help,
	)

	b.WriteString(helpLine)
	b.WriteString("\n")

	return b.String()
}

func (m *EditorModel) renderChatHistory() string {
	var b strings.Builder

	if len(m.conversationHistory) == 0 {
		return b.String()
	}

	for i, msg := range m.conversationHistory {
		if i > 0 {
			b.WriteString("\n")
		}

		if msg.Role == "user" {
			// user indicator
			userIndicator := lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true).
				Render("> ")

			// user message
			userPrompt := lipgloss.NewStyle().
				Foreground(colorDarkGray).
				Bold(true).
				Render("you")

			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, userIndicator, userPrompt))
			b.WriteString("\n\n")

			userContent := lipgloss.NewStyle().
				Foreground(colorWhite).
				Render(msg.Content)
			b.WriteString(userContent)
			b.WriteString("\n\n")

		} else if msg.Role == "assistant" {
			// agent indicator
			agentIndicator := lipgloss.NewStyle().
				Foreground(colorPurple).
				Bold(true).
				Render("> ")

			// agent message
			agentPrompt := lipgloss.NewStyle().
				Foreground(colorDarkGray).
				Bold(true).
				Render("algorave")

			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, agentIndicator, agentPrompt))
			b.WriteString("\n")

			// render code in a styled box
			if len(msg.Content) > 0 {
				// check if this is an error message
				isError := strings.HasPrefix(msg.Content, "Error:")

				// use lipgloss box with syntax highlighting from glamour
				var codeContent string
				if m.glamourRenderer != nil {
					// render as markdown code block for syntax highlighting
					markdown := fmt.Sprintf("```javascript\n%s\n```", msg.Content)
					glamourOutput, err := m.glamourRenderer.Render(markdown)
					if err == nil && glamourOutput != "" {
						codeContent = strings.TrimSpace(glamourOutput)
					} else {
						codeContent = msg.Content
					}
				} else {
					codeContent = msg.Content
				}

				// choose border color based on error status
				borderColor := colorVeryDarkGray
				if isError {
					borderColor = colorRed
				}

				innerBox := lipgloss.NewStyle().
					Render(codeContent)

					// always wrap in bordered box
				boxed := lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(borderColor).
					Render(innerBox)

				b.WriteString(boxed)
				b.WriteString("\n")
			} else if len(msg.Questions) == 0 {
				// no content - show placeholder
				b.WriteString(lipgloss.NewStyle().
					Foreground(colorGray).
					Italic(true).
					Render("no response generated"))
				b.WriteString("\n")
			}

			// show metadata if available
			if msg.Metadata != "" {
				metadata := lipgloss.NewStyle().
					Foreground(colorGray).
					Italic(true).
					Render(msg.Metadata)
				b.WriteString(metadata)
				b.WriteString("\n")
			}

			// show clarifying questions if available
			if len(msg.Questions) > 0 {
				b.WriteString("\n")
				questionsHeader := lipgloss.NewStyle().
					Foreground(colorYellow).
					Bold(true).
					Render("?? clarifying questions:")
				b.WriteString(questionsHeader)
				b.WriteString("\n\n")

				for i, q := range msg.Questions {
					question := lipgloss.NewStyle().
						Foreground(colorLightGray).
						Render(fmt.Sprintf("  %d. %s", i+1, q))
					b.WriteString(question)
					b.WriteString("\n")
				}

				b.WriteString("\n")
			}

			// add spacing after assistant message
			if msg.Metadata != "" || len(msg.Questions) > 0 {
				b.WriteString("\n\n")
			}
		}
	}

	return b.String()
}

func (m *EditorModel) GetCode() string {
	// return the last assistant message
	for i := len(m.conversationHistory) - 1; i >= 0; i-- {
		if m.conversationHistory[i].Role == "assistant" {
			return m.conversationHistory[i].Content
		}
	}
	return ""
}

type AgentResponseMsg struct {
	userQuery string
	code      string
	metadata  string
	questions []string
}

type AgentErrorMsg struct {
	userQuery string
	err       error
}
