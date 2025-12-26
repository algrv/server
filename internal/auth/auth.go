package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/apple"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
)

// sets up all OAuth providers using goth
func InitializeProviders() error {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	if os.Getenv("GOOGLE_CLIENT_ID") == "" || os.Getenv("GOOGLE_CLIENT_SECRET") == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must be set")
	}

	providers := []goth.Provider{
		google.New(
			os.Getenv("GOOGLE_CLIENT_ID"),
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			baseURL+"/api/v1/auth/google/callback",
			"email", "profile",
		),
	}

	if os.Getenv("GITHUB_CLIENT_ID") != "" && os.Getenv("GITHUB_CLIENT_SECRET") != "" {
		providers = append(providers, github.New(
			os.Getenv("GITHUB_CLIENT_ID"),
			os.Getenv("GITHUB_CLIENT_SECRET"),
			baseURL+"/api/v1/auth/github/callback",
			"user:email",
		))
	}

	if os.Getenv("APPLE_CLIENT_ID") != "" && os.Getenv("APPLE_CLIENT_SECRET") != "" {
		providers = append(providers, apple.New(
			os.Getenv("APPLE_CLIENT_ID"),
			os.Getenv("APPLE_CLIENT_SECRET"),
			baseURL+"/api/v1/auth/apple/callback",
			nil,
			apple.ScopeName, apple.ScopeEmail,
		))
	}

	goth.UseProviders(providers...)
	return nil
}

// creates a JWT token for the user
func GenerateJWT(userID, email string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET not set")
	}

	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// validates a JWT token and returns the claims
func ValidateJWT(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET not set")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
