package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"codeberg.org/algorave/server/internal/llm"
	"codeberg.org/algorave/server/internal/strudel"
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

// analyzeResponse determines if an LLM response is code or explanation,
// and extracts code from markdown fences if present.
// Returns the content to display and whether it should be treated as code.
func analyzeResponse(response string) (content string, isCode bool) {
	if response == "" {
		return "", false
	}

	fenceCount := strings.Count(response, "```")

	// no fences - check for raw code patterns
	if fenceCount == 0 {
		return response, hasCodePatterns(response)
	}

	// single fence pair - extract the code
	if fenceCount == 2 {
		code := extractCodeFromFence(response)
		if code != "" {
			return code, true
		}
	}

	// odd fence count (1, 3, 5...) = malformed markdown
	// even fence count > 2 (4, 6...) = multiple code blocks (tutorial/explanation)
	// either way, keep as-is and mark as non-code
	return response, false
}

// extractCodeFromFence extracts code content from a single markdown fence pair.
// Returns empty string if extraction fails.
func extractCodeFromFence(response string) string {
	startIdx := strings.Index(response, "```")
	if startIdx == -1 {
		return ""
	}

	// find end of opening fence line (skip language identifier)
	afterStart := startIdx + 3
	newlineIdx := strings.Index(response[afterStart:], "\n")
	if newlineIdx == -1 {
		return ""
	}
	codeStart := afterStart + newlineIdx + 1

	// find closing fence
	endIdx := strings.Index(response[codeStart:], "```")
	if endIdx == -1 {
		return ""
	}

	code := strings.TrimSpace(response[codeStart : codeStart+endIdx])
	return code
}

// hasCodePatterns checks if the response contains definitive Strudel code patterns.
func hasCodePatterns(response string) bool {
	// definitive code patterns - these only appear in actual code
	definitivePatterns := []string{
		"$:",       // pattern registration (always code)
		"setcpm(",  // tempo setting (always code)
		"sound(\"", // sound with string arg
		"note(\"",  // note with string arg
		"stack(",   // stack function
		"s(\"",     // short sound alias with string arg
		"n(\"",     // short note alias with string arg
		").fast(",  // method chain after closing paren
		").slow(",
		").gain(",
		").lpf(",
		").hpf(",
		").room(",
		").delay(",
		").bank(",
	}

	for _, pattern := range definitivePatterns {
		if strings.Contains(response, pattern) {
			return true
		}
	}

	return false
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
