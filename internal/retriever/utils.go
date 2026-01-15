package retriever

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"

	"codeberg.org/algorave/server/internal/logger"
	"codeberg.org/algorave/server/internal/strudel"
)

const (
	defaultTopK = 5
)

// groups chunks by page and fetches special sections
func (c *Client) organizeByPage(ctx context.Context, chunks []SearchResult) ([]SearchResult, error) {
	pageSet := make(map[string]bool)

	for _, chunk := range chunks {
		pageSet[chunk.PageName] = true
	}

	specialChunks := make(map[string][]SearchResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for pageName := range pageSet {
		wg.Add(1)
		go func(pName string) {
			defer wg.Done()

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

	pageOrder := []string{}
	pageSummaries := make(map[string]SearchResult)
	pageExamples := make(map[string]SearchResult)
	pageSections := make(map[string][]SearchResult)

	for _, chunk := range chunks {
		// track page order
		if !slices.Contains(pageOrder, chunk.PageName) {
			pageOrder = append(pageOrder, chunk.PageName)
		}

		// categorize chunks
		switch chunk.SectionTitle {
		case "PAGE_SUMMARY":
			pageSummaries[chunk.PageName] = chunk
		case "PAGE_EXAMPLES":
			pageExamples[chunk.PageName] = chunk
		default:
			pageSections[chunk.PageName] = append(pageSections[chunk.PageName], chunk)
		}
	}

	// add special chunks from database fetch
	for pageName, chunks := range specialChunks {
		for _, chunk := range chunks {
			switch chunk.SectionTitle {
			case "PAGE_SUMMARY":
				if _, exists := pageSummaries[pageName]; !exists {
					pageSummaries[pageName] = chunk
				}
			case "PAGE_EXAMPLES":
				if _, exists := pageExamples[pageName]; !exists {
					pageExamples[pageName] = chunk
				}
			}
		}
	}

	// build result: summary - examples - sections per page
	result := []SearchResult{}
	for _, pageName := range pageOrder {
		if summary, ok := pageSummaries[pageName]; ok {
			result = append(result, summary)
		}

		if examples, ok := pageExamples[pageName]; ok {
			result = append(result, examples)
		}

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
		if err.Error() == "no rows in result set" {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to fetch special chunk: %w", err)
	}

	return &result, nil
}

// parses editor state for contextual keywords
func extractEditorKeywords(editorState string) string {
	return strudel.ExtractKeywords(editorState)
}

// merges and deduplicates doc search results, ranking by similarity
func mergeAndRankDocs(primary, contextual []SearchResult, topK int) []SearchResult {
	seen := make(map[string]bool)
	merged := []SearchResult{}

	for _, chunk := range append(primary, contextual...) {
		if !seen[chunk.ID] {
			merged = append(merged, chunk)
			seen[chunk.ID] = true
		}
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Similarity > merged[j].Similarity
	})

	if len(merged) > topK {
		return merged[:topK]
	}

	return merged
}

// merges and deduplicates example search results, ranking by similarity
func mergeAndRankExamples(primary, contextual []ExampleResult, topK int) []ExampleResult {
	seen := make(map[string]bool)
	merged := []ExampleResult{}

	for _, example := range append(primary, contextual...) {
		if !seen[example.ID] {
			merged = append(merged, example)
			seen[example.ID] = true
		}
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Similarity > merged[j].Similarity
	})

	if len(merged) > topK {
		return merged[:topK]
	}
	return merged
}

// combines vector and BM25 search results with weighted scoring
func mergeVectorAndBM25Docs(vectorResults, bm25Results []SearchResult, topK int) []SearchResult {
	const (
		vectorWeight = 0.7
		bm25Weight   = 0.3
	)

	scoreMap := make(map[string]float32)
	chunkMap := make(map[string]SearchResult)

	for _, chunk := range vectorResults {
		scoreMap[chunk.ID] = chunk.Similarity * vectorWeight
		chunkMap[chunk.ID] = chunk
	}

	for _, chunk := range bm25Results {
		weightedScore := chunk.Similarity * bm25Weight
		if existingScore, exists := scoreMap[chunk.ID]; exists {
			scoreMap[chunk.ID] = existingScore + weightedScore
		} else {
			scoreMap[chunk.ID] = weightedScore
			chunkMap[chunk.ID] = chunk
		}
	}

	merged := make([]SearchResult, 0, len(chunkMap))
	for id, chunk := range chunkMap {
		chunk.Similarity = scoreMap[id]
		merged = append(merged, chunk)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Similarity > merged[j].Similarity
	})

	if len(merged) > topK {
		return merged[:topK]
	}

	return merged
}

// combines vector and BM25 search results with weighted scoring
func mergeVectorAndBM25Examples(vectorResults, bm25Results []ExampleResult, topK int) []ExampleResult {
	const (
		vectorWeight = 0.7
		bm25Weight   = 0.3
	)

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

	// sort by combined score
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Similarity > merged[j].Similarity
	})

	// return top K
	if len(merged) > topK {
		return merged[:topK]
	}

	return merged
}
