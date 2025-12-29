package users

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindOrCreateByProvider(
	ctx context.Context,
	provider, providerID, email, name, avatarURL string,
) (*User, error) {
	var user User

	err := r.db.QueryRow(
		ctx,
		queryFindOrCreateByProvider,
		provider,
		providerID,
		email,
		name,
		avatarURL,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Provider,
		&user.ProviderID,
		&user.Name,
		&user.AvatarURL,
		&user.Tier,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) FindByID(ctx context.Context, userID string) (*User, error) {
	var user User

	err := r.db.QueryRow(ctx, queryFindByID, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Provider,
		&user.ProviderID,
		&user.Name,
		&user.AvatarURL,
		&user.Tier,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) UpdateProfile(
	ctx context.Context,
	userID, name, avatarURL string,
) (*User, error) {
	var user User

	err := r.db.QueryRow(
		ctx,
		queryUpdateProfile,
		name,
		avatarURL,
		userID,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Provider,
		&user.ProviderID,
		&user.Name,
		&user.AvatarURL,
		&user.Tier,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) CheckUserRateLimit(ctx context.Context, userID string, isBYOK bool) (*RateLimitResult, error) {
	if isBYOK {
		return &RateLimitResult{
			Allowed:   true,
			Current:   0,
			Limit:     DailyLimitBYOK,
			Remaining: -1,
		}, nil
	}

	user, err := r.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var limit int
	switch user.Tier {
	case "pro":
		limit = DailyLimitPro
	case "byok":
		limit = DailyLimitBYOK
	default:
		limit = DailyLimitFree
	}

	if limit == -1 {
		return &RateLimitResult{
			Allowed:   true,
			Current:   0,
			Limit:     limit,
			Remaining: -1,
		}, nil
	}

	var current int
	err = r.db.QueryRow(ctx, queryGetUserDailyUsage, userID).Scan(&current)
	if err != nil {
		return nil, err
	}

	remaining := limit - current
	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitResult{
		Allowed:   current < limit,
		Current:   current,
		Limit:     limit,
		Remaining: remaining,
	}, nil
}

func (r *Repository) CheckSessionRateLimit(ctx context.Context, sessionID string) (*RateLimitResult, error) {
	var current int
	err := r.db.QueryRow(ctx, queryGetSessionDailyUsage, sessionID).Scan(&current)
	if err != nil {
		return nil, err
	}

	remaining := DailyLimitAnonymous - current
	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitResult{
		Allowed:   current < DailyLimitAnonymous,
		Current:   current,
		Limit:     DailyLimitAnonymous,
		Remaining: remaining,
	}, nil
}

func (r *Repository) LogUsage(ctx context.Context, req *UsageLogRequest) error {
	_, err := r.db.Exec(
		ctx,
		queryLogUsage,
		req.UserID,
		req.SessionID,
		req.Provider,
		req.Model,
		req.InputTokens,
		req.OutputTokens,
		req.IsBYOK,
	)
	return err
}
