package retriever

import (
	"context"
	"fmt"
	"sync"

	"github.com/algorave/server/internal/llm"
	"github.com/algorave/server/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// New creates a retriever client with injected dependencies.
// The caller owns the lifecycle of db, embedder, and transformer.
func New(db *pgxpool.Pool, llm llm.LLM) *Client {
	return &Client{
		db:   db,
		llm:  llm,
		topK: defaultTopK,
	}
}

// NewWithTopK creates a retriever with a custom topK value
func NewWithTopK(db *pgxpool.Pool, llm llm.LLM, topK int) *Client {
	return &Client{
		db:   db,
		llm:  llm,
		topK: topK,
	}
}

// VectorSearch performs a vector similarity search on doc_embeddings
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

// SearchExamples performs a vector similarity search on example_strudels
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

// BM25Search performs keyword-based full-text search on doc_embeddings
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

		// Convert BM25 rank to similarity score (0-1 range)
		result.Similarity = float32(rank)
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating BM25 rows: %w", err)
	}

	return results, nil
}

// BM25SearchExamples performs keyword-based full-text search on example_strudels
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
			&result.Title,
			&result.Description,
			&result.Code,
			&result.Tags,
			&result.URL,
			&rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan BM25 example row: %w", err)
		}

		// Convert BM25 rank to similarity score
		result.Similarity = float32(rank)
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating BM25 example rows: %w", err)
	}

	return results, nil
}

// HybridSearchDocs implements hybrid search (vector + BM25) for documentation
func (c *Client) HybridSearchDocs(ctx context.Context, userQuery, editorState string, topK int) ([]SearchResult, error) {
	// transform query to add technical keywords for vector search
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

	// vector search (70% weight) - uses transformed query for semantic matching
	wg.Add(1)
	go func() {
		defer wg.Done()
		vectorResults, vectorErr = c.VectorSearch(ctx, searchQuery, searchK)
	}()

	// BM25 search (30% weight) - uses original query for exact keyword matching
	wg.Add(1)
	go func() {
		defer wg.Done()
		bm25Results, bm25Err = c.BM25Search(ctx, userQuery, searchK)
	}()

	wg.Wait()

	// check for errors
	if vectorErr != nil {
		return nil, fmt.Errorf("vector search failed: %w", vectorErr)
	}

	if bm25Err != nil {
		// don't fail completely, just log and use vector only
		logger.Warn("BM25 search failed, using vector only", "error", bm25Err)
		bm25Results = []SearchResult{}
	}

	// merge vector and BM25 results with weighted scoring (70% vector, 30% BM25)
	merged := mergeVectorAndBM25Docs(vectorResults, bm25Results, topK)

	// fetch special chunks and organize by page
	organized, err := c.organizeByPage(ctx, merged)
	if err != nil {
		return nil, fmt.Errorf("failed to organize results: %w", err)
	}

	return organized, nil
}

// HybridSearchExamples implements hybrid search (vector + BM25) for examples
func (c *Client) HybridSearchExamples(ctx context.Context, userQuery, editorState string, topK int) ([]ExampleResult, error) {
	// transform query to add technical keywords for vector search
	searchQuery, err := c.llm.TransformQuery(ctx, userQuery)
	if err != nil {
		logger.Warn("query transformation failed, using original query", "error", err)
		searchQuery = userQuery
	}

	// run vector and BM25 searches in parallel
	searchK := topK + 5 // get extra results for better merging

	var vectorResults, bm25Results []ExampleResult
	var vectorErr, bm25Err error
	var wg sync.WaitGroup

	// vector search (70% weight) - uses transformed query for semantic matching
	wg.Add(1)
	go func() {
		defer wg.Done()
		vectorResults, vectorErr = c.SearchExamples(ctx, searchQuery, searchK)
	}()

	// BM25 search (30% weight) - uses original query for exact keyword matching
	wg.Add(1)
	go func() {
		defer wg.Done()
		bm25Results, bm25Err = c.BM25SearchExamples(ctx, userQuery, searchK)
	}()

	wg.Wait()

	// check for errors
	if vectorErr != nil {
		return nil, fmt.Errorf("vector search failed: %w", vectorErr)
	}

	if bm25Err != nil {
		// don't fail completely, just log and use vector only
		logger.Warn("BM25 search failed, using vector only", "error", bm25Err)
		bm25Results = []ExampleResult{}
	}

	// merge vector and BM25 results with weighted scoring (70% vector, 30% BM25)
	merged := mergeVectorAndBM25Examples(vectorResults, bm25Results, topK)

	return merged, nil
}
