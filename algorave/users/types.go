package users

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// handles user database operations
type Repository struct {
	db *pgxpool.Pool
}

// represents an authenticated user in the system
type User struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Provider   string    `json:"provider"`
	ProviderID string    `json:"-"`
	Name       string    `json:"name"`
	AvatarURL  string    `json:"avatar_url"`
	Tier       string    `json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// contains data for updating a user's profile
type UpdateProfileRequest struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}
