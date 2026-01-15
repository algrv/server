package llm

import (
	"fmt"
	"os"
	"strconv"

	"codeberg.org/algorave/server/internal/config"
)

func loadConfig() (*Config, error) {
	baseConfig, err := config.LoadEnvironmentVariables()
	if err != nil {
		return nil, fmt.Errorf("failed to load base config: %w", err)
	}

	// transformer config
	transformerProvider := Provider(os.Getenv("TRANSFORMER_PROVIDER"))
	if transformerProvider == "" {
		transformerProvider = ProviderAnthropic // default
	}

	transformerAPIKey := getAPIKeyForProvider(transformerProvider, baseConfig)

	transformerModel := os.Getenv("TRANSFORMER_MODEL")
	if transformerModel == "" {
		transformerModel = "claude-3-haiku-20240307" // default
	}

	// generator config
	generatorProvider := Provider(os.Getenv("GENERATOR_PROVIDER"))
	if generatorProvider == "" {
		generatorProvider = ProviderAnthropic // default
	}

	generatorAPIKey := getAPIKeyForProvider(generatorProvider, baseConfig)

	generatorModel := os.Getenv("GENERATOR_MODEL")
	if generatorModel == "" {
		generatorModel = "claude-sonnet-4-20250514" // default
	}

	// embedder config
	embedderProvider := Provider(os.Getenv("EMBEDDER_PROVIDER"))
	if embedderProvider == "" {
		embedderProvider = ProviderOpenAI // default
	}

	// use API key from base config
	embedderAPIKey := baseConfig.OpenAIKey

	embedderModel := os.Getenv("EMBEDDER_MODEL")
	if embedderModel == "" {
		embedderModel = "text-embedding-3-small" // default
	}

	// optional transformer parameters
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

	// optional generator parameters
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
