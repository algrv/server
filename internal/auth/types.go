package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

// represents JWT claims
type Claims struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
	jwt.RegisteredClaims
}
