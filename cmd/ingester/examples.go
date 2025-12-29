package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/algorave/server/internal/config"
	"github.com/algorave/server/internal/examples"
	"github.com/algorave/server/internal/llm"
	"github.com/algorave/server/internal/logger"
	"github.com/algorave/server/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

// loads and embeds code examples from a JSON file
func IngestExamples(cfg *config.Config, _ *pgxpool.Pool, flags config.Flags) error {
	ctx := context.Background()
	logger.Info("starting examples ingestion", "path", flags.Path, "clear", flags.Clear)

	// create storage client
	storageClient, err := storage.NewClient(ctx, cfg.SupabaseConnString)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}

	defer storageClient.Close()

	// clear existing examples if requested
	if flags.Clear {
		logger.Info("clearing existing examples")
		if err := storageClient.ClearAllExamples(ctx); err != nil {
			return fmt.Errorf("failed to clear existing examples: %w", err)
		}

		logger.Info("cleared existing examples")
	}

	// load JSON file
	logger.Info("loading examples from JSON", "path", flags.Path)
	data, err := os.ReadFile(flags.Path)

	if err != nil {
		return fmt.Errorf("failed to read examples file: %w", err)
	}

	// parse raw examples
	var rawExamples []examples.RawExample

	if err := json.Unmarshal(data, &rawExamples); err != nil {
		return fmt.Errorf("failed to parse examples JSON: %w", err)
	}

	logger.Info("loaded raw examples", "count", len(rawExamples))

	// process raw examples into enriched examples
	processedExamples, err := examples.ProcessRawExamples(rawExamples)
	if err != nil {
		return fmt.Errorf("failed to process examples: %w", err)
	}

	logger.Info("processed examples", "count", len(processedExamples))

	// create OpenAI embedder
	embedder := llm.NewOpenAIEmbedder(llm.OpenAIConfig{
		APIKey: cfg.OpenAIKey,
		Model:  "text-embedding-3-small",
	})

	// generate embeddings for all examples
	logger.Info("generating embeddings for examples")
	texts := make([]string, len(processedExamples))

	for i, example := range processedExamples {
		texts[i] = fmt.Sprintf("%s\n%s\n%s", example.Title, example.Description, example.Code)
	}

	embeddings, err := embedder.GenerateEmbeddings(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	logger.Info("generated embeddings", "count", len(embeddings))

	// insert examples with embeddings into database
	logger.Info("inserting examples into database")
	if err := storageClient.InsertExamplesBatch(ctx, processedExamples, embeddings); err != nil {
		return fmt.Errorf("failed to insert examples: %w", err)
	}

	// verify insertion
	count, err := storageClient.GetExampleCount(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify example count: %w", err)
	}

	logger.Info("successfully ingested examples",
		"examples_inserted", len(processedExamples),
		"total_examples", count,
	)

	return nil
}
