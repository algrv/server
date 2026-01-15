package main

import (
	"context"
	"fmt"

	"codeberg.org/algorave/server/internal/chunker"
	"codeberg.org/algorave/server/internal/config"
	"codeberg.org/algorave/server/internal/llm"
	"codeberg.org/algorave/server/internal/logger"
	"codeberg.org/algorave/server/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

// chunks and embeds documentation files from the specified path
func IngestDocs(cfg *config.Config, db *pgxpool.Pool, flags config.Flags) error {
	ctx := context.Background()
	logger.Info("starting docs ingestion", "path", flags.Path, "clear", flags.Clear)

	// use shared connection pool
	storageClient := storage.NewClientFromPool(db)
	defer storageClient.Close() // no-op since we don't own the pool

	// clear existing docs if requested
	if flags.Clear {
		logger.Info("clearing existing documentation chunks")

		if err := storageClient.ClearAllChunks(ctx); err != nil {
			return fmt.Errorf("failed to clear existing chunks: %w", err)
		}

		logger.Info("cleared existing chunks")
	}

	// chunk all markdown files in directory
	logger.Info("chunking documentation files", "path", flags.Path)
	chunks, errors := chunker.ChunkDocuments(flags.Path)

	if len(errors) > 0 {
		logger.Warn("encountered errors while chunking", "error_count", len(errors))

		for _, err := range errors {
			logger.Warn("chunking error", "error", err)
		}
	}

	if len(chunks) == 0 {
		return fmt.Errorf("no chunks generated from documentation")
	}

	logger.Info("generated chunks", "count", len(chunks))

	// create OpenAI embedder
	embedder := llm.NewOpenAIEmbedder(llm.OpenAIConfig{
		APIKey: cfg.OpenAIKey,
		Model:  "text-embedding-3-small",
	})

	// generate embeddings for all chunks
	logger.Info("generating embeddings for chunks")
	texts := make([]string, len(chunks))

	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	embeddings, err := embedder.GenerateEmbeddings(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	logger.Info("generated embeddings", "count", len(embeddings))

	// insert chunks with embeddings into database
	logger.Info("inserting chunks into database")
	if err := storageClient.InsertChunksBatch(ctx, chunks, embeddings); err != nil {
		return fmt.Errorf("failed to insert chunks: %w", err)
	}

	// verify insertion
	count, err := storageClient.GetChunkCount(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify chunk count: %w", err)
	}

	logger.Info("successfully ingested documentation",
		"chunks_inserted", len(chunks),
		"total_chunks", count,
	)

	return nil
}
