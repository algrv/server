package agent

import (
	"context"
	"fmt"

	"codeberg.org/algorave/server/internal/llm"
	"codeberg.org/algorave/server/internal/strudel"
)

func New(ret Retriever, llmClient llm.LLM) *Agent {
	return &Agent{
		retriever: ret,
		generator: llmClient,
	}
}

// creates an agent with code validation enabled.
func NewWithValidator(ret Retriever, llmClient llm.LLM, validator *strudel.Validator) *Agent {
	return &Agent{
		retriever: ret,
		generator: llmClient,
		validator: validator,
	}
}

// sets the validator for the agent.
func (a *Agent) SetValidator(v *strudel.Validator) {
	a.validator = v
}

func (a *Agent) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	textGenerator := llm.TextGenerator(a.generator)

	if req.CustomGenerator != nil {
		textGenerator = req.CustomGenerator
	}

	// analyze query for actionability (always use platform's transformer)
	analysis, err := a.generator.AnalyzeQuery(ctx, req.UserQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query: %w", err)
	}

	// if query is not actionable, return clarifying questions
	if !analysis.IsActionable {
		return &GenerateResponse{
			IsActionable:        false,
			IsCodeResponse:      false,
			ClarifyingQuestions: analysis.ClarifyingQuestions,
			DocsRetrieved:       0,
			ExamplesRetrieved:   0,
			Model:               textGenerator.Model(),
		}, nil
	}

	// proceed with code generation for actionable queries
	docs, err := a.retriever.HybridSearchDocs(ctx, req.UserQuery, req.EditorState, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve docs: %w", err)
	}

	examples, err := a.retriever.HybridSearchExamples(ctx, req.UserQuery, req.EditorState, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve examples: %w", err)
	}

	systemPrompt := buildSystemPrompt(SystemPromptContext{
		Cheatsheet:    getCheatsheet(),
		EditorState:   req.EditorState,
		Docs:          docs,
		Examples:      examples,
		Conversations: req.ConversationHistory,
	})

	// call LLM for code generation (uses custom generator if BYOK)
	response, err := a.callGeneratorWithClient(ctx, textGenerator, systemPrompt, req.UserQuery, req.ConversationHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	totalInputTokens := response.Usage.InputTokens
	totalOutputTokens := response.Usage.OutputTokens
	didRetry := false
	var validationError string

	// validate and retry if validator is available
	if a.validator != nil && response.Text != "" {
		result, err := a.validator.Validate(ctx, response.Text)
		if err == nil && !result.Valid {
			retryResponse, retryErr := a.retryWithValidationError(
				ctx, textGenerator, systemPrompt, req.UserQuery,
				req.ConversationHistory, response.Text, result,
			)
			if retryErr == nil {
				response = retryResponse
				totalInputTokens += retryResponse.Usage.InputTokens
				totalOutputTokens += retryResponse.Usage.OutputTokens
				didRetry = true
			}

			validationError = result.Error
		}
	}

	// build references for frontend display
	strudelRefs := make([]StrudelReference, 0, len(examples))
	for _, ex := range examples {
		strudelRefs = append(strudelRefs, StrudelReference{
			ID:         ex.ID,
			Title:      ex.Title,
			AuthorName: ex.AuthorName,
			URL:        fmt.Sprintf("/strudel/%s", ex.ID),
		})
	}

	docRefs := make([]DocReference, 0, len(docs))
	seen := make(map[string]bool) // dedupe by page URL
	for _, doc := range docs {
		if seen[doc.PageURL] {
			continue
		}
		seen[doc.PageURL] = true
		docRefs = append(docRefs, DocReference{
			PageName:     doc.PageName,
			SectionTitle: doc.SectionTitle,
			URL:          doc.PageURL,
		})
	}

	// analyze response to determine if it's code and extract from markdown if needed
	content, isCode := analyzeResponse(response.Text)

	return &GenerateResponse{
		Code:              content,
		DocsRetrieved:     len(docs),
		ExamplesRetrieved: len(examples),
		Examples:          examples,
		Docs:              docs,
		StrudelReferences: strudelRefs,
		DocReferences:     docRefs,
		Model:             textGenerator.Model(),
		IsActionable:      true,
		IsCodeResponse:    isCode,
		InputTokens:       totalInputTokens,
		OutputTokens:      totalOutputTokens,
		DidRetry:          didRetry,
		ValidationError:   validationError,
	}, nil
}
