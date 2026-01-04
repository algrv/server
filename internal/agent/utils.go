package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/algrv/server/internal/llm"
	"github.com/algrv/server/internal/strudel"
)

var (
	cheatsheetCache string
	cheatsheetOnce  sync.Once
)

func getCheatsheet() string {
	cheatsheetOnce.Do(func() {
		content, err := os.ReadFile(filepath.Join("resources", "cheatsheet.md"))

		if err != nil {
			cheatsheetCache = ""
			return
		}

		cheatsheetCache = string(content)
	})

	return cheatsheetCache
}

// formats code with line numbers for error context.
func addLineNumbers(code string) string {
	lines := strings.Split(code, "\n")
	var builder strings.Builder

	for i, line := range lines {
		builder.WriteString(fmt.Sprintf("%3d | %s\n", i+1, line))
	}

	return builder.String()
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

// retries code generation with validation error context.
func (a *Agent) retryWithValidationError(
	ctx context.Context,
	generator llm.TextGenerator,
	systemPrompt, userQuery string,
	history []Message,
	invalidCode string,
	validationResult *strudel.ValidationResult,
) (*llm.TextGenerationResponse, error) {
	retryHistory := make([]Message, 0, len(history)+2)
	retryHistory = append(retryHistory, history...)
	retryHistory = append(retryHistory, Message{
		Role:    "user",
		Content: userQuery,
	})

	retryHistory = append(retryHistory, Message{
		Role:    "assistant",
		Content: invalidCode,
	})

	// format code with line numbers so LLM can locate the error
	numberedCode := addLineNumbers(invalidCode)

	// build error message with location info if available
	errorMsg := validationResult.Error

	if validationResult.Line != nil {
		if validationResult.Column != nil {
			errorMsg = fmt.Sprintf("%s (line %d, column %d)", validationResult.Error, *validationResult.Line, *validationResult.Column)
		} else {
			errorMsg = fmt.Sprintf("%s (line %d)", validationResult.Error, *validationResult.Line)
		}
	}

	retryPrompt := fmt.Sprintf(`
	the code you generated has a syntax error and will not run: %s.
	Error: %s.
	please fix the error and return only the corrected strudel code.
	do not include any explanation or line numbers.`, numberedCode, errorMsg)

	return a.callGeneratorWithClient(ctx, generator, systemPrompt, retryPrompt, retryHistory)
}
