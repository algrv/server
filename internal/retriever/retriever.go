package retriever

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/algorave/server/internal/llm"
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

	query := `
		SELECT
			id::text,
			page_name,
			page_url,
			section_title,
			content,
			similarity
		FROM search_docs($1, $2)
	`

	rows, err := c.db.Query(ctx, query, pgvector.NewVector(embedding), topK)
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

	query := `
		SELECT
			id::text,
			title,
			description,
			code,
			tags,
			url,
			similarity
		FROM search_examples($1, $2)
	`

	rows, err := c.db.Query(ctx, query, pgvector.NewVector(embedding), topK)
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

// HybridSearchDocs implements hybrid search (primary + contextual) for documentation
func (c *Client) HybridSearchDocs(ctx context.Context, userQuery, editorState string, topK int) ([]SearchResult, error) {
	// transform query to add technical keywords
	searchQuery, err := c.llm.TransformQuery(ctx, userQuery)
	if err != nil {
		log.Printf("query transformation failed, using original query: %v", err)
		searchQuery = userQuery
	}

	// extract editor context
	editorContext := extractEditorKeywords(editorState)

	// run primary and contextual searches in parallel
	primaryK := topK + 2 // get a few extra for merging

	var primaryResults, contextualResults []SearchResult
	var primaryErr, contextualErr error
	var wg sync.WaitGroup

	// primary search (60% weight) - user intent only
	wg.Add(1)

	go func() {
		defer wg.Done()
		primaryResults, primaryErr = c.VectorSearch(ctx, searchQuery, primaryK)
	}()

	// contextual search (40% weight) - if editor has content
	if editorContext != "" {
		wg.Add(1)

		go func() {
			defer wg.Done()
			contextualQuery := searchQuery + " " + editorContext
			contextualResults, contextualErr = c.VectorSearch(ctx, contextualQuery, topK)
		}()
	}

	wg.Wait()

	// check for errors
	if primaryErr != nil {
		return nil, fmt.Errorf("primary search failed: %w", primaryErr)
	}

	if contextualErr != nil {
		// don't fail completely, just log and use primary only
		log.Printf("contextual search failed, using primary only: %v", contextualErr)
		contextualResults = []SearchResult{}
	}

	// merge and rank by score
	merged := mergeAndRankDocs(primaryResults, contextualResults, topK)

	// fetch special chunks and organize by page
	organized, err := c.organizeByPage(ctx, merged)
	if err != nil {
		return nil, fmt.Errorf("failed to organize results: %w", err)
	}

	return organized, nil
}

// HybridSearchExamples implements hybrid search (primary + contextual) for examples
func (c *Client) HybridSearchExamples(ctx context.Context, userQuery, editorState string, topK int) ([]ExampleResult, error) {
	// transform query to add technical keywords
	searchQuery, err := c.llm.TransformQuery(ctx, userQuery)
	if err != nil {
		log.Printf("query transformation failed, using original query: %v", err)
		searchQuery = userQuery
	}

	// extract editor context
	editorContext := extractEditorKeywords(editorState)

	// run primary and contextual searches in parallel
	primaryK := topK + 2 // get a few extra for merging
	var primaryResults, contextualResults []ExampleResult
	var primaryErr, contextualErr error
	var wg sync.WaitGroup

	// primary search (60% weight) - user intent only
	wg.Add(1)

	go func() {
		defer wg.Done()
		primaryResults, primaryErr = c.SearchExamples(ctx, searchQuery, primaryK)
	}()

	// contextual search (40% weight) - if editor has content
	if editorContext != "" {
		wg.Add(1)

		go func() {
			defer wg.Done()
			contextualQuery := searchQuery + " " + editorContext
			contextualResults, contextualErr = c.SearchExamples(ctx, contextualQuery, topK)
		}()
	}

	wg.Wait()

	// check for errors
	if primaryErr != nil {
		return nil, fmt.Errorf("primary search failed: %w", primaryErr)
	}

	if contextualErr != nil {
		// don't fail completely, just log and use primary only
		log.Printf("contextual search failed, using primary only: %v", contextualErr)
		contextualResults = []ExampleResult{}
	}

	// merge and rank by score
	merged := mergeAndRankExamples(primaryResults, contextualResults, topK)

	return merged, nil
}
