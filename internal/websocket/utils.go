package websocket

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/algrv/server/internal/errors"
	"github.com/algrv/server/internal/llm"
	"github.com/algrv/server/internal/logger"
)

func createBYOKGenerator(provider, apiKey string) (llm.TextGenerator, error) {
	switch provider {
	case "openai":
		return llm.NewOpenAIGenerator(llm.OpenAIConfig{
			APIKey: apiKey,
			Model:  "gpt-4o",
		}), nil
	case "claude":
		return llm.NewAnthropicTransformer(llm.AnthropicConfig{
			APIKey:      apiKey,
			Model:       "claude-sonnet-4-20250514",
			MaxTokens:   4096,
			Temperature: 0.7,
		}), nil
	default:
		return nil, errors.ErrUnsupportedProvider(provider)
	}
}

func getAllowedWebSocketOrigins() []string {
	if envOrigins := os.Getenv("ALLOWED_ORIGINS"); envOrigins != "" {
		origins := strings.Split(envOrigins, ",")

		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}

		return origins
	}

	return []string{}
}

func CheckOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	if origin == "" {
		// allow no origin header in development
		env := os.Getenv("ENVIRONMENT")

		if env != "production" {
			return true
		}

		logger.Warn("websocket connection with no origin header")
		return false
	}

	env := os.Getenv("ENVIRONMENT")
	if env != "production" {
		return true
	}

	// production: validate against allowed origins
	allowedOrigins := getAllowedWebSocketOrigins()

	if len(allowedOrigins) == 0 {
		logger.Warn("websocket origin rejected - ALLOWED_ORIGINS not configured",
			"origin", origin,
		)
		return false
	}

	if slices.Contains(allowedOrigins, origin) {
		return true
	}

	logger.Warn("websocket origin rejected - not in allowed origins",
		"origin", origin,
		"allowed_origins", allowedOrigins,
	)

	return false
}

func GenerateClientID() (string, error) {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
