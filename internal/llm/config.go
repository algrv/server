package llm

import (
	"fmt"
	"os"
	"strconv"
)

// loadConfig loads LLM configuration from environment variables
func loadConfig() (*Config, error) {
	// transformer configuration
	transformerProvider := Provider(os.Getenv("TRANSFORMER_PROVIDER"))
	if transformerProvider == "" {
		transformerProvider = ProviderAnthropic // default
	}

	transformerAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	if transformerAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	transformerModel := os.Getenv("TRANSFORMER_MODEL")
	if transformerModel == "" {
		transformerModel = "claude-3-haiku-20240307" // default
	}

	// embedder configuration
	embedderProvider := Provider(os.Getenv("EMBEDDER_PROVIDER"))
	if embedderProvider == "" {
		embedderProvider = ProviderOpenAI // default
	}

	embedderAPIKey := os.Getenv("OPENAI_API_KEY")
	if embedderAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	embedderModel := os.Getenv("EMBEDDER_MODEL")
	if embedderModel == "" {
		embedderModel = "text-embedding-3-small" // default
	}

	// optional parameters
	maxTokens := 200 // default
	if maxTokensStr := os.Getenv("TRANSFORMER_MAX_TOKENS"); maxTokensStr != "" {
		if val, err := strconv.Atoi(maxTokensStr); err == nil {
			maxTokens = val
		}
	}

	temperature := float32(0.3) // default
	if tempStr := os.Getenv("TRANSFORMER_TEMPERATURE"); tempStr != "" {
		if val, err := strconv.ParseFloat(tempStr, 32); err == nil {
			temperature = float32(val)
		}
	}

	return &Config{
		TransformerProvider: transformerProvider,
		TransformerAPIKey:   transformerAPIKey,
		TransformerModel:    transformerModel,
		EmbedderProvider:    embedderProvider,
		EmbedderAPIKey:      embedderAPIKey,
		EmbedderModel:       embedderModel,
		MaxTokens:           maxTokens,
		Temperature:         temperature,
	}, nil
}
