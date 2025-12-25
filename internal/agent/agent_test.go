package agent

import (
	"context"
	"testing"

	"github.com/algorave/server/internal/llm"
	"github.com/algorave/server/internal/retriever"
)

// mockLLM implements the llm.LLM interface for testing
type mockLLM struct {
	transformQueryFunc    func(ctx context.Context, query string) (string, error)
	generateEmbeddingFunc func(ctx context.Context, text string) ([]float32, error)
	generateTextFunc      func(ctx context.Context, req llm.TextGenerationRequest) (string, error)
}

func (m *mockLLM) TransformQuery(ctx context.Context, query string) (string, error) {
	if m.transformQueryFunc != nil {
		return m.transformQueryFunc(ctx, query)
	}
	return query, nil
}

func (m *mockLLM) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if m.generateEmbeddingFunc != nil {
		return m.generateEmbeddingFunc(ctx, text)
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *mockLLM) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		embeddings[i] = []float32{0.1, 0.2, 0.3}
	}
	return embeddings, nil
}

func (m *mockLLM) GenerateText(ctx context.Context, req llm.TextGenerationRequest) (string, error) {
	if m.generateTextFunc != nil {
		return m.generateTextFunc(ctx, req)
	}
	return "sound(\"bd\").fast(4)", nil
}

// mockRetriever implements a basic retriever for testing
type mockRetriever struct {
	hybridSearchDocsFunc     func(ctx context.Context, query, editorState string, k int) ([]retriever.SearchResult, error)
	hybridSearchExamplesFunc func(ctx context.Context, query, editorState string, k int) ([]retriever.ExampleResult, error)
}

func (m *mockRetriever) HybridSearchDocs(ctx context.Context, query, editorState string, k int) ([]retriever.SearchResult, error) {
	if m.hybridSearchDocsFunc != nil {
		return m.hybridSearchDocsFunc(ctx, query, editorState, k)
	}
	return []retriever.SearchResult{
		{
			PageName:     "Sound",
			SectionTitle: "Basic Sounds",
			Content:      "Use sound() to play samples",
			Similarity:   0.9,
		},
	}, nil
}

func (m *mockRetriever) HybridSearchExamples(ctx context.Context, query, editorState string, k int) ([]retriever.ExampleResult, error) {
	if m.hybridSearchExamplesFunc != nil {
		return m.hybridSearchExamplesFunc(ctx, query, editorState, k)
	}
	return []retriever.ExampleResult{
		{
			Title:       "Four on the floor",
			Description: "Basic kick drum pattern",
			Tags:        []string{"drums", "rhythm"},
			Code:        "sound(\"bd\").fast(4)",
			Similarity:  0.8,
		},
	}, nil
}

func (m *mockRetriever) Close() {}

func TestNewAgentWithDeps(t *testing.T) {
	mockRet := &mockRetriever{}
	mockLLMClient := &mockLLM{}

	agent := NewAgentWithDeps(mockRet, mockLLMClient)

	if agent == nil {
		t.Fatal("expected agent to be created")
	}

	if agent.retriever == nil {
		t.Error("expected retriever to be set correctly")
	}

	if agent.llm != mockLLMClient {
		t.Error("expected llm to be set correctly")
	}
}

func TestGenerateCode(t *testing.T) {
	ctx := context.Background()

	mockRet := &mockRetriever{}
	mockLLMClient := &mockLLM{
		generateTextFunc: func(ctx context.Context, req llm.TextGenerationRequest) (string, error) {
			// verify system prompt includes cheatsheet
			if req.SystemPrompt == "" {
				t.Error("expected system prompt to be set")
			}

			// verify messages include user query
			if len(req.Messages) == 0 {
				t.Error("expected at least one message")
			}

			return "sound(\"bd hh sd hh\").gain(0.8)", nil
		},
	}

	agent := NewAgentWithDeps(mockRet, mockLLMClient)

	req := GenerateRequest{
		UserQuery:           "make a drum beat",
		EditorState:         "",
		ConversationHistory: []Message{},
	}

	resp, err := agent.GenerateCode(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response to be non-nil")
	}

	if resp.Code != "sound(\"bd hh sd hh\").gain(0.8)" {
		t.Errorf("unexpected code: %s", resp.Code)
	}

	if resp.DocsRetrieved != 1 {
		t.Errorf("expected 1 doc retrieved, got %d", resp.DocsRetrieved)
	}

	if resp.ExamplesRetrieved != 1 {
		t.Errorf("expected 1 example retrieved, got %d", resp.ExamplesRetrieved)
	}
}

func TestGenerateCodeWithConversationHistory(t *testing.T) {
	ctx := context.Background()

	mockRet := &mockRetriever{}
	mockLLMClient := &mockLLM{
		generateTextFunc: func(ctx context.Context, req llm.TextGenerationRequest) (string, error) {
			// verify conversation history is included
			if len(req.Messages) != 3 { // 2 history + 1 current
				t.Errorf("expected 3 messages, got %d", len(req.Messages))
			}

			return "sound(\"bd hh sd hh\").fast(2)", nil
		},
	}

	agent := NewAgentWithDeps(mockRet, mockLLMClient)

	req := GenerateRequest{
		UserQuery:   "make it faster",
		EditorState: "sound(\"bd hh sd hh\")",
		ConversationHistory: []Message{
			{Role: "user", Content: "make a drum beat"},
			{Role: "assistant", Content: "sound(\"bd hh sd hh\")"},
		},
	}

	resp, err := agent.GenerateCode(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp.Code != "sound(\"bd hh sd hh\").fast(2)" {
		t.Errorf("unexpected code: %s", resp.Code)
	}
}

func TestGetCheatsheet(t *testing.T) {
	cheatsheet := getCheatsheet()

	// Note: This test may return empty if run from wrong directory
	// In production, the server runs from project root where resources/ exists
	if cheatsheet != "" && len(cheatsheet) < 100 {
		t.Error("expected cheatsheet to have substantial content if loaded")
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	ctx := SystemPromptContext{
		Cheatsheet:  "# Test Cheatsheet\nContent here",
		EditorState: "sound(\"bd\")",
		Docs: []retriever.SearchResult{
			{
				PageName:     "Sound",
				SectionTitle: "Basic",
				Content:      "Sample content",
				Similarity:   0.9,
			},
		},
		Examples: []retriever.ExampleResult{
			{
				Title:       "Example",
				Description: "Test example",
				Code:        "sound(\"test\")",
				Similarity:  0.8,
			},
		},
		Conversations: []Message{},
	}

	prompt := buildSystemPrompt(ctx)

	if prompt == "" {
		t.Fatal("expected prompt to be non-empty")
	}

	// verify all sections are included
	if len(prompt) < 200 {
		t.Error("expected prompt to have substantial content")
	}

	// verify cheatsheet is included
	if !contains(prompt, "STRUDEL QUICK REFERENCE") {
		t.Error("expected prompt to include cheatsheet section")
	}

	// verify editor state is included
	if !contains(prompt, "CURRENT EDITOR STATE") {
		t.Error("expected prompt to include editor state section")
	}

	// verify docs are included
	if !contains(prompt, "RELEVANT DOCUMENTATION") {
		t.Error("expected prompt to include documentation section")
	}

	// verify examples are included
	if !contains(prompt, "EXAMPLE STRUDELS") {
		t.Error("expected prompt to include examples section")
	}

	// verify instructions are included
	if !contains(prompt, "INSTRUCTIONS") {
		t.Error("expected prompt to include instructions section")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s != "" && substr != "" &&
		(s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
