package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func NewApp(mode string) *Model {
	return &Model{
		state:   StateWelcome,
		mode:    mode,
		welcome: NewWelcome(mode),
		editor:  NewEditor(),
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// only quit from welcome screen, not from editor
		if msg.String() == "ctrl+c" && m.state == StateWelcome {
			return m, tea.Quit
		}

		// in editor, ctrl+c should go back to welcome
		if msg.String() == "ctrl+c" && m.state == StateEditor {
			m.state = StateWelcome
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.state == StateEditor {
			m.editor, _ = m.editor.Update(msg)
		}

	case ErrorMsg:
		m.err = msg.err
		return m, nil

	case EnterEditorMsg:
		m.state = StateEditor
		return m, m.editor.Init()
	}

	switch m.state {
	case StateWelcome:
		return m.updateWelcome(msg)

	case StateEditor:
		return m.updateEditor(msg)

	default:
		return m, nil
	}
}

func (m *Model) View() string {
	if m.err != nil {
		return errorView(m.err)
	}

	switch m.state {
	case StateWelcome:
		return m.welcome.View()

	case StateEditor:
		return m.editor.View()

	default:
		return "Unknown state"
	}
}

func (m *Model) updateWelcome(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.welcome, cmd = m.welcome.Update(msg)

	return m, cmd
}

func (m *Model) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)

	return m, cmd
}

func errorView(err error) string {
	return fmt.Sprintf("\n  Error: %v\n\n  Press Ctrl+C to exit\n", err)
}
