package websocket

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"slices"
	"strings"

	"codeberg.org/algorave/server/internal/logger"
)

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
