package llm

import (
	"context"
	"fmt"
)

// combines a QueryTransformer and Embedder into a single LLM
type CompositeLLM struct {
	QueryTransformer
	Embedder
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

	// create transformer based on provider
	var transformer QueryTransformer

	switch config.TransformerProvider {
	case ProviderAnthropic:
		transformer = NewAnthropicTransformer(AnthropicConfig{
			APIKey:      config.TransformerAPIKey,
			Model:       config.TransformerModel,
			MaxTokens:   config.MaxTokens,
			Temperature: config.Temperature,
		})
	default:
		return nil, fmt.Errorf("unsupported transformer provider: %s", config.TransformerProvider)
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
	}, nil
}
