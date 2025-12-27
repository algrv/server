package retriever

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"

	"github.com/algorave/server/internal/logger"
	"github.com/algorave/server/internal/strudel"
)

const (
	defaultTopK = 5
)

// organizeByPage groups chunks by page and fetches special sections
func (c *Client) organizeByPage(ctx context.Context, chunks []SearchResult) ([]SearchResult, error) {
	// identify unique pages
	pageSet := make(map[string]bool)

	for _, chunk := range chunks {
		pageSet[chunk.PageName] = true
	}

	// fetch special chunks for each page in parallel
	specialChunks := make(map[string][]SearchResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for pageName := range pageSet {
		wg.Add(1)
		go func(pName string) {
			defer wg.Done()

			// fetch PAGE_SUMMARY
			summary, err := c.fetchSpecialChunk(ctx, pName, "PAGE_SUMMARY")

			if err != nil {
				logger.Warn("failed to fetch PAGE_SUMMARY",
					"page_name", pName,
					"error", err,
				)
			} else if summary != nil {
				mu.Lock()
				specialChunks[pName] = append(specialChunks[pName], *summary)
				mu.Unlock()
			}

			// fetch PAGE_EXAMPLES (if < 500 chars)
			examples, err := c.fetchSpecialChunk(ctx, pName, "PAGE_EXAMPLES")

			if err != nil {
				logger.Warn("failed to fetch PAGE_EXAMPLES",
					"page_name", pName,
					"error", err,
				)
			} else if examples != nil && len(examples.Content) < 500 {
				mu.Lock()
				specialChunks[pName] = append(specialChunks[pName], *examples)
				mu.Unlock()
			}
		}(pageName)
	}

	wg.Wait()

	// organize by page: track page order, categorize chunks
	pageOrder := []string{}
	pageSummaries := make(map[string]SearchResult)
	pageExamples := make(map[string]SearchResult)
	pageSections := make(map[string][]SearchResult)

	for _, chunk := range chunks {
		// track page order
		if !contains(pageOrder, chunk.PageName) {
			pageOrder = append(pageOrder, chunk.PageName)
		}

		// categorize chunks
		if chunk.SectionTitle == "PAGE_SUMMARY" {
			pageSummaries[chunk.PageName] = chunk
		} else if chunk.SectionTitle == "PAGE_EXAMPLES" {
			pageExamples[chunk.PageName] = chunk
		} else {
			pageSections[chunk.PageName] = append(pageSections[chunk.PageName], chunk)
		}
	}

	// add special chunks from database fetch
	for pageName, chunks := range specialChunks {
		for _, chunk := range chunks {
			if chunk.SectionTitle == "PAGE_SUMMARY" {
				if _, exists := pageSummaries[pageName]; !exists {
					pageSummaries[pageName] = chunk
				}
			} else if chunk.SectionTitle == "PAGE_EXAMPLES" {
				if _, exists := pageExamples[pageName]; !exists {
					pageExamples[pageName] = chunk
				}
			}
		}
	}

	// build result: summary → examples → sections per page
	result := []SearchResult{}
	for _, pageName := range pageOrder {
		// add summary first (if exists)
		if summary, ok := pageSummaries[pageName]; ok {
			result = append(result, summary)
		}
		// add examples second (if exists)
		if examples, ok := pageExamples[pageName]; ok {
			result = append(result, examples)
		}
		// then add sections
		if sections, ok := pageSections[pageName]; ok {
			result = append(result, sections...)
		}
	}

	return result, nil
}

// fetchSpecialChunk fetches a special chunk (PAGE_SUMMARY or PAGE_EXAMPLES) by page name
func (c *Client) fetchSpecialChunk(ctx context.Context, pageName, sectionTitle string) (*SearchResult, error) {
	var result SearchResult
	err := c.db.QueryRow(ctx, fetchSpecialChunkQuery, pageName, sectionTitle).Scan(
		&result.ID,
		&result.PageName,
		&result.PageURL,
		&result.SectionTitle,
		&result.Content,
		&result.Similarity,
	)

	if err != nil {
		// no rows is not an error, just means the special chunk doesn't exist
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch special chunk: %w", err)
	}

	return &result, nil
}

// extractEditorKeywords parses editor state for contextual keywords
// Uses the shared strudel package for consistent parsing
func extractEditorKeywords(editorState string) string {
	return strudel.ExtractKeywords(editorState)
}

// mergeAndRankDocs merges and deduplicates doc search results, ranking by similarity
func mergeAndRankDocs(primary, contextual []SearchResult, topK int) []SearchResult {
	// deduplicate by chunk ID
	seen := make(map[string]bool)
	merged := []SearchResult{}

	for _, chunk := range append(primary, contextual...) {
		if !seen[chunk.ID] {
			merged = append(merged, chunk)
			seen[chunk.ID] = true
		}
	}

	// sort by similarity score (higher is better)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Similarity > merged[j].Similarity
	})

	// return top K
	if len(merged) > topK {
		return merged[:topK]
	}

	return merged
}

// mergeAndRankExamples merges and deduplicates example search results, ranking by similarity
func mergeAndRankExamples(primary, contextual []ExampleResult, topK int) []ExampleResult {
	// deduplicate by example ID
	seen := make(map[string]bool)
	merged := []ExampleResult{}

	for _, example := range append(primary, contextual...) {
		if !seen[example.ID] {
			merged = append(merged, example)
			seen[example.ID] = true
		}
	}

	// sort by similarity score (higher is better)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Similarity > merged[j].Similarity
	})

	// return top K
	if len(merged) > topK {
		return merged[:topK]
	}
	return merged
}

// mergeVectorAndBM25Docs combines vector and BM25 search results with weighted scoring
// Vector results get 70% weight, BM25 results get 30% weight (following Cursor's approach)
func mergeVectorAndBM25Docs(vectorResults, bm25Results []SearchResult, topK int) []SearchResult {
	const (
		vectorWeight = 0.7
		bm25Weight   = 0.3
	)

	// create a map to track combined scores by chunk ID
	scoreMap := make(map[string]float32)
	chunkMap := make(map[string]SearchResult)

	// add vector results with 70% weight
	for _, chunk := range vectorResults {
		scoreMap[chunk.ID] = chunk.Similarity * vectorWeight
		chunkMap[chunk.ID] = chunk
	}

	// add BM25 results with 30% weight, combining scores if already present
	for _, chunk := range bm25Results {
		weightedScore := chunk.Similarity * bm25Weight
		if existingScore, exists := scoreMap[chunk.ID]; exists {
			scoreMap[chunk.ID] = existingScore + weightedScore
		} else {
			scoreMap[chunk.ID] = weightedScore
			chunkMap[chunk.ID] = chunk
		}
	}

	// build result list with combined scores
	merged := make([]SearchResult, 0, len(chunkMap))
	for id, chunk := range chunkMap {
		chunk.Similarity = scoreMap[id]
		merged = append(merged, chunk)
	}

	// sort by combined score (higher is better)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Similarity > merged[j].Similarity
	})

	// return top K
	if len(merged) > topK {
		return merged[:topK]
	}

	return merged
}

// mergeVectorAndBM25Examples combines vector and BM25 search results with weighted scoring
// Vector results get 70% weight, BM25 results get 30% weight (following Cursor's approach)
func mergeVectorAndBM25Examples(vectorResults, bm25Results []ExampleResult, topK int) []ExampleResult {
	const (
		vectorWeight = 0.7
		bm25Weight   = 0.3
	)

	// create a map to track combined scores by example ID
	scoreMap := make(map[string]float32)
	exampleMap := make(map[string]ExampleResult)

	// add vector results with 70% weight
	for _, example := range vectorResults {
		scoreMap[example.ID] = example.Similarity * vectorWeight
		exampleMap[example.ID] = example
	}

	// add BM25 results with 30% weight, combining scores if already present
	for _, example := range bm25Results {
		weightedScore := example.Similarity * bm25Weight
		if existingScore, exists := scoreMap[example.ID]; exists {
			scoreMap[example.ID] = existingScore + weightedScore
		} else {
			scoreMap[example.ID] = weightedScore
			exampleMap[example.ID] = example
		}
	}

	// build result list with combined scores
	merged := make([]ExampleResult, 0, len(exampleMap))
	for id, example := range exampleMap {
		example.Similarity = scoreMap[id]
		merged = append(merged, example)
	}

	// sort by combined score (higher is better)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Similarity > merged[j].Similarity
	})

	// return top K
	if len(merged) > topK {
		return merged[:topK]
	}

	return merged
}

// contains checks if a string slice contains a string
func contains(slice []string, str string) bool {
	return slices.Contains(slice, str)
}
