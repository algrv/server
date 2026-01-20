package main

import (
	"fmt"
	"os"

	"codeberg.org/algopatterns/server/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	env := os.Getenv("ALGOPATTERNS_ENV")

	if env == "" {
		env = "development"
	}

	app := tui.NewApp(env)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Printf("error running algopatterns: %v\n", err)
		os.Exit(1)
	}
}
