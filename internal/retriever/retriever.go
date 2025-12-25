package retriever

import (
	"context"
	"fmt"
	"log"

	"github.com/algorave/server/internal/llm"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// NewClient creates a new retriever client with auto-configuration from environment
func NewClient(ctx context.Context) (*Client, error) {
	config, err := loadRetrieverConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load retriever config: %w", err)
	}

	return NewClientWithConfig(ctx, config)
}

// NewClientWithConfig creates a new retriever client with explicit configuration
func NewClientWithConfig(ctx context.Context, config *RetrieverConfig) (*Client, error) {
	// initialize database connection pool
	pool, err := pgxpool.New(ctx, config.DBConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// initialize LLM client (loads its own config from env)
	llmClient, err := llm.NewLLM(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	return &Client{
		pool: pool,
		llm:  llmClient,
		topK: config.TopK,
	}, nil
}

// Close closes the retriever client
func (c *Client) Close() {
	c.pool.Close()
}

// VectorSearch performs a vector similarity search on doc_embeddings
func (c *Client) VectorSearch(ctx context.Context, queryText string, topK int) ([]SearchResult, error) {
	// generate embedding for query
	embedding, err := c.llm.GenerateEmbedding(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// use the search_docs function from Supabase
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

	rows, err := c.pool.Query(ctx, query, pgvector.NewVector(embedding), topK)
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
	// generate embedding for query
	embedding, err := c.llm.GenerateEmbedding(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// use the search_examples function from Supabase
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

	rows, err := c.pool.Query(ctx, query, pgvector.NewVector(embedding), topK)
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
		// fallback to original query if transformation fails
		log.Printf("query transformation failed, using original query: %v", err)
		searchQuery = userQuery
	}

	// extract editor context
	editorContext := extractEditorKeywords(editorState)

	// primary search (60% weight) - user intent only
	primaryK := topK + 2 // get a few extra for merging
	primaryResults, err := c.VectorSearch(ctx, searchQuery, primaryK)
	if err != nil {
		return nil, fmt.Errorf("primary search failed: %w", err)
	}

	// contextual search (40% weight) - if editor has content
	var contextualResults []SearchResult
	if editorContext != "" {
		contextualQuery := searchQuery + " " + editorContext
		contextualResults, err = c.VectorSearch(ctx, contextualQuery, topK)
		if err != nil {
			// don't fail completely, just log and use primary only
			log.Printf("contextual search failed, using primary only: %v", err)
			contextualResults = []SearchResult{}
		}
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
		// fallback to original query if transformation fails
		log.Printf("query transformation failed, using original query: %v", err)
		searchQuery = userQuery
	}

	// extract editor context
	editorContext := extractEditorKeywords(editorState)

	// primary search (60% weight) - user intent only
	primaryK := topK + 2 // get a few extra for merging
	primaryResults, err := c.SearchExamples(ctx, searchQuery, primaryK)
	if err != nil {
		return nil, fmt.Errorf("primary search failed: %w", err)
	}

	// contextual search (40% weight) - if editor has content
	var contextualResults []ExampleResult
	if editorContext != "" {
		contextualQuery := searchQuery + " " + editorContext
		contextualResults, err = c.SearchExamples(ctx, contextualQuery, topK)
		if err != nil {
			// don't fail completely, just log and use primary only
			log.Printf("contextual search failed, using primary only: %v", err)
			contextualResults = []ExampleResult{}
		}
	}

	// merge and rank by score
	merged := mergeAndRankExamples(primaryResults, contextualResults, topK)

	return merged, nil
}
