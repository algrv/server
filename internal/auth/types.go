package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// represents an authenticated user
type User struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Provider   string    `json:"provider"`
	ProviderID string    `json:"-"`
	Name       string    `json:"name"`
	AvatarURL  string    `json:"avatar_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// represents JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}
