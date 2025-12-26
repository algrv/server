package llm

import (
	"context"
	"fmt"
)

// combines a QueryTransformer, Embedder, and TextGenerator into a single LLM
type CompositeLLM struct {
	QueryTransformer
	Embedder
	TextGenerator
}

// creates a new LLM with auto-configuration from environment variables
func NewLLM(ctx context.Context) (LLM, error) {
	config, err := loadConfig()

	if err != nil {
		return nil, fmt.Errorf("failed to load LLM config: %w", err)
	}

	return NewLLMWithConfig(ctx, config)
}

// creates a new LLM with explicit configuration
func NewLLMWithConfig(ctx context.Context, config *Config) (LLM, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// create transformer based on provider (for query transformation)
	var transformer QueryTransformer

	switch config.TransformerProvider {
	case ProviderAnthropic:
		transformer = NewAnthropicTransformer(AnthropicConfig{
			APIKey:      config.TransformerAPIKey,
			Model:       config.TransformerModel,
			MaxTokens:   config.TransformerMaxTokens,
			Temperature: config.TransformerTemperature,
		})
	default:
		return nil, fmt.Errorf("unsupported transformer provider: %s", config.TransformerProvider)
	}

	// create generator based on provider (for code generation)
	var textGenerator TextGenerator

	switch config.GeneratorProvider {
	case ProviderAnthropic:
		textGenerator = NewAnthropicTransformer(AnthropicConfig{
			APIKey:      config.GeneratorAPIKey,
			Model:       config.GeneratorModel,
			MaxTokens:   config.GeneratorMaxTokens,
			Temperature: config.GeneratorTemperature,
		})
	default:
		return nil, fmt.Errorf("unsupported generator provider: %s", config.GeneratorProvider)
	}

	// create embedder based on provider
	var embedder Embedder

	switch config.EmbedderProvider {
	case ProviderOpenAI:
		embedder = NewOpenAIEmbedder(OpenAIConfig{
			APIKey: config.EmbedderAPIKey,
			Model:  config.EmbedderModel,
		})
	default:
		return nil, fmt.Errorf("unsupported embedder provider: %s", config.EmbedderProvider)
	}

	return &CompositeLLM{
		QueryTransformer: transformer,
		Embedder:         embedder,
		TextGenerator:    textGenerator,
	}, nil
}
