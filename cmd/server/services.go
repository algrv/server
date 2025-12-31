package main

import (
	"context"
	"fmt"

	"github.com/algoraveai/server/internal/agent"
	"github.com/algoraveai/server/internal/config"
	"github.com/algoraveai/server/internal/llm"
	"github.com/algoraveai/server/internal/retriever"
	"github.com/algoraveai/server/internal/storage"
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
	agentClient := agent.New(retrieverClient, llmClient)

	return &Services{
		Agent:     agentClient,
		LLM:       llmClient,
		Retriever: retrieverClient,
		Storage:   storageClient,
	}, nil
}
