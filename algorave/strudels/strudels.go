package strudels

import (
	"context"
	"errors"
	"fmt"

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

func (r *Repository) List(ctx context.Context, userID string, limit, offset int, filter ListFilter) ([]Strudel, int, error) {
	// build dynamic query with filters
	baseWhere := "WHERE user_id = $1"
	args := []interface{}{userID}
	argIndex := 2

	if filter.Search != "" {
		baseWhere += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	if len(filter.Tags) > 0 {
		baseWhere += fmt.Sprintf(" AND tags && $%d", argIndex)
		args = append(args, filter.Tags)
		argIndex++
	}

	// get total count first
	var total int
	countQuery := "SELECT COUNT(*) FROM user_strudels " + baseWhere
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// build list query
	listQuery := fmt.Sprintf(`
		SELECT id, user_id, title, code, is_public, use_in_training, description, tags, categories, conversation_history, created_at, updated_at
		FROM user_strudels
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
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

func (r *Repository) ListPublic(ctx context.Context, limit, offset int, filter ListFilter) ([]Strudel, int, error) {
	// build dynamic query with filters
	baseWhere := "WHERE is_public = true"
	args := []interface{}{}
	argIndex := 1

	if filter.Search != "" {
		baseWhere += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	if len(filter.Tags) > 0 {
		baseWhere += fmt.Sprintf(" AND tags && $%d", argIndex)
		args = append(args, filter.Tags)
		argIndex++
	}

	// get total count first
	var total int
	countQuery := "SELECT COUNT(*) FROM user_strudels " + baseWhere

	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// build list query
	listQuery := fmt.Sprintf(`
		SELECT id, user_id, title, code, is_public, use_in_training, description, tags, categories, conversation_history, created_at, updated_at
		FROM user_strudels
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
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

// returns strudels that are trainable but don't have embeddings yet
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

// sets the embedding vector for a strudel
func (r *Repository) UpdateEmbedding(ctx context.Context, strudelID string, embedding []float32) error {
	_, err := r.db.Exec(ctx, queryUpdateEmbedding, embedding, strudelID)
	return err
}

// gets any strudel by ID (admin only, no user check)
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

// sets the use_in_training flag (admin only)
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

// returns all unique tags from public strudels
func (r *Repository) ListPublicTags(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, queryListPublicTags)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

// returns all unique tags from a user's strudels
func (r *Repository) ListUserTags(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, queryListUserTags, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

// adds an AI conversation message to a strudel
func (r *Repository) AddStrudelMessage(ctx context.Context, req *AddStrudelMessageRequest) (*StrudelMessage, error) {
	var msg StrudelMessage
	var displayName *string
	if req.DisplayName != "" {
		displayName = &req.DisplayName
	}

	err := r.db.QueryRow(
		ctx,
		queryAddStrudelMessage,
		req.StrudelID,
		req.UserID,
		req.Role,
		req.Content,
		req.IsActionable,
		req.IsCodeResponse,
		req.ClarifyingQuestions,
		displayName,
	).Scan(
		&msg.ID,
		&msg.StrudelID,
		&msg.UserID,
		&msg.Role,
		&msg.Content,
		&msg.IsActionable,
		&msg.IsCodeResponse,
		&msg.ClarifyingQuestions,
		&msg.DisplayName,
		&msg.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}

// retrieves AI conversation messages for a strudel
func (r *Repository) GetStrudelMessages(ctx context.Context, strudelID string, limit int) ([]*StrudelMessage, error) {
	rows, err := r.db.Query(ctx, queryGetStrudelMessages, strudelID, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var messages []*StrudelMessage

	for rows.Next() {
		var msg StrudelMessage
		err := rows.Scan(
			&msg.ID,
			&msg.StrudelID,
			&msg.UserID,
			&msg.Role,
			&msg.Content,
			&msg.IsActionable,
			&msg.IsCodeResponse,
			&msg.ClarifyingQuestions,
			&msg.DisplayName,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
