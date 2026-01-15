package main

import (
	"fmt"
	"os"

	"codeberg.org/algorave/server/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	env := os.Getenv("ALGORAVE_ENV")

	if env == "" {
		env = "development"
	}

	app := tui.NewApp(env)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Printf("error running algorave: %v\n", err)
		os.Exit(1)
	}
}
