package agent

import (
	"context"
	"fmt"

	"github.com/algoraveai/server/internal/llm"
)

func New(ret Retriever, llmClient llm.LLM) *Agent {
	return &Agent{
		retriever: ret,
		generator: llmClient,
	}
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

	return &GenerateResponse{
		Code:              response.Text,
		DocsRetrieved:     len(docs),
		ExamplesRetrieved: len(examples),
		Model:             textGenerator.Model(),
		IsActionable:      true,
		InputTokens:       response.Usage.InputTokens,
		OutputTokens:      response.Usage.OutputTokens,
	}, nil
}

func (a *Agent) callGeneratorWithClient(ctx context.Context, generator llm.TextGenerator, systemPrompt, userQuery string, history []Message) (*llm.TextGenerationResponse, error) {
	llmMessages := make([]llm.Message, 0, len(history)+1)

	for _, msg := range history {
		llmMessages = append(llmMessages, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	llmMessages = append(llmMessages, llm.Message{
		Role:    "user",
		Content: userQuery,
	})

	response, err := generator.GenerateText(ctx, llm.TextGenerationRequest{
		SystemPrompt: systemPrompt,
		Messages:     llmMessages,
		MaxTokens:    4096,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate text: %w", err)
	}

	return response, nil
}
