package buffer

import (
	"context"
	"sync"
	"time"

	"github.com/algrv/server/algorave/sessions"
	"github.com/algrv/server/internal/logger"
)

// handles periodic flushing of buffered data from Redis to Postgres
type Flusher struct {
	buffer      *SessionBuffer
	sessionRepo sessions.Repository
	interval    time.Duration
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// creates a new flusher that periodically flushes Redis to Postgres
func NewFlusher(buffer *SessionBuffer, sessionRepo sessions.Repository, interval time.Duration) *Flusher {
	return &Flusher{
		buffer:      buffer,
		sessionRepo: sessionRepo,
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}

// begins the background flush loop
func (f *Flusher) Start() {
	f.wg.Add(1)
	go f.run()
	logger.Info("buffer flusher started", "interval", f.interval.String())
}

// gracefully stops the flusher and flushes any remaining data
func (f *Flusher) Stop() {
	close(f.stopCh)
	f.wg.Wait()
	logger.Info("buffer flusher stopped")
}

func (f *Flusher) run() {
	defer f.wg.Done()

	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.flush()
		case <-f.stopCh:
			// final flush before stopping
			logger.Info("flushing remaining buffer data before shutdown")
			f.flush()
			return
		}
	}
}

func (f *Flusher) flush() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// flush code updates
	f.flushCode(ctx)

	// flush messages
	f.flushMessages(ctx)
}

func (f *Flusher) flushCode(ctx context.Context) {
	sessionIDs, err := f.buffer.GetDirtyCodeSessions(ctx)
	if err != nil {
		logger.ErrorErr(err, "failed to get dirty code sessions")
		return
	}

	if len(sessionIDs) == 0 {
		return
	}

	logger.Debug("flushing code for sessions", "count", len(sessionIDs))

	for _, sessionID := range sessionIDs {
		code, err := f.buffer.FlushCode(ctx, sessionID)
		if err != nil {
			logger.ErrorErr(err, "failed to flush code from buffer", "session_id", sessionID)
			continue
		}

		if code == "" {
			continue
		}

		if err := f.sessionRepo.UpdateSessionCode(ctx, sessionID, code); err != nil {
			logger.ErrorErr(err, "failed to persist code to postgres", "session_id", sessionID)
			// re-add to dirty set so we retry next flush
			f.buffer.SetCode(ctx, sessionID, code) //nolint:errcheck,gosec // best-effort retry
		} else {
			logger.Debug("flushed code to postgres", "session_id", sessionID)
		}
	}
}

func (f *Flusher) flushMessages(ctx context.Context) {
	sessionIDs, err := f.buffer.GetDirtyMessageSessions(ctx)
	if err != nil {
		logger.ErrorErr(err, "failed to get dirty message sessions")
		return
	}

	if len(sessionIDs) == 0 {
		return
	}

	logger.Debug("flushing messages for sessions", "count", len(sessionIDs))

	for _, sessionID := range sessionIDs {
		messages, err := f.buffer.FlushMessages(ctx, sessionID)
		if err != nil {
			logger.ErrorErr(err, "failed to flush messages from buffer", "session_id", sessionID)
			continue
		}

		for _, msg := range messages {
			_, err := f.sessionRepo.AddMessage(
				ctx,
				msg.SessionID,
				msg.UserID,
				msg.Role,
				msg.MessageType,
				msg.Content,
				msg.IsActionable,
				msg.IsCodeResponse,
				msg.DisplayName,
				msg.AvatarURL,
			)
			if err != nil {
				logger.ErrorErr(err, "failed to persist message to postgres",
					"session_id", msg.SessionID,
					"message_type", msg.MessageType,
				)
				// re-add failed message to buffer
				f.buffer.AddMessage(ctx, &msg) //nolint:errcheck,gosec // best-effort retry
			}
		}
	}
}

// immediately flushes all data for a specific session
func (f *Flusher) FlushSession(ctx context.Context, sessionID string) error {
	// flush code
	code, err := f.buffer.FlushCode(ctx, sessionID)
	if err != nil {
		return err
	}

	if code != "" {
		if err := f.sessionRepo.UpdateSessionCode(ctx, sessionID, code); err != nil {
			logger.ErrorErr(err, "failed to persist code on session flush", "session_id", sessionID)
		}
	}

	// flush messages
	messages, err := f.buffer.FlushMessages(ctx, sessionID)
	if err != nil {
		return err
	}

	for _, msg := range messages {
		_, err := f.sessionRepo.AddMessage(
			ctx,
			msg.SessionID,
			msg.UserID,
			msg.Role,
			msg.MessageType,
			msg.Content,
			msg.IsActionable,
			msg.IsCodeResponse,
			msg.DisplayName,
			msg.AvatarURL,
		)
		if err != nil {
			logger.ErrorErr(err, "failed to persist message on session flush",
				"session_id", msg.SessionID,
			)
		}
	}

	return nil
}
