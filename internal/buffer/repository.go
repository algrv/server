package buffer

import (
	"context"
	"time"

	"codeberg.org/algorave/server/algorave/sessions"
	"codeberg.org/algorave/server/internal/logger"
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

// buffers chat message to Redis instead of direct Postgres write
func (r *BufferedRepository) AddChatMessage(
	ctx context.Context,
	sessionID, userID, content, displayName, avatarURL string,
) (*sessions.Message, error) {
	msg := &BufferedChatMessage{
		SessionID:   sessionID,
		UserID:      userID,
		Content:     content,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		CreatedAt:   time.Now(),
	}

	if err := r.buffer.AddChatMessage(ctx, msg); err != nil {
		logger.ErrorErr(err, "failed to buffer chat message", "session_id", sessionID)
		// fall back to direct DB write
		return r.db.AddChatMessage(ctx, sessionID, userID, content, displayName, avatarURL)
	}

	// return a placeholder message (real one will be created on flush)
	return &sessions.Message{
		SessionID: sessionID,
		Role:      "user",
		Content:   content,
		CreatedAt: msg.CreatedAt,
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

// GetChatMessages retrieves chat messages for a session
func (r *BufferedRepository) GetChatMessages(ctx context.Context, sessionID string, limit int) ([]*sessions.Message, error) {
	// get messages from Postgres
	dbMessages, err := r.db.GetChatMessages(ctx, sessionID, limit)
	if err != nil {
		return nil, err
	}

	// get unflushed messages from Redis buffer
	bufferedMsgs, err := r.buffer.GetBufferedChatMessages(ctx, sessionID)
	if err != nil {
		// log but don't fail - Postgres messages are still valid
		logger.Warn("failed to get buffered chat messages", "session_id", sessionID, "error", err)
		return dbMessages, nil
	}

	if len(bufferedMsgs) == 0 {
		return dbMessages, nil
	}

	// convert buffered messages to session messages
	for _, bm := range bufferedMsgs {
		msg := &sessions.Message{
			SessionID: bm.SessionID,
			Role:      "user",
			Content:   bm.Content,
			CreatedAt: bm.CreatedAt,
		}
		if bm.DisplayName != "" {
			msg.DisplayName = &bm.DisplayName
		}
		if bm.AvatarURL != "" {
			msg.AvatarURL = &bm.AvatarURL
		}
		dbMessages = append(dbMessages, msg)
	}

	return dbMessages, nil
}

func (r *BufferedRepository) UpdateLastActivity(ctx context.Context, sessionID string) error {
	return r.db.UpdateLastActivity(ctx, sessionID)
}

// === NEW SOFT-END AND CLEANUP OPERATIONS ===

func (r *BufferedRepository) RevokeAllInviteTokens(ctx context.Context, sessionID string) error {
	return r.db.RevokeAllInviteTokens(ctx, sessionID)
}

func (r *BufferedRepository) HasActiveInviteTokens(ctx context.Context, sessionID string) (bool, error) {
	return r.db.HasActiveInviteTokens(ctx, sessionID)
}

func (r *BufferedRepository) MarkAllNonHostParticipantsLeft(ctx context.Context, sessionID, hostUserID string) error {
	return r.db.MarkAllNonHostParticipantsLeft(ctx, sessionID, hostUserID)
}

func (r *BufferedRepository) GetLastUserSession(ctx context.Context, userID string) (*sessions.Session, error) {
	return r.db.GetLastUserSession(ctx, userID)
}

func (r *BufferedRepository) ListStaleSessions(ctx context.Context, threshold time.Time) ([]*sessions.Session, error) {
	return r.db.ListStaleSessions(ctx, threshold)
}

func (r *BufferedRepository) CountActiveParticipants(ctx context.Context, sessionID string) (int, error) {
	return r.db.CountActiveParticipants(ctx, sessionID)
}
