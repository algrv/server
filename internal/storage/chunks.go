package storage

import (
	"context"
	"errors"
	"fmt"

	"codeberg.org/algorave/server/internal/chunker"
	"codeberg.org/algorave/server/internal/logger"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

// deletes all existing chunks from the database
func (c *Client) ClearAllChunks(ctx context.Context) error {
	_, err := c.pool.Exec(ctx, deleteAllChunksQuery)
	if err != nil {
		return fmt.Errorf("failed to clear chunks: %w", err)
	}

	return nil
}

// inserts a single chunk with its embedding into the database
func (c *Client) InsertChunk(ctx context.Context, chunk chunker.Chunk, embedding []float32) error {
	_, err := c.pool.Exec(ctx,
		insertChunkQuery,
		chunk.PageName,
		chunk.PageURL,
		chunk.SectionTitle,
		chunk.Content,
		pgvector.NewVector(embedding),
		chunk.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to insert chunk: %w", err)
	}

	return nil
}

// multiple chunks in a single transaction
func (c *Client) InsertChunksBatch(ctx context.Context, chunks []chunker.Chunk, embeddings [][]float32) error {
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunks and embeddings length mismatch")
	}

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// defer rollback - will be no-op if commit succeeds
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			logger.Warn("failed to rollback transaction", "error", err)
		}
	}()

	batch := &pgx.Batch{}

	for i, chunk := range chunks {
		batch.Queue(insertChunkQuery,
			chunk.PageName,
			chunk.PageURL,
			chunk.SectionTitle,
			chunk.Content,
			pgvector.NewVector(embeddings[i]),
			chunk.Metadata,
		)
	}

	br := tx.SendBatch(ctx, batch)

	for i := range len(chunks) {
		_, err := br.Exec()
		if err != nil {
			br.Close() //nolint:errcheck,gosec // G104: error path cleanup
			return fmt.Errorf("failed to insert chunk %d: %w", i, err)
		}
	}

	if err := br.Close(); err != nil {
		return fmt.Errorf("failed to close batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// returns the total number of chunks in the database
func (c *Client) GetChunkCount(ctx context.Context) (int, error) {
	var count int

	err := c.pool.QueryRow(ctx, getChunkCountQuery).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get chunk count: %w", err)
	}

	return count, nil
}
