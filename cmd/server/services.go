package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"codeberg.org/algorave/server/internal/agent"
	"codeberg.org/algorave/server/internal/attribution"
	"codeberg.org/algorave/server/internal/config"
	"codeberg.org/algorave/server/internal/llm"
	"codeberg.org/algorave/server/internal/logger"
	"codeberg.org/algorave/server/internal/retriever"
	"codeberg.org/algorave/server/internal/storage"
	"codeberg.org/algorave/server/internal/strudel"
	"github.com/jackc/pgx/v5/pgxpool"
)

// creates and configures all service clients
func InitializeServices(_ *config.Config, db *pgxpool.Pool) (*Services, error) {
	llmClient, err := llm.NewLLM(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	retrieverClient := retriever.New(db, llmClient)
	storageClient := &storage.Client{}

	// initialize validator (optional/continues without if unavailable)
	var validator *strudel.Validator
	if scriptDir := findValidatorScriptDir(); scriptDir != "" {
		v, err := strudel.NewValidator(scriptDir)
		if err != nil {
			logger.Warn("strudel validator unavailable, continuing without validation", "error", err)
		} else {
			validator = v
			logger.Info("strudel validator initialized")
		}
	} else {
		logger.Warn("strudel validator script not found, continuing without validation")
	}

	agentClient := agent.NewWithValidator(retrieverClient, llmClient, validator)
	attrService := attribution.New(db)

	return &Services{
		Agent:       agentClient,
		Attribution: attrService,
		LLM:         llmClient,
		Retriever:   retrieverClient,
		Storage:     storageClient,
		Validator:   validator,
	}, nil
}

// locates the validator script directory
func findValidatorScriptDir() string {
	candidates := []string{
		"scripts/validate-strudel",
		"/app/scripts/validate-strudel",
		filepath.Join(os.Getenv("HOME"), "scripts/validate-strudel"),
	}

	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "validator.js")); err == nil {
			return dir
		}
	}

	return ""
}
