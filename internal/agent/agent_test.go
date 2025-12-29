package agent

import (
	"context"
	"testing"

	"github.com/algorave/server/internal/llm"
	"github.com/algorave/server/internal/retriever"
)

// implements llm.LLM for testing
type mockLLM struct {
	generateTextFunc func(ctx context.Context, req llm.TextGenerationRequest) (string, error)
	model            string
}

func (m *mockLLM) GenerateText(ctx context.Context, req llm.TextGenerationRequest) (string, error) {
	if m.generateTextFunc != nil {
		return m.generateTextFunc(ctx, req)
	}

	return "sound(\"bd\").fast(4)", nil
}

func (m *mockLLM) Model() string {
	if m.model != "" {
		return m.model
	}

	return "mock-model"
}

func (m *mockLLM) GenerateEmbedding(_ context.Context, _ string) ([]float32, error) {
	return make([]float32, 1536), nil
}

func (m *mockLLM) GenerateEmbeddings(_ context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		embeddings[i] = make([]float32, 1536)
	}
	return embeddings, nil
}

func (m *mockLLM) TransformQuery(_ context.Context, query string) (string, error) {
	return query + " expanded", nil
}

func (m *mockLLM) AnalyzeQuery(_ context.Context, query string) (*llm.QueryAnalysis, error) {
	return &llm.QueryAnalysis{
		TransformedQuery:    query + " expanded",
		IsActionable:        true,
		ConcreteRequests:    []string{query},
		ClarifyingQuestions: []string{},
	}, nil
}

// implements Retriever for testing
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

func TestNew(t *testing.T) {
	mockRet := &mockRetriever{}
	mockGen := &mockLLM{}

	agent := New(mockRet, mockGen)

	if agent == nil {
		t.Fatal("expected agent to be created")
	}

	if agent.retriever == nil {
		t.Error("expected retriever to be set correctly")
	}

	if agent.generator == nil {
		t.Error("expected generator to be set correctly")
	}
}

func TestGenerateCode(t *testing.T) {
	ctx := context.Background()

	mockRet := &mockRetriever{}
	mockGen := &mockLLM{
		generateTextFunc: func(_ context.Context, req llm.TextGenerationRequest) (string, error) {
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

	agent := New(mockRet, mockGen)

	req := GenerateRequest{
		UserQuery:           "make a drum beat",
		EditorState:         "",
		ConversationHistory: []Message{},
	}

	resp, err := agent.Generate(ctx, req)
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
	mockGen := &mockLLM{
		generateTextFunc: func(_ context.Context, req llm.TextGenerationRequest) (string, error) {
			// verify conversation history is included
			if len(req.Messages) != 3 { // 2 history + 1 current
				t.Errorf("expected 3 messages, got %d", len(req.Messages))
			}

			return "sound(\"bd hh sd hh\").fast(2)", nil
		},
	}

	agent := New(mockRet, mockGen)

	req := GenerateRequest{
		UserQuery:   "make it faster",
		EditorState: "sound(\"bd hh sd hh\")",
		ConversationHistory: []Message{
			{Role: "user", Content: "make a drum beat"},
			{Role: "assistant", Content: "sound(\"bd hh sd hh\")"},
		},
	}

	resp, err := agent.Generate(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp.Code != "sound(\"bd hh sd hh\").fast(2)" {
		t.Errorf("unexpected code: %s", resp.Code)
	}
}

func TestGetCheatsheet(t *testing.T) {
	cheatsheet := getCheatsheet()

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
	if !containsSubstr(prompt, "STRUDEL QUICK REFERENCE") {
		t.Error("expected prompt to include cheatsheet section")
	}

	// verify editor state is included
	if !containsSubstr(prompt, "CURRENT EDITOR STATE") {
		t.Error("expected prompt to include editor state section")
	}

	// verify docs are included
	if !containsSubstr(prompt, "RELEVANT DOCUMENTATION") {
		t.Error("expected prompt to include documentation section")
	}

	// verify examples are included
	if !containsSubstr(prompt, "EXAMPLE STRUDELS") {
		t.Error("expected prompt to include examples section")
	}

	// verify instructions are included
	if !containsSubstr(prompt, "INSTRUCTIONS") {
		t.Error("expected prompt to include instructions section")
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

func TestEnhancedInstructionsPresent(t *testing.T) {
	instructions := getInstructions()

	// verify enhanced instructions sections are present
	requiredSections := []string{
		"REQUEST TYPE ANALYSIS",
		"A. ADDITIVE REQUESTS",
		"B. MODIFICATION REQUESTS",
		"C. DELETION REQUESTS",
		"D. QUESTIONS",
		"SURGICAL PRECISION",
		"Step 1: IDENTIFY",
		"Step 2: LOCATE",
		"Step 3: MAKE THE CHANGE SURGICALLY",
		"Step 4: PRESERVE EVERYTHING ELSE",
		"Example 1: MODIFICATION",
		"Example 2: DELETION",
		"Example 3: ADDITIVE",
		"Example 4: MODIFICATION",
		"Example 5: MODIFICATION",
	}

	for _, section := range requiredSections {
		if !containsSubstr(instructions, section) {
			t.Errorf("missing required enhanced instruction section: %q", section)
		}
	}

	// verify critical keywords are present (case-sensitive)
	criticalKeywords := []string{
		"SURGICAL",
		"PRESERVE",
		"EXACTLY",
	}

	for _, keyword := range criticalKeywords {
		if !containsSubstr(instructions, keyword) {
			t.Errorf("missing critical keyword in instructions: %q", keyword)
		}
	}

	t.Logf("✓ All enhanced instruction sections and keywords present")
}
