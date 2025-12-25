package chunker

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	frontmatterRegex  = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n`)
	headerRegex       = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	mdxComponentRegex = regexp.MustCompile(`<[A-Z]\w+[^>]*>.*?</[A-Z]\w+>|<[A-Z]\w+[^/>]*/>`)
	importRegex       = regexp.MustCompile(`(?m)^import\s+.*$`)
)

func splitByHeaders(content string) []Section {
	lines := strings.Split(content, "\n")

	var sections []Section
	var currentSection *Section

	for _, line := range lines {
		matches := headerRegex.FindStringSubmatch(line)

		if len(matches) > 0 {
			if currentSection != nil && strings.TrimSpace(currentSection.Content) != "" {
				sections = append(sections, *currentSection)
			}

			level := len(matches[1])
			title := strings.TrimSpace(matches[2])
			currentSection = &Section{
				Title:   title,
				Level:   level,
				Content: line + "\n",
			}
		} else if currentSection != nil {
			currentSection.Content += line + "\n"
		} else {
			// content before any header - create an untitled section
			currentSection = &Section{
				Title:   "",
				Level:   0,
				Content: line + "\n",
			}
		}
	}

	if currentSection != nil && strings.TrimSpace(currentSection.Content) != "" {
		sections = append(sections, *currentSection)
	}

	return sections
}

func splitLargeSection(section Section, opts ChunkOptions) []string {
	var chunks []string
	paragraphs := strings.Split(section.Content, "\n\n")

	var currentChunk strings.Builder
	headerWritten := false

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)

		if para == "" {
			continue
		}

		testContent := currentChunk.String() + "\n\n" + para

		if estimateTokens(testContent) > opts.MaxTokens && currentChunk.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
			headerWritten = false
		}

		if !headerWritten && opts.PreserveHeaders && section.Title != "" {
			headerPrefix := strings.Repeat("#", section.Level)
			currentChunk.WriteString(fmt.Sprintf("%s %s\n\n", headerPrefix, section.Title))
			headerWritten = true
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}

		currentChunk.WriteString(para)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	return chunks
}

func estimateTokens(text string) int {
	return len(text) / 4
}

func extractFrontmatter(content string) map[string]interface{} {
	metadata := make(map[string]interface{})

	matches := frontmatterRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return metadata
	}

	frontmatter := matches[1]
	lines := strings.Split(frontmatter, "\n")

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			metadata[key] = value
		}
	}

	return metadata
}

func stripMDXComponents(content string) string {
	return mdxComponentRegex.ReplaceAllString(content, "")
}

func generateURL(pageName string) string {
	name := strings.TrimSuffix(pageName, ".mdx")

	url := strings.ToLower(name)
	url = strings.ReplaceAll(url, " ", "-")

	return fmt.Sprintf("/learn/%s", url)
}

// isSummarySection checks if a section title indicates it's a summary or overview section
func isSummarySection(title string) bool {
	normalized := strings.ToLower(strings.TrimSpace(title))
	return normalized == "summary" || normalized == "overview" || normalized == "recap"
}

// isExamplesSection checks if a section title indicates it's an examples section
func isExamplesSection(title string) bool {
	normalized := strings.ToLower(strings.TrimSpace(title))
	return normalized == "examples" || normalized == "example"
}

// extractSectionText extracts the text content from a section
// it removes the header line and returns just the content
// preserves newlines for code blocks
func extractSectionText(content string) string {
	lines := strings.Split(content, "\n")
	var contentLines []string

	// skip the first line if it's a header (starts with #)
	startIdx := 0
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "#") {
		startIdx = 1
	}

	// collect remaining lines, preserving newlines for code blocks
	for i := startIdx; i < len(lines); i++ {
		contentLines = append(contentLines, lines[i])
	}

	return strings.Join(contentLines, "\n")
}
