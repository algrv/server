package chunker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChunkDocument(t *testing.T) {
	testFile := filepath.Join("..", "..", "docs", "strudel", "learn", "notes.mdx")
	content, err := os.ReadFile(testFile)

	if err != nil {
		t.Skipf("Test file not found (skip if docs not present): %v", err)
	}

	opts := DefaultOptions()

	chunks, err := ChunkDocument(string(content), "learn", "notes.mdx", opts)
	if err != nil {
		t.Fatalf("ChunkDocument failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk, got 0")
	}

	t.Logf("Generated %d chunks from notes.mdx\n", len(chunks))
	t.Logf("========================================\n")

	for i, chunk := range chunks {
		t.Logf("\n--- Chunk %d ---", i+1)
		t.Logf("Page: %s", chunk.PageName)
		t.Logf("URL: %s", chunk.PageURL)
		t.Logf("Section: %s", chunk.SectionTitle)
		t.Logf("Content length: %d chars (~%d tokens)", len(chunk.Content), estimateTokens(chunk.Content))
		t.Logf("Metadata: %+v", chunk.Metadata)
		t.Logf("\nContent preview (first 200 chars):")
		preview := chunk.Content

		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}

		t.Logf("%s\n", preview)
	}
}

func TestSpecialSectionExtraction(t *testing.T) {
	// test document with both summary and examples sections
	testContent := `---
	title: Test Document
	---

	# Test Page

	This is the intro content.

	## Summary

	This page covers the basics of testing special section extraction.
	It ensures that summary and examples sections are properly extracted.

	## Examples

	` + "```" + `javascript
	sound('bd')
	note('c e g')
	` + "```" + `

	These are basic examples showing how to use sound and note functions.

	## Regular Section

	This is a regular section that should be chunked normally.

	## Another Section

	Another regular section with content.
	`

	opts := DefaultOptions()

	chunks, err := ChunkDocument(testContent, "/docs/strudel", "test-page.mdx", opts)
	if err != nil {
		t.Fatalf("ChunkDocument failed: %v", err)
	}

	// verify we have the expected chunks:
	// 1. PAGE_SUMMARY
	// 2. PAGE_EXAMPLES
	// 3. intro content (Test Page section)
	// 4. Regular Section
	// 5. Another Section
	expectedMinChunks := 5

	if len(chunks) < expectedMinChunks {
		t.Errorf("Expected at least %d chunks, got %d", expectedMinChunks, len(chunks))
	}

	// verify PAGE_SUMMARY chunk exists and has correct content
	foundSummary := false

	for _, chunk := range chunks {
		if chunk.SectionTitle == "PAGE_SUMMARY" {
			foundSummary = true

			if !contains(chunk.Content, "SUMMARY:") {
				t.Errorf("PAGE_SUMMARY chunk should start with 'SUMMARY:', got: %s", chunk.Content[:50])
			}

			if !contains(chunk.Content, "testing special section extraction") {
				t.Errorf("PAGE_SUMMARY content incorrect, got: %s", chunk.Content)
			}

			t.Logf("✓ PAGE_SUMMARY chunk found: %s", chunk.Content)
		}
	}

	if !foundSummary {
		t.Error("PAGE_SUMMARY chunk not found")
	}

	// verify PAGE_EXAMPLES chunk exists and has correct content
	foundExamples := false

	for _, chunk := range chunks {
		if chunk.SectionTitle == "PAGE_EXAMPLES" {
			foundExamples = true

			if !contains(chunk.Content, "sound('bd')") {
				t.Errorf("PAGE_EXAMPLES should contain code examples, got: %s", chunk.Content)
			}
			// verify code block is preserved

			if !contains(chunk.Content, "```") {
				t.Errorf("PAGE_EXAMPLES should preserve code blocks with backticks")
			}

			t.Logf("✓ PAGE_EXAMPLES chunk found with code blocks preserved")
		}
	}

	if !foundExamples {
		t.Error("PAGE_EXAMPLES chunk not found")
	}

	// verify regular sections are still chunked
	foundRegular := false

	for _, chunk := range chunks {
		if chunk.SectionTitle == "Regular Section" {
			foundRegular = true
			t.Logf("✓ Regular section chunked correctly")
		}
	}

	if !foundRegular {
		t.Error("Regular sections should still be chunked normally")
	}

	// log all chunks for inspection
	t.Logf("\nAll chunks:")

	for i, chunk := range chunks {
		t.Logf("%d. SectionTitle: %s, ContentLength: %d", i+1, chunk.SectionTitle, len(chunk.Content))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

func TestRealDocumentWithExamples(t *testing.T) {
	testFile := filepath.Join("..", "..", "docs", "strudel", "workshop", "first-sounds.mdx")

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Skipf("Test file not found (skip if docs not present): %v", err)
	}

	opts := DefaultOptions()

	chunks, err := ChunkDocument(string(content), "/docs/strudel", "first-sounds.mdx", opts)
	if err != nil {
		t.Fatalf("ChunkDocument failed: %v", err)
	}

	t.Logf("Generated %d chunks from first-sounds.mdx\n", len(chunks))

	// verify PAGE_EXAMPLES chunk was created
	foundExamples := false

	for i, chunk := range chunks {
		if chunk.SectionTitle == "PAGE_EXAMPLES" {
			foundExamples = true

			t.Logf("\n✓ PAGE_EXAMPLES chunk found at position %d", i+1)
			t.Logf("  Content length: %d chars (~%d tokens)", len(chunk.Content), estimateTokens(chunk.Content))

			// show preview
			preview := chunk.Content

			if len(preview) > 300 {
				preview = preview[:300] + "..."
			}

			t.Logf("  Preview: %s", preview)
		}
	}

	if !foundExamples {
		t.Log("⚠ No PAGE_EXAMPLES chunk found - this is OK if the document has no Examples section")
	}

	// log summary of all chunks
	t.Logf("\nChunk summary:")

	for i, chunk := range chunks {
		t.Logf("  %d. Section: %s (%d chars)", i+1, chunk.SectionTitle, len(chunk.Content))
	}
}
