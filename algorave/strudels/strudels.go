package strudels

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrStrudelNotFound = errors.New("strudel not found")
)

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(
	ctx context.Context,
	userID string,
	req CreateStrudelRequest,
) (*Strudel, error) {
	var strudel Strudel

	// initialize empty arrays if nil to avoid null in JSON responses
	tags := req.Tags

	if tags == nil {
		tags = []string{}
	}

	categories := req.Categories

	if categories == nil {
		categories = []string{}
	}

	err := r.db.QueryRow(
		ctx,
		queryCreate,
		userID,
		req.Title,
		req.Code,
		req.IsPublic,
		req.Description,
		tags,
		categories,
		req.ConversationHistory,
	).Scan(
		&strudel.ID,
		&strudel.UserID,
		&strudel.Title,
		&strudel.Code,
		&strudel.IsPublic,
		&strudel.UseInTraining,
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

func (r *Repository) List(ctx context.Context, userID string, limit, offset int) ([]Strudel, int, error) {
	// get total count first

	var total int
	if err := r.db.QueryRow(ctx, queryCountByUser, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, queryList, userID, limit, offset)
	if err != nil {
		return nil, 0, err
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
			&s.UseInTraining,
			&s.Description,
			&s.Tags,
			&s.Categories,
			&s.ConversationHistory,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		strudels = append(strudels, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return strudels, total, nil
}

func (r *Repository) ListPublic(ctx context.Context, limit, offset int) ([]Strudel, int, error) {
	// Get total count first
	var total int
	if err := r.db.QueryRow(ctx, queryCountPublic).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, queryListPublic, limit, offset)
	if err != nil {
		return nil, 0, err
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
			&s.UseInTraining,
			&s.Description,
			&s.Tags,
			&s.Categories,
			&s.ConversationHistory,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		strudels = append(strudels, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return strudels, total, nil
}

func (r *Repository) GetPublic(ctx context.Context, strudelID string) (*Strudel, error) {
	var strudel Strudel

	err := r.db.QueryRow(ctx, queryGetPublic, strudelID).Scan(
		&strudel.ID,
		&strudel.UserID,
		&strudel.Title,
		&strudel.Code,
		&strudel.IsPublic,
		&strudel.UseInTraining,
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

func (r *Repository) Get(ctx context.Context, strudelID, userID string) (*Strudel, error) {
	var strudel Strudel

	err := r.db.QueryRow(ctx, queryGet, strudelID, userID).Scan(
		&strudel.ID,
		&strudel.UserID,
		&strudel.Title,
		&strudel.Code,
		&strudel.IsPublic,
		&strudel.UseInTraining,
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
		&strudel.UseInTraining,
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

// ListTrainableWithoutEmbedding returns strudels that are trainable but don't have embeddings yet
func (r *Repository) ListTrainableWithoutEmbedding(ctx context.Context, limit int) ([]Strudel, error) {
	rows, err := r.db.Query(ctx, queryListTrainableWithoutEmbedding, limit)
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
			&s.UseInTraining,
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

// UpdateEmbedding sets the embedding vector for a strudel
func (r *Repository) UpdateEmbedding(ctx context.Context, strudelID string, embedding []float32) error {
	_, err := r.db.Exec(ctx, queryUpdateEmbedding, embedding, strudelID)
	return err
}

// AdminGetStrudel gets any strudel by ID (admin only, no user check)
func (r *Repository) AdminGetStrudel(ctx context.Context, strudelID string) (*Strudel, error) {
	var strudel Strudel

	err := r.db.QueryRow(ctx, queryAdminGetStrudel, strudelID).Scan(
		&strudel.ID,
		&strudel.UserID,
		&strudel.Title,
		&strudel.Code,
		&strudel.IsPublic,
		&strudel.UseInTraining,
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

// AdminSetUseInTraining sets the use_in_training flag (admin only)
func (r *Repository) AdminSetUseInTraining(ctx context.Context, strudelID string, useInTraining bool) (*Strudel, error) {
	var strudel Strudel

	err := r.db.QueryRow(ctx, queryAdminSetUseInTraining, useInTraining, strudelID).Scan(
		&strudel.ID,
		&strudel.UserID,
		&strudel.Title,
		&strudel.Code,
		&strudel.IsPublic,
		&strudel.UseInTraining,
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
