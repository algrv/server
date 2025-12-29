package users

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// creates a new user repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// finds a user by OAuth provider or creates a new one
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

// finds a user by their ID
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

// updates a user's name and avatar URL
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
