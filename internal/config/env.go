package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	OpenAIKey          string
	AnthropicKey       string
	SupabaseConnString string
	Environment        string
}

func LoadEnvironmentVariables() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		_ = err
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	supabaseConnStr := os.Getenv("SUPABASE_CONNECTION_STRING")
	environment := os.Getenv("ENVIRONMENT")

	if openaiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	if anthropicKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	if supabaseConnStr == "" {
		return nil, fmt.Errorf("SUPABASE_CONNECTION_STRING environment variable is required")
	}

	if environment == "" {
		environment = "development"
	}

	return &Config{
		OpenAIKey:          openaiKey,
		AnthropicKey:       anthropicKey,
		SupabaseConnString: supabaseConnStr,
		Environment:        environment,
	}, nil
}
