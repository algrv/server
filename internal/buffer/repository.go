package buffer

import (
	"context"
	"time"

	"github.com/algrv/server/algorave/sessions"
	"github.com/algrv/server/internal/logger"
)

// wraps a sessions.Repository with Redis buffering
// write operations go to Redis first, reads fall through to Postgres
type BufferedRepository struct {
	db     sessions.Repository
	buffer *SessionBuffer
}

// creates a new buffered repository wrapper
func NewBufferedRepository(db sessions.Repository, buffer *SessionBuffer) *BufferedRepository {
	return &BufferedRepository{
		db:     db,
		buffer: buffer,
	}
}

// === BUFFERED WRITE OPERATIONS ===

// writes to Redis buffer instead of Postgres
func (r *BufferedRepository) UpdateSessionCode(ctx context.Context, sessionID, code string) error {
	if err := r.buffer.SetCode(ctx, sessionID, code); err != nil {
		logger.ErrorErr(err, "failed to buffer code", "session_id", sessionID)
		// fall back to direct DB write
		return r.db.UpdateSessionCode(ctx, sessionID, code)
	}
	return nil
}

// buffers to Redis instead of direct Postgres write
func (r *BufferedRepository) AddMessage(
	ctx context.Context,
	sessionID, userID, role, messageType, content string,
	isActionable, isCodeResponse bool,
	displayName, avatarURL string,
) (*sessions.Message, error) {
	msg := &BufferedMessage{
		SessionID:      sessionID,
		UserID:         userID,
		Role:           role,
		MessageType:    messageType,
		Content:        content,
		IsActionable:   isActionable,
		IsCodeResponse: isCodeResponse,
		DisplayName:    displayName,
		AvatarURL:      avatarURL,
		CreatedAt:      time.Now(),
	}

	if err := r.buffer.AddMessage(ctx, msg); err != nil {
		logger.ErrorErr(err, "failed to buffer message", "session_id", sessionID)
		// fall back to direct DB write
		return r.db.AddMessage(ctx, sessionID, userID, role, messageType, content, isActionable, isCodeResponse, displayName, avatarURL)
	}

	// return a placeholder message (real one will be created on flush)
	return &sessions.Message{
		SessionID:      sessionID,
		Role:           role,
		MessageType:    messageType,
		Content:        content,
		IsActionable:   isActionable,
		IsCodeResponse: isCodeResponse,
		CreatedAt:      msg.CreatedAt,
	}, nil
}

// === PASS-THROUGH OPERATIONS (no buffering needed) ===

func (r *BufferedRepository) CreateSession(ctx context.Context, req *sessions.CreateSessionRequest) (*sessions.Session, error) {
	return r.db.CreateSession(ctx, req)
}

func (r *BufferedRepository) CreateAnonymousSession(ctx context.Context) (*sessions.Session, error) {
	return r.db.CreateAnonymousSession(ctx)
}

func (r *BufferedRepository) GetSession(ctx context.Context, sessionID string) (*sessions.Session, error) {
	session, err := r.db.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// check Redis for fresher code (may not have been flushed yet)
	if code, err := r.buffer.GetCode(ctx, sessionID); err == nil && code != "" {
		session.Code = code
	}

	return session, nil
}

func (r *BufferedRepository) GetUserSessions(ctx context.Context, userID string, activeOnly bool) ([]*sessions.Session, error) {
	return r.db.GetUserSessions(ctx, userID, activeOnly)
}

func (r *BufferedRepository) ListDiscoverableSessions(ctx context.Context, limit, offset int) ([]*sessions.Session, int, error) {
	return r.db.ListDiscoverableSessions(ctx, limit, offset)
}

func (r *BufferedRepository) SetDiscoverable(ctx context.Context, sessionID string, isDiscoverable bool) error {
	return r.db.SetDiscoverable(ctx, sessionID, isDiscoverable)
}

func (r *BufferedRepository) EndSession(ctx context.Context, sessionID string) error {
	return r.db.EndSession(ctx, sessionID)
}

func (r *BufferedRepository) AddAuthenticatedParticipant(ctx context.Context, sessionID, userID, displayName, role string) (*sessions.Participant, error) {
	return r.db.AddAuthenticatedParticipant(ctx, sessionID, userID, displayName, role)
}

func (r *BufferedRepository) GetAuthenticatedParticipant(ctx context.Context, sessionID, userID string) (*sessions.Participant, error) {
	return r.db.GetAuthenticatedParticipant(ctx, sessionID, userID)
}

func (r *BufferedRepository) MarkAuthenticatedParticipantLeft(ctx context.Context, participantID string) error {
	return r.db.MarkAuthenticatedParticipantLeft(ctx, participantID)
}

func (r *BufferedRepository) GetParticipantByID(ctx context.Context, participantID string) (*sessions.CombinedParticipant, error) {
	return r.db.GetParticipantByID(ctx, participantID)
}

func (r *BufferedRepository) UpdateParticipantRole(ctx context.Context, participantID, role string) error {
	return r.db.UpdateParticipantRole(ctx, participantID, role)
}

func (r *BufferedRepository) AddAnonymousParticipant(ctx context.Context, sessionID, displayName, role string) (*sessions.AnonymousParticipant, error) {
	return r.db.AddAnonymousParticipant(ctx, sessionID, displayName, role)
}

func (r *BufferedRepository) ListAllParticipants(ctx context.Context, sessionID string) ([]*sessions.CombinedParticipant, error) {
	return r.db.ListAllParticipants(ctx, sessionID)
}

func (r *BufferedRepository) RemoveParticipant(ctx context.Context, participantID string) error {
	return r.db.RemoveParticipant(ctx, participantID)
}

func (r *BufferedRepository) CreateInviteToken(ctx context.Context, req *sessions.CreateInviteTokenRequest) (*sessions.InviteToken, error) {
	return r.db.CreateInviteToken(ctx, req)
}

func (r *BufferedRepository) ListInviteTokens(ctx context.Context, sessionID string) ([]*sessions.InviteToken, error) {
	return r.db.ListInviteTokens(ctx, sessionID)
}

func (r *BufferedRepository) ValidateInviteToken(ctx context.Context, token string) (*sessions.InviteToken, error) {
	return r.db.ValidateInviteToken(ctx, token)
}

func (r *BufferedRepository) IncrementTokenUses(ctx context.Context, tokenID string) error {
	return r.db.IncrementTokenUses(ctx, tokenID)
}

func (r *BufferedRepository) RevokeInviteToken(ctx context.Context, tokenID string) error {
	return r.db.RevokeInviteToken(ctx, tokenID)
}

func (r *BufferedRepository) GetMessages(ctx context.Context, sessionID string, limit int) ([]*sessions.Message, error) {
	return r.db.GetMessages(ctx, sessionID, limit)
}

func (r *BufferedRepository) CreateMessage(
	ctx context.Context,
	sessionID string,
	userID *string,
	role, messageType, content string,
	isActionable, isCodeResponse bool,
	displayName, avatarURL *string,
) (*sessions.Message, error) {
	return r.db.CreateMessage(ctx, sessionID, userID, role, messageType, content, isActionable, isCodeResponse, displayName, avatarURL)
}

func (r *BufferedRepository) UpdateLastActivity(ctx context.Context, sessionID string) error {
	return r.db.UpdateLastActivity(ctx, sessionID)
}
