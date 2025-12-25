package agent

import (
	"context"
	"fmt"

	"github.com/algorave/server/internal/llm"
	"github.com/algorave/server/internal/retriever"
)

// creates a new agent with auto-configuration from environment
func NewAgent(ctx context.Context) (*Agent, error) {
	retrieverClient, err := retriever.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create retriever: %w", err)
	}

	llmClient, err := llm.NewLLM(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	return NewAgentWithDeps(retrieverClient, llmClient), nil
}

// creates a new agent with explicit dependencies
func NewAgentWithDeps(ret Retriever, llm llm.LLM) *Agent {
	return &Agent{
		retriever: ret,
		llm:       llm,
	}
}

// closes the agent and its dependencies
func (a *Agent) Close() {
	if a.retriever != nil {
		a.retriever.Close()
	}
}

// generates Strudel code using RAG
func (a *Agent) GenerateCode(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	// retrieve relevant documentation using hybrid search
	docs, err := a.retriever.HybridSearchDocs(ctx, req.UserQuery, req.EditorState, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve docs: %w", err)
	}

	// retrieve relevant examples using hybrid search
	examples, err := a.retriever.HybridSearchExamples(ctx, req.UserQuery, req.EditorState, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve examples: %w", err)
	}

	// build the system prompt with all context
	systemPrompt := buildSystemPrompt(SystemPromptContext{
		Cheatsheet:    getCheatsheet(),
		EditorState:   req.EditorState,
		Docs:          docs,
		Examples:      examples,
		Conversations: req.ConversationHistory,
	})

	// call Claude API for code generation
	response, err := a.callLLM(ctx, systemPrompt, req.UserQuery, req.ConversationHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	return &GenerateResponse{
		Code:              response,
		DocsRetrieved:     len(docs),
		ExamplesRetrieved: len(examples),
		Model:             "claude-sonnet-4-20250514", // from env/config
	}, nil
}

// makes the actual API call using the LLM abstraction
func (a *Agent) callLLM(ctx context.Context, systemPrompt, userQuery string, history []Message) (string, error) {
	// convert agent Messages to LLM Messages
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

	// call LLM to generate code
	response, err := a.llm.GenerateText(ctx, llm.TextGenerationRequest{
		SystemPrompt: systemPrompt,
		Messages:     llmMessages,
		MaxTokens:    4096, // allow longer responses for code generation
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate text: %w", err)
	}

	return response, nil
}
