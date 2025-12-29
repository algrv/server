package strudels

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrStrudelNotFound = errors.New("strudel not found")
)

// creates a new strudel repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// creates a new strudel
func (r *Repository) Create(
	ctx context.Context,
	userID string,
	req CreateStrudelRequest,
) (*Strudel, error) {
	var strudel Strudel

	err := r.db.QueryRow(
		ctx,
		queryCreate,
		userID,
		req.Title,
		req.Code,
		req.IsPublic,
		req.Description,
		req.Tags,
		req.Categories,
		req.ConversationHistory,
	).Scan(
		&strudel.ID,
		&strudel.UserID,
		&strudel.Title,
		&strudel.Code,
		&strudel.IsPublic,
		&strudel.Description,
		&strudel.Tags,
		&strudel.Categories,
		&strudel.ConversationHistory,
		&strudel.CreatedAt,
		&strudel.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &strudel, nil
}

// retrieves all strudels for a user
func (r *Repository) List(ctx context.Context, userID string) ([]Strudel, error) {
	rows, err := r.db.Query(ctx, queryList, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strudels []Strudel

	for rows.Next() {
		var s Strudel
		err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.Title,
			&s.Code,
			&s.IsPublic,
			&s.Description,
			&s.Tags,
			&s.Categories,
			&s.ConversationHistory,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		strudels = append(strudels, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return strudels, nil
}

// retrieves publicly shared strudels
func (r *Repository) ListPublic(ctx context.Context, limit int) ([]Strudel, error) {
	rows, err := r.db.Query(ctx, queryListPublic, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var strudels []Strudel

	for rows.Next() {
		var s Strudel
		err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.Title,
			&s.Code,
			&s.IsPublic,
			&s.Description,
			&s.Tags,
			&s.Categories,
			&s.ConversationHistory,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		strudels = append(strudels, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return strudels, nil
}

// retrieves a specific strudel by ID for a user
func (r *Repository) Get(ctx context.Context, strudelID, userID string) (*Strudel, error) {
	var strudel Strudel

	err := r.db.QueryRow(ctx, queryGet, strudelID, userID).Scan(
		&strudel.ID,
		&strudel.UserID,
		&strudel.Title,
		&strudel.Code,
		&strudel.IsPublic,
		&strudel.Description,
		&strudel.Tags,
		&strudel.Categories,
		&strudel.ConversationHistory,
		&strudel.CreatedAt,
		&strudel.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &strudel, nil
}

// updates a strudel
func (r *Repository) Update(
	ctx context.Context,
	strudelID, userID string,
	req UpdateStrudelRequest,
) (*Strudel, error) {
	var strudel Strudel

	err := r.db.QueryRow(
		ctx,
		queryUpdate,
		req.Title,
		req.Code,
		req.IsPublic,
		req.Description,
		req.Tags,
		req.Categories,
		req.ConversationHistory,
		strudelID,
		userID,
	).Scan(
		&strudel.ID,
		&strudel.UserID,
		&strudel.Title,
		&strudel.Code,
		&strudel.IsPublic,
		&strudel.Description,
		&strudel.Tags,
		&strudel.Categories,
		&strudel.ConversationHistory,
		&strudel.CreatedAt,
		&strudel.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &strudel, nil
}

// deletes a strudel
func (r *Repository) Delete(ctx context.Context, strudelID, userID string) error {
	result, err := r.db.Exec(ctx, queryDelete, strudelID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrStrudelNotFound
	}

	return nil
}
