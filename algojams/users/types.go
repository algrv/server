package users

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DailyLimitAnonymous = 4    // anonymous users: 4/day
	DailyLimitFree      = 4    // free tier: 4/day
	DailyLimitPAYG      = 1000 // payg tier: 1000/day (pay as you go)
	DailyLimitBYOK      = -1   // BYOK: unlimited (using own keys)
)

type Repository struct {
	db *pgxpool.Pool
}

type User struct {
	ID                string    `json:"id"`
	Email             string    `json:"email"`
	Provider          string    `json:"provider"`
	ProviderID        string    `json:"-"`
	Name              string    `json:"name"`
	AvatarURL         string    `json:"avatar_url"`
	Tier              string    `json:"-"`
	IsAdmin           bool      `json:"-"` // not exposed to clients
	TrainingConsent   bool      `json:"training_consent"`
	AIFeaturesEnabled bool      `json:"ai_features_enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type UpdateProfileRequest struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

type UsageLogRequest struct {
	UserID       *string // nil for anonymous
	SessionID    string  // for anonymous users
	Provider     string  // "anthropic", "openai"
	Model        string  // model name
	InputTokens  int     // estimated input tokens
	OutputTokens int     // estimated output tokens
	IsBYOK       bool    // true if user provided own API key
}

type RateLimitResult struct {
	Allowed   bool
	Current   int
	Limit     int
	Remaining int
}
