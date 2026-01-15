package retriever

import (
	"context"
	"fmt"
	"sync"

	"codeberg.org/algorave/server/internal/llm"
	"codeberg.org/algorave/server/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

func New(db *pgxpool.Pool, llm llm.LLM) *Client {
	return &Client{
		db:   db,
		llm:  llm,
		topK: defaultTopK,
	}
}

func NewWithTopK(db *pgxpool.Pool, llm llm.LLM, topK int) *Client {
	return &Client{
		db:   db,
		llm:  llm,
		topK: topK,
	}
}

func (c *Client) VectorSearch(ctx context.Context, queryText string, topK int) ([]SearchResult, error) {
	embedding, err := c.llm.GenerateEmbedding(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	rows, err := c.db.Query(ctx, vectorSearchQuery, pgvector.NewVector(embedding), topK)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}

	defer rows.Close()
	var results []SearchResult

	for rows.Next() {
		var result SearchResult

		err := rows.Scan(
			&result.ID,
			&result.PageName,
			&result.PageURL,
			&result.SectionTitle,
			&result.Content,
			&result.Similarity,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

func (c *Client) SearchExamples(ctx context.Context, queryText string, topK int) ([]ExampleResult, error) {
	embedding, err := c.llm.GenerateEmbedding(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	rows, err := c.db.Query(ctx, searchExamplesQuery, pgvector.NewVector(embedding), topK)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}

	defer rows.Close()
	var results []ExampleResult

	for rows.Next() {
		var result ExampleResult

		err := rows.Scan(
			&result.ID,
			&result.Title,
			&result.Description,
			&result.Code,
			&result.Tags,
			&result.UserID,
			&result.AuthorName,
			&result.URL,
			&result.Similarity,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

func (c *Client) BM25Search(ctx context.Context, queryText string, topK int) ([]SearchResult, error) {
	rows, err := c.db.Query(ctx, bm25SearchDocsQuery, queryText, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to execute BM25 search: %w", err)
	}

	defer rows.Close()
	var results []SearchResult

	for rows.Next() {
		var result SearchResult
		var rank float64

		err := rows.Scan(
			&result.ID,
			&result.PageName,
			&result.PageURL,
			&result.SectionTitle,
			&result.Content,
			&rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan BM25 row: %w", err)
		}

		result.Similarity = float32(rank)
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating BM25 rows: %w", err)
	}

	return results, nil
}

func (c *Client) BM25SearchExamples(ctx context.Context, queryText string, topK int) ([]ExampleResult, error) {
	rows, err := c.db.Query(ctx, bm25SearchExamplesQuery, queryText, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to execute BM25 example search: %w", err)
	}

	defer rows.Close()
	var results []ExampleResult

	for rows.Next() {
		var result ExampleResult
		var rank float64

		err := rows.Scan(
			&result.ID,
			&result.UserID,
			&result.Title,
			&result.Description,
			&result.Code,
			&result.Tags,
			&result.AuthorName,
			&result.URL,
			&rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan BM25 example row: %w", err)
		}

		result.Similarity = float32(rank)
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating BM25 example rows: %w", err)
	}

	return results, nil
}

func (c *Client) HybridSearchDocs(ctx context.Context, userQuery, _ string, topK int) ([]SearchResult, error) {
	searchQuery, err := c.llm.TransformQuery(ctx, userQuery)
	if err != nil {
		logger.Warn("query transformation failed, using original query", "error", err)
		searchQuery = userQuery
	}

	// run vector and BM25 searches in parallel
	searchK := topK + 5 // get extra results for better merging

	var vectorResults, bm25Results []SearchResult
	var vectorErr, bm25Err error
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		vectorResults, vectorErr = c.VectorSearch(ctx, searchQuery, searchK)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		bm25Results, bm25Err = c.BM25Search(ctx, userQuery, searchK)
	}()

	wg.Wait()

	if vectorErr != nil {
		return nil, fmt.Errorf("vector search failed: %w", vectorErr)
	}

	if bm25Err != nil {
		logger.Warn("BM25 search failed, using vector only", "error", bm25Err)
		bm25Results = []SearchResult{}
	}

	merged := mergeVectorAndBM25Docs(vectorResults, bm25Results, topK)

	organized, err := c.organizeByPage(ctx, merged)
	if err != nil {
		return nil, fmt.Errorf("failed to organize results: %w", err)
	}

	return organized, nil
}

// hybrid search (vector + BM25) for strudel examples
func (c *Client) HybridSearchExamples(ctx context.Context, userQuery, _ string, topK int) ([]ExampleResult, error) {
	searchQuery, err := c.llm.TransformQuery(ctx, userQuery)
	if err != nil {
		logger.Warn("query transformation failed, using original query", "error", err)
		searchQuery = userQuery
	}

	searchK := topK + 5 // get extra results for better merging

	var vectorResults, bm25Results []ExampleResult
	var vectorErr, bm25Err error
	var wg sync.WaitGroup

	// vector search (70% weight)
	wg.Add(1)
	go func() {
		defer wg.Done()
		vectorResults, vectorErr = c.SearchExamples(ctx, searchQuery, searchK)
	}()

	// BM25 search (30% weight)
	wg.Add(1)
	go func() {
		defer wg.Done()
		bm25Results, bm25Err = c.BM25SearchExamples(ctx, userQuery, searchK)
	}()

	wg.Wait()

	if vectorErr != nil {
		return nil, fmt.Errorf("vector search failed: %w", vectorErr)
	}

	if bm25Err != nil {
		logger.Warn("BM25 search failed, using vector only", "error", bm25Err)
		bm25Results = []ExampleResult{}
	}

	merged := mergeVectorAndBM25Examples(vectorResults, bm25Results, topK)

	return merged, nil
}
