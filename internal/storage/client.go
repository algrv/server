package storage

import (
	"context"
	"fmt"

	"github.com/algorave/server/internal/chunker"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type Client struct {
	pool *pgxpool.Pool
}

type DocumentChunk struct {
	PageName     string
	PageURL      string
	SectionTitle string
	Content      string
	Embedding    []float32
	Metadata     map[string]interface{}
}

func NewClient(ctx context.Context, connString string) (*Client, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Client{pool: pool}, nil
}

func (c *Client) Close() {
	c.pool.Close()
}

// deletes all existing chunks from the database
func (c *Client) ClearAllChunks(ctx context.Context) error {
	_, err := c.pool.Exec(ctx, "DELETE FROM doc_embeddings")
	if err != nil {
		return fmt.Errorf("failed to clear chunks: %w", err)
	}

	return nil
}

// inserts a single chunk with its embedding into the database
func (c *Client) InsertChunk(ctx context.Context, chunk chunker.Chunk, embedding []float32) error {
	query := `
		INSERT INTO doc_embeddings (page_name, page_url, section_title, content, embedding, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := c.pool.Exec(ctx,
		query,
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
	defer tx.Rollback(ctx)

	batch := &pgx.Batch{}
	query := `
		INSERT INTO doc_embeddings (page_name, page_url, section_title, content, embedding, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for i, chunk := range chunks {
		batch.Queue(query,
			chunk.PageName,
			chunk.PageURL,
			chunk.SectionTitle,
			chunk.Content,
			pgvector.NewVector(embeddings[i]),
			chunk.Metadata,
		)
	}

	br := tx.SendBatch(ctx, batch)

	for i := 0; i < len(chunks); i++ {
		_, err := br.Exec()
		if err != nil {
			br.Close()
			return fmt.Errorf("failed to insert chunk %d: %w", i, err)
		}
	}

	// must close batch results before committing, otherwise connection is still "busy"
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

	err := c.pool.QueryRow(ctx, "SELECT COUNT(*) FROM doc_embeddings").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get chunk count: %w", err)
	}

	return count, nil
}
