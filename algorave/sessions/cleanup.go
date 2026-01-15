package sessions

import (
	"context"
	"time"

	"codeberg.org/algorave/server/internal/logger"
)

// handles automatic expiry of stale sessions
type CleanupService struct {
	repo                Repository
	checkInterval       time.Duration
	inactivityThreshold time.Duration
	sessionEnder        SessionEnderFunc
}

// called to notify WebSocket clients when a session is being cleaned up
type SessionEnderFunc func(sessionID string, reason string)

// creates a new cleanup service
func NewCleanupService(
	repo Repository,
	checkInterval time.Duration,
	inactivityThreshold time.Duration,
	sessionEnder SessionEnderFunc,
) *CleanupService {
	return &CleanupService{
		repo:                repo,
		checkInterval:       checkInterval,
		inactivityThreshold: inactivityThreshold,
		sessionEnder:        sessionEnder,
	}
}

// begins the cleanup service background loop
func (s *CleanupService) Start(ctx context.Context) {
	logger.Info("starting session cleanup service",
		"check_interval", s.checkInterval,
		"inactivity_threshold", s.inactivityThreshold,
	)

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("session cleanup service stopped")
			return
		case <-ticker.C:
			s.cleanupStaleSessions(ctx)
		}
	}
}

// finds and ends sessions that have been inactive
func (s *CleanupService) cleanupStaleSessions(ctx context.Context) {
	threshold := time.Now().Add(-s.inactivityThreshold)

	staleSessions, err := s.repo.ListStaleSessions(ctx, threshold)
	if err != nil {
		logger.ErrorErr(err, "failed to list stale sessions")
		return
	}

	if len(staleSessions) == 0 {
		return
	}

	logger.Info("found stale sessions to clean up", "count", len(staleSessions))

	for _, session := range staleSessions {
		if err := s.endStaleSession(ctx, session); err != nil {
			logger.ErrorErr(err, "failed to end stale session",
				"session_id", session.ID,
				"last_activity", session.LastActivity,
			)
		}
	}
}

// performs cleanup for a single stale session
func (s *CleanupService) endStaleSession(ctx context.Context, session *Session) error {
	logger.Info("ending stale session",
		"session_id", session.ID,
		"title", session.Title,
		"last_activity", session.LastActivity,
	)

	// notify WebSocket clients before ending
	if s.sessionEnder != nil {
		s.sessionEnder(session.ID, "session expired due to inactivity")
	}

	// end the session in database
	if err := s.repo.EndSession(ctx, session.ID); err != nil {
		return err
	}

	logger.Info("stale session ended successfully", "session_id", session.ID)
	return nil
}
