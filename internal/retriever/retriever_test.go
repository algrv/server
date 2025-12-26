package retriever

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/algorave/server/internal/strudel"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type mockTransformer struct{}

type mockEmbedder struct {
	embedding []float32
}

func init() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}
}

func (m *mockEmbedder) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if m.embedding != nil {
		return m.embedding, nil
	}

	return make([]float32, 1536), nil
}

func (m *mockEmbedder) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i := range texts {
		if m.embedding != nil {
			embeddings[i] = m.embedding
		} else {
			embeddings[i] = make([]float32, 1536)
		}
	}

	return embeddings, nil
}

func (m *mockTransformer) TransformQuery(ctx context.Context, query string) (string, error) {
	return query + " expanded keywords", nil
}

// helper to create a test client with real DB connection
func createTestClient(t *testing.T) *Client {
	t.Helper()

	ctx := context.Background()
	connString := os.Getenv("SUPABASE_CONNECTION_STRING")

	if connString == "" {
		t.Skip("SUPABASE_CONNECTION_STRING not set, skipping integration test")
	}

	db, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	return New(db, &mockEmbedder{}, &mockTransformer{})
}

// verifies the merge logic works correctly
func TestMergeAndRankDocs(t *testing.T) {
	primary := []SearchResult{
		{ID: "1", Similarity: 0.95},
		{ID: "2", Similarity: 0.90},
		{ID: "3", Similarity: 0.85},
	}

	contextual := []SearchResult{
		{ID: "2", Similarity: 0.88}, // duplicate
		{ID: "4", Similarity: 0.80},
		{ID: "5", Similarity: 0.75},
	}

	merged := mergeAndRankDocs(primary, contextual, 5)

	// verify deduplication
	if len(merged) != 5 {
		t.Errorf("Expected 5 unique results, got %d", len(merged))
	}

	// verify ordering by similarity (descending)
	for i := 0; i < len(merged)-1; i++ {
		if merged[i].Similarity < merged[i+1].Similarity {
			t.Errorf("Results not sorted correctly: %f < %f at position %d",
				merged[i].Similarity, merged[i+1].Similarity, i)
		}
	}

	// verify no duplicate IDs
	seen := make(map[string]bool)

	for _, result := range merged {
		if seen[result.ID] {
			t.Errorf("Duplicate ID found: %s", result.ID)
		}
		seen[result.ID] = true
	}

	// verify top K limit
	topK := 3
	limited := mergeAndRankDocs(primary, contextual, topK)

	if len(limited) != topK {
		t.Errorf("Expected %d results after topK limit, got %d", topK, len(limited))
	}
}

// verifies the merge logic works correctly for examples
func TestMergeAndRankExamples(t *testing.T) {
	primary := []ExampleResult{
		{ID: "1", Similarity: 0.95},
		{ID: "2", Similarity: 0.90},
		{ID: "3", Similarity: 0.85},
	}

	contextual := []ExampleResult{
		{ID: "2", Similarity: 0.88}, // duplicate
		{ID: "4", Similarity: 0.80},
		{ID: "5", Similarity: 0.75},
	}

	merged := mergeAndRankExamples(primary, contextual, 5)

	// verify deduplication
	if len(merged) != 5 {
		t.Errorf("Expected 5 unique results, got %d", len(merged))
	}

	// verify ordering by similarity (descending)
	for i := 0; i < len(merged)-1; i++ {
		if merged[i].Similarity < merged[i+1].Similarity {
			t.Errorf("Results not sorted correctly: %f < %f at position %d",
				merged[i].Similarity, merged[i+1].Similarity, i)
		}
	}

	// verify no duplicate IDs
	seen := make(map[string]bool)

	for _, result := range merged {
		if seen[result.ID] {
			t.Errorf("Duplicate ID found: %s", result.ID)
		}

		seen[result.ID] = true
	}
}

// verifies keyword extraction from editor state
func TestExtractEditorKeywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty editor",
			input:    "",
			expected: []string{},
		},
		{
			name:     "simple sound",
			input:    `sound("bd")`,
			expected: []string{"bd"},
		},
		{
			name:     "sound with functions",
			input:    `sound("bd").fast(4).gain(0.8)`,
			expected: []string{"bd", "fast", "gain"},
		},
		{
			name:     "notes",
			input:    `note("c e g")`,
			expected: []string{"c", "e", "g"},
		},
		{
			name:     "complex pattern",
			input:    `sound("bd").fast(2).stack(sound("sd").slow(4))`,
			expected: []string{"bd", "fast", "stack", "sd", "slow"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEditorKeywords(tt.input)

			if tt.input == "" {
				if result != "" {
					t.Errorf("Expected empty string for empty input, got %q", result)
				}

				return
			}

			// check that expected keywords are present
			for _, keyword := range tt.expected {
				if !contains([]string{result}, keyword) {
					// split result to check individual keywords
					resultWords := make(map[string]bool)

					for _, word := range strings.Split(result, " ") {
						resultWords[word] = true
					}

					if !resultWords[keyword] {
						t.Errorf("Expected keyword %q not found in result %q", keyword, result)
					}
				}
			}
		})
	}
}

// verifies deduplication utility
func TestUniqueStrings(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b", "d"}
	result := strudel.UniqueStrings(input)

	if len(result) != 4 {
		t.Errorf("Expected 4 unique strings, got %d", len(result))
	}

	// verify all unique
	seen := make(map[string]bool)
	for _, s := range result {
		if seen[s] {
			t.Errorf("Duplicate string found: %s", s)
		}
		seen[s] = true
	}

	// verify all original strings present
	for _, expected := range []string{"a", "b", "c", "d"} {
		if !seen[expected] {
			t.Errorf("Expected string %q not found in result", expected)
		}
	}
}

// verifies the contains helper
func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	if !contains(slice, "banana") {
		t.Error("Expected contains to return true for 'banana'")
	}

	if contains(slice, "grape") {
		t.Error("Expected contains to return false for 'grape'")
	}

	if contains([]string{}, "anything") {
		t.Error("Expected contains to return false for empty slice")
	}
}
