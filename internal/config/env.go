package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// loads configuration from environment variables
func LoadEnvironmentVariables() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		_ = err // not an error - production environments may not have .env file
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	supabaseConnStr := os.Getenv("SUPABASE_CONNECTION_STRING")
	redisURL := os.Getenv("REDIS_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
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

	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL environment variable is required")
	}

	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	if environment == "" {
		environment = "development"
	}

	return &Config{
		OpenAIKey:          openaiKey,
		AnthropicKey:       anthropicKey,
		SupabaseConnString: supabaseConnStr,
		RedisURL:           redisURL,
		Environment:        environment,
	}, nil
}
