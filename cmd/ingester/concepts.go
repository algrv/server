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

// chunks and embeds teaching concept files from MDX
func IngestConcepts(cfg *config.Config, db *pgxpool.Pool, flags config.Flags) error {
	ctx := context.Background()

	logger.Info("starting concepts ingestion", "path", flags.Path, "clear", flags.Clear)

	// use shared connection pool
	storageClient := storage.NewClientFromPool(db)
	defer storageClient.Close() // no-op since we don't own the pool

	// note: we don't clear all chunks here, as concepts share the same table as docs
	// if clear flag is set, we would have already cleared in the docs ingestion
	if flags.Clear {
		logger.Info("clearing existing concept chunks")
		logger.Warn("clear flag ignored for concepts to preserve documentation chunks")
	}

	// chunk all MDX files in concepts directory
	logger.Info("chunking concept files", "path", flags.Path)
	chunks, errors := chunker.ChunkDocuments(flags.Path)

	if len(errors) > 0 {
		logger.Warn("encountered errors while chunking concepts", "error_count", len(errors))
		for _, err := range errors {
			logger.Warn("chunking error", "error", err)
		}
	}

	if len(chunks) == 0 {
		logger.Info("no concept chunks generated, skipping")
		return nil
	}

	logger.Info("generated concept chunks", "count", len(chunks))

	// create OpenAI embedder
	embedder := llm.NewOpenAIEmbedder(llm.OpenAIConfig{
		APIKey: cfg.OpenAIKey,
		Model:  "text-embedding-3-small",
	})

	// generate embeddings for all concept chunks
	logger.Info("generating embeddings for concept chunks")
	texts := make([]string, len(chunks))

	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	embeddings, err := embedder.GenerateEmbeddings(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	logger.Info("generated embeddings", "count", len(embeddings))

	// insert concept chunks with embeddings into database
	logger.Info("inserting concept chunks into database")
	if err := storageClient.InsertChunksBatch(ctx, chunks, embeddings); err != nil {
		return fmt.Errorf("failed to insert concept chunks: %w", err)
	}

	// verify insertion
	count, err := storageClient.GetChunkCount(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify chunk count: %w", err)
	}

	logger.Info("successfully ingested concepts",
		"chunks_inserted", len(chunks),
		"total_chunks", count,
	)

	return nil
}
