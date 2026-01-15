package tui

import (
	"fmt"
	"os"
	"os/exec"

	"codeberg.org/algorave/server/internal/logger"
	tea "github.com/charmbracelet/bubbletea"
)

func startServer() tea.Msg {
	serverPath := "bin/server"

	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", serverPath, "cmd/server/main.go", "cmd/server/server.go", "cmd/server/services.go", "cmd/server/types.go", "cmd/server/routes.go", "cmd/server/middleware.go")
		if err := buildCmd.Run(); err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to build server: %w", err)}
		}
	}

	cmd := exec.Command(serverPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		if err := cmd.Run(); err != nil {
			logger.ErrorErr(err, "server error")
		}
	}()

	return ServerStartedMsg{}
}

func runIngester() tea.Msg {
	ingesterPath := "bin/ingester"

	if _, err := os.Stat(ingesterPath); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", ingesterPath, "cmd/ingester/main.go", "cmd/ingester/docs.go", "cmd/ingester/examples.go", "cmd/ingester/concepts.go")

		if err := buildCmd.Run(); err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to build ingester: %w", err)}
		}
	}

	cmd := exec.Command(ingesterPath, "all", "--clear")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return ErrorMsg{err: fmt.Errorf("ingester failed: %w", err)}
	}

	return IngesterCompleteMsg{}
}
