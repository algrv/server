package sessions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db}
}

// creates a new collaborative session
func (r *repository) CreateSession(ctx context.Context, req *CreateSessionRequest) (*Session, error) {
	var session Session

	err := r.db.QueryRow(
		ctx,
		queryCreateSession,
		req.HostUserID,
		req.Title,
		req.Code,
	).Scan(
		&session.ID,
		&session.HostUserID,
		&session.Title,
		&session.Code,
		&session.IsActive,
		&session.IsDiscoverable,
		&session.CreatedAt,
		&session.EndedAt,
		&session.LastActivity,
	)

	if err != nil {
		return nil, err
	}

	return &session, nil
}

// creates a new anonymous session (no host user)
func (r *repository) CreateAnonymousSession(ctx context.Context) (*Session, error) {
	var session Session

	err := r.db.QueryRow(ctx, queryCreateAnonymousSession).Scan(
		&session.ID,
		&session.HostUserID,
		&session.Title,
		&session.Code,
		&session.IsActive,
		&session.IsDiscoverable,
		&session.CreatedAt,
		&session.EndedAt,
		&session.LastActivity,
	)

	if err != nil {
		return nil, err
	}

	return &session, nil
}

// retrieves a session by ID
func (r *repository) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	var session Session

	err := r.db.QueryRow(ctx, queryGetSession, sessionID).Scan(
		&session.ID,
		&session.HostUserID,
		&session.Title,
		&session.Code,
		&session.IsActive,
		&session.IsDiscoverable,
		&session.CreatedAt,
		&session.EndedAt,
		&session.LastActivity,
	)

	if err != nil {
		return nil, err
	}

	return &session, nil
}

// retrieves all sessions for a user (as host or participant)
func (r *repository) GetUserSessions(ctx context.Context, userID string, activeOnly bool) ([]*Session, error) {
	query := queryGetUserSessions

	if activeOnly {
		query = queryGetUserSessionsActiveOnly
	}

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var sessions []*Session

	for rows.Next() {
		var s Session
		err := rows.Scan(
			&s.ID,
			&s.HostUserID,
			&s.Title,
			&s.Code,
			&s.IsActive,
			&s.IsDiscoverable,
			&s.CreatedAt,
			&s.EndedAt,
			&s.LastActivity,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

// lists all discoverable active sessions with pagination
func (r *repository) ListDiscoverableSessions(ctx context.Context, limit, offset int) ([]*Session, int, error) {
	// get total count first
	var total int

	if err := r.db.QueryRow(ctx, queryCountDiscoverableSessions).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, queryListDiscoverableSessions, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()
	var sessions []*Session

	for rows.Next() {
		var s Session
		err := rows.Scan(
			&s.ID,
			&s.HostUserID,
			&s.Title,
			&s.Code,
			&s.IsActive,
			&s.IsDiscoverable,
			&s.CreatedAt,
			&s.EndedAt,
			&s.LastActivity,
		)
		if err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return sessions, total, nil
}

// sets the discoverable flag for a session
func (r *repository) SetDiscoverable(ctx context.Context, sessionID string, isDiscoverable bool) error {
	_, err := r.db.Exec(ctx, querySetDiscoverable, isDiscoverable, sessionID)
	return err
}

func (r *repository) UpdateSessionCode(ctx context.Context, sessionID, code string) error {
	_, err := r.db.Exec(ctx, queryUpdateSessionCode, code, sessionID)
	return err
}

func (r *repository) EndSession(ctx context.Context, sessionID string) error {
	_, err := r.db.Exec(ctx, queryEndSession, sessionID)
	return err
}

func (r *repository) AddAuthenticatedParticipant(
	ctx context.Context,
	sessionID, userID, displayName, role string,
) (*Participant, error) {
	var participant Participant

	err := r.db.QueryRow(
		ctx,
		queryAddAuthenticatedParticipant,
		sessionID,
		userID,
		displayName,
		role,
	).Scan(
		&participant.ID,
		&participant.SessionID,
		&participant.UserID,
		&participant.DisplayName,
		&participant.Role,
		&participant.Status,
		&participant.JoinedAt,
		&participant.LeftAt,
	)

	if err != nil {
		return nil, err
	}

	return &participant, nil
}

func (r *repository) GetAuthenticatedParticipant(ctx context.Context, sessionID, userID string) (*Participant, error) {
	var participant Participant

	err := r.db.QueryRow(ctx, queryGetAuthenticatedParticipant, sessionID, userID).Scan(
		&participant.ID,
		&participant.SessionID,
		&participant.UserID,
		&participant.DisplayName,
		&participant.Role,
		&participant.Status,
		&participant.JoinedAt,
		&participant.LeftAt,
	)

	if err != nil {
		return nil, err
	}

	return &participant, nil
}

func (r *repository) MarkAuthenticatedParticipantLeft(ctx context.Context, participantID string) error {
	_, err := r.db.Exec(ctx, queryMarkAuthenticatedParticipantLeft, participantID)
	return err
}

// retrieves a participant by their ID (authenticated or anonymous)
func (r *repository) GetParticipantByID(ctx context.Context, participantID string) (*CombinedParticipant, error) {
	// try authenticated participants first
	var authParticipant Participant
	err := r.db.QueryRow(ctx, queryGetParticipantByID, participantID).Scan(
		&authParticipant.ID,
		&authParticipant.SessionID,
		&authParticipant.UserID,
		&authParticipant.DisplayName,
		&authParticipant.Role,
		&authParticipant.Status,
		&authParticipant.JoinedAt,
		&authParticipant.LeftAt,
	)

	if err == nil {
		return &CombinedParticipant{
			ID:          authParticipant.ID,
			SessionID:   authParticipant.SessionID,
			UserID:      &authParticipant.UserID,
			DisplayName: authParticipant.DisplayName,
			Role:        authParticipant.Role,
			Status:      authParticipant.Status,
			JoinedAt:    authParticipant.JoinedAt,
			LeftAt:      authParticipant.LeftAt,
		}, nil
	}

	// try anonymous participants
	var anonParticipant AnonymousParticipant
	err = r.db.QueryRow(ctx, queryGetAnonymousParticipantByID, participantID).Scan(
		&anonParticipant.ID,
		&anonParticipant.SessionID,
		&anonParticipant.DisplayName,
		&anonParticipant.Role,
		&anonParticipant.Status,
		&anonParticipant.JoinedAt,
		&anonParticipant.LeftAt,
		&anonParticipant.ExpiresAt,
	)

	if err != nil {
		return nil, err
	}

	return &CombinedParticipant{
		ID:          anonParticipant.ID,
		SessionID:   anonParticipant.SessionID,
		UserID:      nil,
		DisplayName: anonParticipant.DisplayName,
		Role:        anonParticipant.Role,
		Status:      anonParticipant.Status,
		JoinedAt:    anonParticipant.JoinedAt,
		LeftAt:      anonParticipant.LeftAt,
	}, nil
}

func (r *repository) UpdateParticipantRole(ctx context.Context, participantID, role string) error {
	_, err := r.db.Exec(ctx, queryUpdateParticipantRole, role, participantID)
	return err
}

func (r *repository) AddAnonymousParticipant(
	ctx context.Context,
	sessionID, displayName, role string,
) (*AnonymousParticipant, error) {
	var participant AnonymousParticipant

	err := r.db.QueryRow(
		ctx,
		queryAddAnonymousParticipant,
		sessionID,
		displayName,
		role,
	).Scan(
		&participant.ID,
		&participant.SessionID,
		&participant.DisplayName,
		&participant.Role,
		&participant.Status,
		&participant.JoinedAt,
		&participant.LeftAt,
		&participant.ExpiresAt,
	)

	if err != nil {
		return nil, err
	}

	return &participant, nil
}

// retrieves both authenticated and anonymous participants
func (r *repository) ListAllParticipants(ctx context.Context, sessionID string) ([]*CombinedParticipant, error) {
	var participants []*CombinedParticipant

	// get authenticated participants
	authRows, err := r.db.Query(ctx, queryListAuthenticatedParticipants, sessionID)
	if err != nil {
		return nil, err
	}

	defer authRows.Close()

	for authRows.Next() {
		var p Participant
		err := authRows.Scan(
			&p.ID,
			&p.SessionID,
			&p.UserID,
			&p.DisplayName,
			&p.Role,
			&p.Status,
			&p.JoinedAt,
			&p.LeftAt,
		)
		if err != nil {
			return nil, err
		}

		participants = append(participants, &CombinedParticipant{
			ID:          p.ID,
			SessionID:   p.SessionID,
			UserID:      &p.UserID,
			DisplayName: p.DisplayName,
			Role:        p.Role,
			Status:      p.Status,
			JoinedAt:    p.JoinedAt,
			LeftAt:      p.LeftAt,
		})
	}

	if err := authRows.Err(); err != nil {
		return nil, err
	}

	// get anonymous participants
	anonRows, err := r.db.Query(ctx, queryListAnonymousParticipants, sessionID)
	if err != nil {
		return nil, err
	}

	defer anonRows.Close()

	for anonRows.Next() {
		var p AnonymousParticipant
		err := anonRows.Scan(
			&p.ID,
			&p.SessionID,
			&p.DisplayName,
			&p.Role,
			&p.Status,
			&p.JoinedAt,
			&p.LeftAt,
			&p.ExpiresAt,
		)
		if err != nil {
			return nil, err
		}

		participants = append(participants, &CombinedParticipant{
			ID:          p.ID,
			SessionID:   p.SessionID,
			UserID:      nil,
			DisplayName: p.DisplayName,
			Role:        p.Role,
			Status:      p.Status,
			JoinedAt:    p.JoinedAt,
			LeftAt:      p.LeftAt,
		})
	}

	if err := anonRows.Err(); err != nil {
		return nil, err
	}

	return participants, nil
}

// removes a participant (authenticated or anonymous)
func (r *repository) RemoveParticipant(ctx context.Context, participantID string) error {
	// try authenticated participants first
	result, err := r.db.Exec(ctx, queryRemoveAuthenticatedParticipant, participantID)
	if err != nil {
		return err
	}

	if result.RowsAffected() > 0 {
		return nil
	}

	// try anonymous participants
	_, err = r.db.Exec(ctx, queryRemoveAnonymousParticipant, participantID)
	return err
}

// creates a new invite token for a session
func (r *repository) CreateInviteToken(ctx context.Context, req *CreateInviteTokenRequest) (*InviteToken, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	var inviteToken InviteToken

	err = r.db.QueryRow(
		ctx,
		queryCreateInviteToken,
		req.SessionID,
		token,
		req.Role,
		req.MaxUses,
		req.ExpiresAt,
	).Scan(
		&inviteToken.ID,
		&inviteToken.SessionID,
		&inviteToken.Token,
		&inviteToken.Role,
		&inviteToken.MaxUses,
		&inviteToken.UsesCount,
		&inviteToken.ExpiresAt,
		&inviteToken.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &inviteToken, nil
}

// retrieves all invite tokens for a session
func (r *repository) ListInviteTokens(ctx context.Context, sessionID string) ([]*InviteToken, error) {
	rows, err := r.db.Query(ctx, queryListInviteTokens, sessionID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var tokens []*InviteToken

	for rows.Next() {
		var t InviteToken

		err := rows.Scan(
			&t.ID,
			&t.SessionID,
			&t.Token,
			&t.Role,
			&t.MaxUses,
			&t.UsesCount,
			&t.ExpiresAt,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

// validates and retrieves an invite token
func (r *repository) ValidateInviteToken(ctx context.Context, token string) (*InviteToken, error) {
	var inviteToken InviteToken

	err := r.db.QueryRow(ctx, queryValidateInviteToken, token).Scan(
		&inviteToken.ID,
		&inviteToken.SessionID,
		&inviteToken.Token,
		&inviteToken.Role,
		&inviteToken.MaxUses,
		&inviteToken.UsesCount,
		&inviteToken.ExpiresAt,
		&inviteToken.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &inviteToken, nil
}

func (r *repository) IncrementTokenUses(ctx context.Context, tokenID string) error {
	_, err := r.db.Exec(ctx, queryIncrementTokenUses, tokenID)
	return err
}

func (r *repository) RevokeInviteToken(ctx context.Context, tokenID string) error {
	_, err := r.db.Exec(ctx, queryRevokeInviteToken, tokenID)
	return err
}

// retrieves chat messages for a session
func (r *repository) GetChatMessages(ctx context.Context, sessionID string, limit int) ([]*Message, error) {
	rows, err := r.db.Query(ctx, queryGetChatMessages, sessionID, limit)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var messages []*Message

	for rows.Next() {
		var m Message
		err := rows.Scan(
			&m.ID,
			&m.SessionID,
			&m.UserID,
			&m.Role,
			&m.Content,
			&m.DisplayName,
			&m.AvatarURL,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// adds a chat message to the session
func (r *repository) AddChatMessage(
	ctx context.Context,
	sessionID, userID, content, displayName, avatarURL string,
) (*Message, error) {
	// convert empty strings to nil pointers
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	var displayNamePtr *string
	if displayName != "" {
		displayNamePtr = &displayName
	}

	var avatarURLPtr *string
	if avatarURL != "" {
		avatarURLPtr = &avatarURL
	}

	var message Message
	err := r.db.QueryRow(
		ctx,
		queryAddChatMessage,
		sessionID,
		userIDPtr,
		content,
		displayNamePtr,
		avatarURLPtr,
	).Scan(
		&message.ID,
		&message.SessionID,
		&message.UserID,
		&message.Role,
		&message.Content,
		&message.DisplayName,
		&message.AvatarURL,
		&message.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &message, nil
}

// updates the last activity timestamp for a session
func (r *repository) UpdateLastActivity(ctx context.Context, sessionID string) error {
	_, err := r.db.Exec(ctx, queryUpdateLastActivity, sessionID)
	return err
}

// generates a cryptographically secure random token
func generateToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}
