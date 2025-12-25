package retriever

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
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
				log.Printf("failed to fetch PAGE_SUMMARY for %s: %v", pName, err)
			} else if summary != nil {
				mu.Lock()
				specialChunks[pName] = append(specialChunks[pName], *summary)
				mu.Unlock()
			}

			// fetch PAGE_EXAMPLES (if < 500 chars)
			examples, err := c.fetchSpecialChunk(ctx, pName, "PAGE_EXAMPLES")
			if err != nil {
				log.Printf("failed to fetch PAGE_EXAMPLES for %s: %v", pName, err)
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
	query := `
		SELECT
			id::text,
			page_name,
			page_url,
			section_title,
			content,
			0.0 as similarity
		FROM doc_embeddings
		WHERE page_name = $1 AND section_title = $2
		LIMIT 1
	`

	var result SearchResult
	err := c.pool.QueryRow(ctx, query, pageName, sectionTitle).Scan(
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
func extractEditorKeywords(editorState string) string {
	if editorState == "" {
		return ""
	}

	keywords := []string{}

	// extract sound sample names: sound("bd") → "bd"
	soundRegex := regexp.MustCompile(`sound\("(\w+)"\)`)
	for _, match := range soundRegex.FindAllStringSubmatch(editorState, -1) {
		if len(match) > 1 {
			keywords = append(keywords, match[1])
		}
	}

	// extract note names: note("c e g") → "c e g"
	noteRegex := regexp.MustCompile(`note\("([^"]+)"\)`)
	for _, match := range noteRegex.FindAllStringSubmatch(editorState, -1) {
		if len(match) > 1 {
			// split notes and add individually
			notes := strings.Fields(match[1])
			keywords = append(keywords, notes...)
		}
	}

	// extract function calls: .fast(2) → "fast"
	funcRegex := regexp.MustCompile(`\.(\w+)\(`)
	for _, match := range funcRegex.FindAllStringSubmatch(editorState, -1) {
		if len(match) > 1 {
			keywords = append(keywords, match[1])
		}
	}

	// deduplicate and limit to ~10 keywords max to avoid noise
	uniqueKeywords := uniqueStrings(keywords)
	if len(uniqueKeywords) > 10 {
		uniqueKeywords = uniqueKeywords[:10]
	}

	return strings.Join(uniqueKeywords, " ")
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

// uniqueStrings returns a deduplicated slice of strings
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, s := range slice {
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}
	return result
}

// contains checks if a string slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// helper to split space-separated words
func splitWords(s string) []string {
	words := []string{}
	current := ""

	for _, char := range s {
		if char == ' ' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		words = append(words, current)
	}

	return words
}
