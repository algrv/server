package notifications

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, req *CreateRequest) (*Notification, error) {
	var n Notification
	var dataJSON *string

	if req.Data != nil {
		bytes, err := json.Marshal(req.Data)
		if err != nil {
			return nil, err
		}

		str := string(bytes)
		dataJSON = &str
	}

	err := s.db.QueryRow(
		ctx,
		queryCreate,
		req.UserID,
		req.Type,
		req.Title,
		req.Body,
		dataJSON,
	).Scan(
		&n.ID,
		&n.UserID,
		&n.Type,
		&n.Title,
		&n.Body,
		&n.Read,
		&n.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &n, nil
}

func (s *Service) ListForUser(ctx context.Context, userID string, limit int, unreadOnly bool) ([]Notification, error) {
	query := queryListForUser
	if unreadOnly {
		query = queryListUnreadForUser
	}

	rows, err := s.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var notifications []Notification

	for rows.Next() {
		var n Notification
		var dataJSON []byte

		err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.Type,
			&n.Title,
			&n.Body,
			&dataJSON,
			&n.Read,
			&n.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		if len(dataJSON) > 0 {
			if err := json.Unmarshal(dataJSON, &n.Data); err != nil {
				n.Data = nil // ignore malformed JSON
			}
		}

		notifications = append(notifications, n)
	}

	return notifications, rows.Err()
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) error {
	_, err := s.db.Exec(ctx, queryMarkRead, notificationID, userID)
	return err
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	_, err := s.db.Exec(ctx, queryMarkAllRead, userID)
	return err
}

func (s *Service) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, queryUnreadCount, userID).Scan(&count)
	return count, err
}
