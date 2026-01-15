package agent

import (
	"context"
	"testing"

	"codeberg.org/algorave/server/internal/llm"
	"codeberg.org/algorave/server/internal/retriever"
)

// implements llm.LLM for testing
type mockLLM struct {
	generateTextFunc func(ctx context.Context, req llm.TextGenerationRequest) (*llm.TextGenerationResponse, error)
	model            string
}

func (m *mockLLM) GenerateText(ctx context.Context, req llm.TextGenerationRequest) (*llm.TextGenerationResponse, error) {
	if m.generateTextFunc != nil {
		return m.generateTextFunc(ctx, req)
	}

	return &llm.TextGenerationResponse{
		Text: "sound(\"bd\").fast(4)",
		Usage: llm.Usage{
			InputTokens:  100,
			OutputTokens: 20,
		},
	}, nil
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
		IsCodeRequest:       true,
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
		generateTextFunc: func(_ context.Context, req llm.TextGenerationRequest) (*llm.TextGenerationResponse, error) {
			// verify system prompt includes cheatsheet
			if req.SystemPrompt == "" {
				t.Error("expected system prompt to be set")
			}

			// verify messages include user query
			if len(req.Messages) == 0 {
				t.Error("expected at least one message")
			}

			return &llm.TextGenerationResponse{
				Text: "sound(\"bd hh sd hh\").gain(0.8)",
				Usage: llm.Usage{
					InputTokens:  150,
					OutputTokens: 25,
				},
			}, nil
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
		generateTextFunc: func(_ context.Context, req llm.TextGenerationRequest) (*llm.TextGenerationResponse, error) {
			// verify conversation history is included
			if len(req.Messages) != 3 { // 2 history + 1 current
				t.Errorf("expected 3 messages, got %d", len(req.Messages))
			}

			return &llm.TextGenerationResponse{
				Text: "sound(\"bd hh sd hh\").fast(2)",
				Usage: llm.Usage{
					InputTokens:  200,
					OutputTokens: 30,
				},
			}, nil
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

func TestAnalyzeResponse(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		wantContent string
		wantIsCode  bool
	}{
		// raw code patterns (no fences)
		{
			name:        "simple sound pattern",
			response:    `$: sound("bd hh sd hh")`,
			wantContent: `$: sound("bd hh sd hh")`,
			wantIsCode:  true,
		},
		{
			name:        "pattern with setcpm",
			response:    "setcpm(120)\n\n$: sound(\"bd*4\")",
			wantContent: "setcpm(120)\n\n$: sound(\"bd*4\")",
			wantIsCode:  true,
		},
		{
			name:        "note pattern",
			response:    `$: note("c3 e3 g3").sound("sawtooth")`,
			wantContent: `$: note("c3 e3 g3").sound("sawtooth")`,
			wantIsCode:  true,
		},
		{
			name:        "stack pattern",
			response:    `$: stack(sound("bd*4"), sound("hh*8"))`,
			wantContent: `$: stack(sound("bd*4"), sound("hh*8"))`,
			wantIsCode:  true,
		},
		{
			name:        "method chain",
			response:    `$: s("bd").fast(4).gain(0.8)`,
			wantContent: `$: s("bd").fast(4).gain(0.8)`,
			wantIsCode:  true,
		},

		// single fence - should extract code
		{
			name:        "single fence extracts code",
			response:    "```javascript\n$: sound(\"bd\").lpf(400)\n```",
			wantContent: `$: sound("bd").lpf(400)`,
			wantIsCode:  true,
		},
		{
			name:        "single fence with preamble extracts code",
			response:    "Here's the code:\n```js\nsetcpm(120)\n$: sound(\"bd*4\")\n```",
			wantContent: "setcpm(120)\n$: sound(\"bd*4\")",
			wantIsCode:  true,
		},
		{
			name:        "single fence no language identifier",
			response:    "```\n$: sound(\"hh*8\")\n```",
			wantContent: `$: sound("hh*8")`,
			wantIsCode:  true,
		},

		// multiple fences - keep as explanation
		{
			name:        "multiple fences is explanation",
			response:    "You can use lpf like this:\n```js\n$: sound(\"bd\").lpf(400)\n```\nOr with a pattern:\n```js\n$: sound(\"bd\").lpf(\"<400 800>\")\n```",
			wantContent: "You can use lpf like this:\n```js\n$: sound(\"bd\").lpf(400)\n```\nOr with a pattern:\n```js\n$: sound(\"bd\").lpf(\"<400 800>\")\n```",
			wantIsCode:  false,
		},

		// pure explanations (no code patterns, no fences)
		{
			name:        "pure explanation",
			response:    "The note() function allows you to play melodic patterns. You can specify notes using letter names like c3, e3, g3.",
			wantContent: "The note() function allows you to play melodic patterns. You can specify notes using letter names like c3, e3, g3.",
			wantIsCode:  false,
		},
		{
			name:        "empty response",
			response:    "",
			wantContent: "",
			wantIsCode:  false,
		},
		{
			name:        "explanation asking followup",
			response:    "I can help you create a bassline. What tempo would you like? What style are you going for?",
			wantContent: "I can help you create a bassline. What tempo would you like? What style are you going for?",
			wantIsCode:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotContent, gotIsCode := analyzeResponse(tt.response)
			if gotContent != tt.wantContent {
				t.Errorf("analyzeResponse() content = %q, want %q", gotContent, tt.wantContent)
			}
			if gotIsCode != tt.wantIsCode {
				t.Errorf("analyzeResponse() isCode = %v, want %v", gotIsCode, tt.wantIsCode)
			}
		})
	}
}

func TestExtractCodeFromFence(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
	}{
		{
			name:     "javascript fence",
			response: "```javascript\n$: sound(\"bd\")\n```",
			want:     `$: sound("bd")`,
		},
		{
			name:     "js fence",
			response: "```js\nsetcpm(120)\n```",
			want:     "setcpm(120)",
		},
		{
			name:     "no language identifier",
			response: "```\n$: note(\"c3\")\n```",
			want:     `$: note("c3")`,
		},
		{
			name:     "with surrounding text",
			response: "Here's your code:\n```js\n$: sound(\"hh*8\")\n```\nEnjoy!",
			want:     `$: sound("hh*8")`,
		},
		{
			name:     "multiline code",
			response: "```js\nsetcpm(120)\n\n$: sound(\"bd*4\")\n$: sound(\"hh*8\")\n```",
			want:     "setcpm(120)\n\n$: sound(\"bd*4\")\n$: sound(\"hh*8\")",
		},
		{
			name:     "no fence",
			response: "$: sound(\"bd\")",
			want:     "",
		},
		{
			name:     "unclosed fence",
			response: "```js\n$: sound(\"bd\")",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCodeFromFence(tt.response)
			if got != tt.want {
				t.Errorf("extractCodeFromFence() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasCodePatterns(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
	}{
		{"pattern registration", "$: sound(\"bd\")", true},
		{"setcpm", "setcpm(120)", true},
		{"sound with string", "sound(\"bd hh\")", true},
		{"note with string", "note(\"c3 e3\")", true},
		{"stack", "stack(a, b)", true},
		{"method chain fast", "x).fast(2)", true},
		{"method chain gain", "x).gain(0.5)", true},
		{"plain text", "This is just text", false},
		{"mentions note() function", "The note() function is useful", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasCodePatterns(tt.response)
			if got != tt.want {
				t.Errorf("hasCodePatterns() = %v, want %v", got, tt.want)
			}
		})
	}
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
