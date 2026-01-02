package users

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DailyLimitAnonymous = 50   // anonymous users: 50/day
	DailyLimitFree      = 100  // free tier: 100/day
	DailyLimitPAYG      = 1000 // payg tier: 1000/day (pay as you go)
	DailyLimitBYOK      = -1   // BYOK: unlimited (using own keys)
)

const (
	MinuteLimitDefault = 10 // default for all users
	MinuteLimitPAYG    = 20 // payg tier
	MinuteLimitBYOK    = 30 // BYOK
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
