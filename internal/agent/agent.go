package agent

import (
	"context"
	"fmt"

	"github.com/algorave/server/internal/llm"
)

func New(ret Retriever, llmClient llm.LLM) *Agent {
	return &Agent{
		retriever: ret,
		generator: llmClient,
	}
}

func (a *Agent) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	// analyze query for actionability
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
			Model:               a.generator.Model(),
		}, nil
	}

	// proceed with code generation for actionable queries
	docs, err := a.retriever.HybridSearchDocs(ctx, req.UserQuery, req.EditorState, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve docs: %w", err)
	}

	examples, err := a.retriever.HybridSearchExamples(ctx, req.UserQuery, req.EditorState, 3)
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

	// call LLM for code generation
	response, err := a.callGenerator(ctx, systemPrompt, req.UserQuery, req.ConversationHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	return &GenerateResponse{
		Code:              response,
		DocsRetrieved:     len(docs),
		ExamplesRetrieved: len(examples),
		Model:             a.generator.Model(),
		IsActionable:      true,
	}, nil
}

// makes the actual API call using the TextGenerator
func (a *Agent) callGenerator(ctx context.Context, systemPrompt, userQuery string, history []Message) (string, error) {
	llmMessages := make([]llm.Message, 0, len(history)+1)

	// add conversation history
	for _, msg := range history {
		llmMessages = append(llmMessages, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// add current user query
	llmMessages = append(llmMessages, llm.Message{
		Role:    "user",
		Content: userQuery,
	})

	response, err := a.generator.GenerateText(ctx, llm.TextGenerationRequest{
		SystemPrompt: systemPrompt,
		Messages:     llmMessages,
		MaxTokens:    4096,
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate text: %w", err)
	}

	return response, nil
}
