package chunker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/algorave/server/internal/logger"
)

func DefaultOptions() ChunkOptions {
	return ChunkOptions{
		MaxTokens:       800,
		OverlapTokens:   100,
		PreserveHeaders: true,
	}
}

func ChunkDocument(content, sourcePathname string, pageName string, opts ChunkOptions) ([]Chunk, error) {
	metadata := extractFrontmatter(content)
	content = frontmatterRegex.ReplaceAllString(content, "")
	content = importRegex.ReplaceAllString(content, "")
	content = stripMDXComponents(content)

	pageURL := generateURL(sourcePathname, pageName)
	sections := splitByHeaders(content)

	var chunks []Chunk
	var summaryContent string
	var examplesContent string

	// extract special sections if they exist
	// track indices to remove them from regular processing
	var indicesToRemove []int

	for i, section := range sections {
		if isSummarySection(section.Title) {
			summaryContent = extractSectionText(section.Content)
			indicesToRemove = append(indicesToRemove, i)
		} else if isExamplesSection(section.Title) {
			examplesContent = extractSectionText(section.Content)
			indicesToRemove = append(indicesToRemove, i)
		}
	}

	// remove special sections from regular processing (in reverse order to preserve indices)
	for i := len(indicesToRemove) - 1; i >= 0; i-- {
		idx := indicesToRemove[i]
		sections = append(sections[:idx], sections[idx+1:]...)
	}

	// create PAGE_SUMMARY chunk if summary was found
	if summaryContent != "" {
		chunks = append(chunks, Chunk{
			PageName:     pageName,
			PageURL:      pageURL,
			SectionTitle: "PAGE_SUMMARY",
			Content:      "SUMMARY: " + strings.TrimSpace(summaryContent),
			Metadata:     metadata,
		})
	}

	// create PAGE_EXAMPLES chunk if examples were found
	if examplesContent != "" {
		chunks = append(chunks, Chunk{
			PageName:     pageName,
			PageURL:      pageURL,
			SectionTitle: "PAGE_EXAMPLES",
			Content:      strings.TrimSpace(examplesContent),
			Metadata:     metadata,
		})
	}

	// create chunks for regular sections
	for _, section := range sections {
		if estimateTokens(section.Content) <= opts.MaxTokens {
			chunks = append(chunks, Chunk{
				PageName:     pageName,
				PageURL:      pageURL,
				SectionTitle: section.Title,
				Content:      strings.TrimSpace(section.Content),
				Metadata:     metadata,
			})

			continue
		}

		subChunks := splitLargeSection(section, opts)

		for _, subChunk := range subChunks {
			chunks = append(chunks, Chunk{
				PageName:     pageName,
				PageURL:      pageURL,
				SectionTitle: section.Title,
				Content:      strings.TrimSpace(subChunk),
				Metadata:     metadata,
			})
		}
	}

	return chunks, nil
}

// discovers all markdown files in a directory and chunks them
// returns chunks and a slice of errors encountered (one per failed file)
func ChunkDocuments(docsPath string) ([]Chunk, []error) {
	opts := DefaultOptions()
	var allChunks []Chunk
	var errors []error
	fileCount := 0

	// walk the directory tree to find all markdown files
	walkErr := filepath.Walk(docsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Warn("error accessing path",
				"path", path,
				"error", err,
			)
			errors = append(errors, fmt.Errorf("path %s: %w", path, err))
			return nil // continue walking
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".mdx" {
			return nil
		}

		fileCount++

		content, err := os.ReadFile(path)
		if err != nil {
			logger.Warn("failed to read file",
				"path", path,
				"error", err,
			)
			errors = append(errors, fmt.Errorf("read %s: %w", path, err))
			return nil // continue with other files
		}

		pageName, err := filepath.Rel(docsPath, path)
		if err != nil {
			pageName = filepath.Base(path)
		}

		chunks, err := ChunkDocument(string(content), strings.TrimPrefix(docsPath, "./"), pageName, opts)
		if err != nil {
			logger.Warn("failed to chunk document",
				"path", path,
				"error", err,
			)
			errors = append(errors, fmt.Errorf("chunk %s: %w", path, err))
			return nil // continue with other files
		}

		allChunks = append(allChunks, chunks...)

		return nil
	})

	if walkErr != nil {
		errors = append(errors, fmt.Errorf("walk error: %w", walkErr))
	}

	logger.Info("processed markdown files",
		"file_count", fileCount,
		"chunks_generated", len(allChunks),
		"errors", len(errors),
	)

	return allChunks, errors
}
