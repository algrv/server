package llm

import (
	"context"
	"fmt"
)

// creates a new LLM with config from environment variables
func NewLLM(ctx context.Context) (LLM, error) {
	config, err := loadConfig()

	if err != nil {
		return nil, fmt.Errorf("failed to load LLM config: %w", err)
	}

	return NewLLMWithConfig(ctx, config)
}

// creates a new LLM with explicit configuration
func NewLLMWithConfig(_ context.Context, config *Config) (LLM, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	var transformer QueryTransformer

	switch config.TransformerProvider {
	case ProviderAnthropic:
		transformer = NewAnthropicTransformer(AnthropicConfig{
			APIKey:      config.TransformerAPIKey,
			Model:       config.TransformerModel,
			MaxTokens:   config.TransformerMaxTokens,
			Temperature: config.TransformerTemperature,
		})
	case ProviderOpenAI:
		transformer = NewOpenAIGenerator(OpenAIConfig{
			APIKey: config.TransformerAPIKey,
			Model:  config.TransformerModel,
		})
	default:
		return nil, fmt.Errorf("unsupported transformer provider: %s", config.TransformerProvider)
	}

	var textGenerator TextGenerator

	switch config.GeneratorProvider {
	case ProviderAnthropic:
		textGenerator = NewAnthropicTransformer(AnthropicConfig{
			APIKey:      config.GeneratorAPIKey,
			Model:       config.GeneratorModel,
			MaxTokens:   config.GeneratorMaxTokens,
			Temperature: config.GeneratorTemperature,
		})
	case ProviderOpenAI:
		textGenerator = NewOpenAIGenerator(OpenAIConfig{
			APIKey: config.GeneratorAPIKey,
			Model:  config.GeneratorModel,
		})
	default:
		return nil, fmt.Errorf("unsupported generator provider: %s", config.GeneratorProvider)
	}

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
