package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Welcome struct {
	mode     string
	input    string
	commands []Command
}

type Command struct {
	Name        string
	Description string
	Available   bool
}

// returns a new welcome screen
func NewWelcome(mode string) *Welcome {
	commands := []Command{
		{Name: "start", Description: "start the algorave server", Available: true},
		{Name: "ingest", Description: "run documentation ingester", Available: mode == "development"},
		{Name: "editor", Description: "interactive code editor", Available: true},
		{Name: "quit", Description: "exit algorave", Available: true},
	}

	return &Welcome{
		mode:     mode,
		commands: commands,
	}
}

func (m *Welcome) Update(msg tea.Msg) (*Welcome, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m, m.executeCommand()
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.input += msg.String()
			}
		}

	case ServerStartedMsg:
		m.input = ""
		return m, nil

	case IngesterCompleteMsg:
		m.input = ""
		return m, nil
	}

	return m, nil
}

func (m *Welcome) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(logo))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("create music with human language"))
	b.WriteString("\n\n")

	modeText := fmt.Sprintf("mode: %s", strings.ToUpper(m.mode))
	b.WriteString(infoStyle.Render(modeText))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render("commands:"))
	b.WriteString("\n\n")

	for _, cmd := range m.commands {
		if !cmd.Available {
			continue
		}
		line := fmt.Sprintf("  %s %s",
			commandStyle.Render(cmd.Name),
			commandDescStyle.Render("- "+cmd.Description),
		)
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	prompt := promptStyle.Render("> ")
	input := inputStyle.Render(m.input + "_")
	b.WriteString(prompt + input)
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("type a command and press enter. press ctrl+c to quit."))

	return b.String()
}

func (m *Welcome) executeCommand() tea.Cmd {
	cmd := strings.TrimSpace(m.input)

	switch cmd {
	case "quit":
		return tea.Quit

	case "start":
		return startServer

	case "ingest":
		if m.mode == "development" {
			return runIngester
		}
		return func() tea.Msg {
			return ErrorMsg{err: fmt.Errorf("ingester not available in production mode.")}
		}

	case "editor":
		return func() tea.Msg {
			return EnterEditorMsg{}
		}

	default:
		if cmd != "" {
			return func() tea.Msg {
				return ErrorMsg{err: fmt.Errorf("unknown command: %s.", cmd)}
			}
		}
		return nil
	}
}

type ServerStartedMsg struct{}
type IngesterCompleteMsg struct{}
