package llm

import "codeberg.org/algopatterns/server/internal/config"

// returns the appropriate API key for the given provider
func getAPIKeyForProvider(provider Provider, baseConfig *config.Config) string {
	switch provider {
	case ProviderOpenAI:
		return baseConfig.OpenAIKey
	default:
		return baseConfig.AnthropicKey
	}
}
