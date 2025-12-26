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

	// generator configuration
	generatorProvider := Provider(os.Getenv("GENERATOR_PROVIDER"))
	if generatorProvider == "" {
		generatorProvider = ProviderAnthropic // default
	}

	generatorAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	if generatorAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	generatorModel := os.Getenv("GENERATOR_MODEL")
	if generatorModel == "" {
		generatorModel = "claude-sonnet-4-20250514" // default
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

	// transformer optional parameters
	transformerMaxTokens := 200 // default
	if maxTokensStr := os.Getenv("TRANSFORMER_MAX_TOKENS"); maxTokensStr != "" {
		if val, err := strconv.Atoi(maxTokensStr); err == nil {
			transformerMaxTokens = val
		}
	}

	transformerTemperature := float32(0.3) // default
	if tempStr := os.Getenv("TRANSFORMER_TEMPERATURE"); tempStr != "" {
		if val, err := strconv.ParseFloat(tempStr, 32); err == nil {
			transformerTemperature = float32(val)
		}
	}

	// generator optional parameters
	generatorMaxTokens := 4096 // default
	if maxTokensStr := os.Getenv("GENERATOR_MAX_TOKENS"); maxTokensStr != "" {
		if val, err := strconv.Atoi(maxTokensStr); err == nil {
			generatorMaxTokens = val
		}
	}

	generatorTemperature := float32(0.7) // default
	if tempStr := os.Getenv("GENERATOR_TEMPERATURE"); tempStr != "" {
		if val, err := strconv.ParseFloat(tempStr, 32); err == nil {
			generatorTemperature = float32(val)
		}
	}

	return &Config{
		TransformerProvider:    transformerProvider,
		TransformerAPIKey:      transformerAPIKey,
		TransformerModel:       transformerModel,
		TransformerMaxTokens:   transformerMaxTokens,
		TransformerTemperature: transformerTemperature,
		GeneratorProvider:      generatorProvider,
		GeneratorAPIKey:        generatorAPIKey,
		GeneratorModel:         generatorModel,
		GeneratorMaxTokens:     generatorMaxTokens,
		GeneratorTemperature:   generatorTemperature,
		EmbedderProvider:       embedderProvider,
		EmbedderAPIKey:         embedderAPIKey,
		EmbedderModel:          embedderModel,
	}, nil
}
