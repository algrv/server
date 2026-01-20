package websocket

import (
	"context"
	"strings"
	"time"

	"codeberg.org/algojams/server/algojams/sessions"
	"codeberg.org/algojams/server/internal/ccsignals"
	"codeberg.org/algojams/server/internal/logger"
)

// handles code update messages with CC signals detection
func CodeUpdateHandler(sessionRepo sessions.Repository, detector *ccsignals.Detector) MessageHandler {
	return func(hub *Hub, client *Client, msg *Message) error {
		// check rate limit
		if !client.checkCodeUpdateRateLimit() {
			client.SendError("too_many_requests", "too many code updates. maximum 10 per second.", "")
			return ErrRateLimitExceeded
		}

		// check if client has write permissions
		if !client.CanWrite() {
			client.SendError("forbidden", "you don't have permission to edit code", "")
			return ErrReadOnly
		}

		// parse payload
		var payload CodeUpdatePayload
		if err := msg.UnmarshalPayload(&payload); err != nil {
			client.SendError("validation_error", "failed to parse code update", err.Error())
			return err
		}

		// validate code size
		codeSize := len([]byte(payload.Code))
		if codeSize > maxCodeSize {
			client.SendError("bad_request", "code exceeds maximum size. maximum 100 KB allowed.", "")
			return ErrCodeTooLarge
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// get previous code for paste detection
		session, err := sessionRepo.GetSession(ctx, client.SessionID)
		previousCode := ""
		if err == nil && session != nil {
			previousCode = session.Code
		}

		// use ccsignals detector for paste detection
		// only check for 'paste' or 'typed' sources (skip loaded_strudel, forked)
		shouldCheckPaste := payload.Source == "" || payload.Source == "typed" || payload.Source == "paste"
		if detector != nil {
			if shouldCheckPaste {
				handlePasteDetection(ctx, hub, client, detector, previousCode, payload.Code)
			} else if payload.Source == "loaded_strudel" {
				// clear any existing paste lock when loading a new/fresh strudel
				wasLocked, err := detector.IsLocked(ctx, client.SessionID)
				if err != nil {
					logger.ErrorErr(err, "failed to check lock status on strudel load", "session_id", client.SessionID)
				} else if wasLocked {
					if err := detector.RemoveLock(ctx, client.SessionID); err != nil {
						logger.ErrorErr(err, "failed to clear paste lock on strudel load", "session_id", client.SessionID)
					} else {
						logger.Info("paste lock cleared on strudel load", "session_id", client.SessionID)
						sendPasteLockStatus(hub, client, false, "new_strudel")
					}
				}
			}
		}

		// save code (goes to redis buffer via BufferedRepository)
		if err := sessionRepo.UpdateSessionCode(ctx, client.SessionID, payload.Code); err != nil {
			logger.ErrorErr(err, "failed to save code",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			// don't fail the request, broadcast still happens
		}

		// enrich payload with sender information for cursor tracking
		payload.DisplayName = client.DisplayName
		payload.UserID = client.UserID
		payload.Role = client.Role

		// create new message with updated payload
		broadcastMsg, err := NewMessage(TypeCodeUpdate, client.SessionID, client.UserID, payload)
		if err != nil {
			logger.ErrorErr(err, "failed to create broadcast message",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			return err
		}

		// broadcast to all other clients in the session
		hub.BroadcastToSession(client.SessionID, broadcastMsg, client.ID)

		return nil
	}
}

// uses the ccsignals detector to manage paste locks
func handlePasteDetection(ctx context.Context, hub *Hub, client *Client, detector *ccsignals.Detector, previousCode, newCode string) {
	deltaChars := len(newCode) - len(previousCode)
	logger.Info("paste detection check",
		"session_id", client.SessionID,
		"delta_chars", deltaChars,
		"previous_len", len(previousCode),
		"new_len", len(newCode),
	)

	// check if this is a large delta (potential paste)
	if detector.IsLargeDelta(previousCode, newCode) {
		logger.Info("large delta detected",
			"session_id", client.SessionID,
			"delta_chars", deltaChars,
		)
		// detect paste and get result
		result, err := detector.DetectPaste(ctx, client.SessionID, client.UserID, previousCode, newCode)
		if err != nil {
			logger.ErrorErr(err, "paste detection failed", "session_id", client.SessionID)
			return
		}

		logger.Info("paste detection result",
			"session_id", client.SessionID,
			"should_lock", result.ShouldLock,
			"reason", result.Reason,
		)

		if result.ShouldLock {
			// determine reason for the lock
			reason := "paste_detected"
			if result.FingerprintMatch != nil && !result.FingerprintMatch.Record.CCSignal.AllowsAI() {
				reason = "similar_to_protected"
			} else if result.MatchedContent != nil && result.MatchedContent.CCSignal == ccsignals.SignalNoAI {
				reason = "parent_no_ai"
			}

			// always set/update the lock with new baseline (even if already locked)
			// this ensures baseline is updated when user pastes different content
			config := ccsignals.DefaultConfig()
			if err := detector.SetLock(ctx, client.SessionID, newCode, config.LockTTL); err != nil {
				logger.ErrorErr(err, "failed to set paste lock", "session_id", client.SessionID)
				return
			}

			logger.Info("paste lock set",
				"session_id", client.SessionID,
				"reason", reason,
			)

			// notify client
			sendPasteLockStatus(hub, client, true, reason)
		} else {
			logger.Debug("paste detected but allowed",
				"session_id", client.SessionID,
				"reason", result.Reason,
			)
		}
	} else {
		// no large delta - check if edits are significant enough to unlock
		// first check if there's actually a lock to potentially remove
		wasLocked, err := detector.IsLocked(ctx, client.SessionID)
		if err != nil {
			logger.ErrorErr(err, "failed to check lock status", "session_id", client.SessionID)
			return
		}

		// only process unlock logic if there was a lock
		if wasLocked {
			if err := detector.CheckUnlock(ctx, client.SessionID, newCode); err != nil {
				logger.ErrorErr(err, "failed to check unlock", "session_id", client.SessionID)
				return
			}

			// check if lock was removed
			stillLocked, err := detector.IsLocked(ctx, client.SessionID)
			if err == nil && !stillLocked {
				logger.Info("paste lock removed due to significant edits",
					"session_id", client.SessionID,
				)
				sendPasteLockStatus(hub, client, false, "edits_sufficient")
			}
		}
	}
}

// handles play messages from host/co-author
func PlayHandler() MessageHandler {
	return func(hub *Hub, client *Client, _ *Message) error {
		// check if client has write permissions (host or co-author)
		if !client.CanWrite() {
			client.SendError("forbidden", "only host and co-authors can control playback", "")
			return ErrReadOnly
		}

		// create broadcast payload with display name
		payload := PlayPayload{
			DisplayName: client.DisplayName,
		}

		// create broadcast message
		broadcastMsg, err := NewMessage(TypePlay, client.SessionID, client.UserID, payload)
		if err != nil {
			logger.ErrorErr(err, "failed to create play broadcast message",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			return err
		}

		// broadcast to all other clients in the session (exclude sender)
		hub.BroadcastToSession(client.SessionID, broadcastMsg, client.ID)

		logger.Info("playback started",
			"client_id", client.ID,
			"session_id", client.SessionID,
			"display_name", client.DisplayName,
		)

		return nil
	}
}

// handles stop messages from host/co-author
func StopHandler() MessageHandler {
	return func(hub *Hub, client *Client, _ *Message) error {
		// check if client has write permissions (host or co-author)
		if !client.CanWrite() {
			client.SendError("forbidden", "only host and co-authors can control playback", "")
			return ErrReadOnly
		}

		// create broadcast payload with display name
		payload := StopPayload{
			DisplayName: client.DisplayName,
		}

		// create broadcast message
		broadcastMsg, err := NewMessage(TypeStop, client.SessionID, client.UserID, payload)
		if err != nil {
			logger.ErrorErr(err, "failed to create stop broadcast message",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			return err
		}

		// broadcast to all other clients in the session (exclude sender)
		hub.BroadcastToSession(client.SessionID, broadcastMsg, client.ID)

		logger.Info("playback stopped",
			"client_id", client.ID,
			"session_id", client.SessionID,
			"display_name", client.DisplayName,
		)

		return nil
	}
}

// handles session chat message messages
func ChatHandler(sessionRepo sessions.Repository) MessageHandler {
	return func(hub *Hub, client *Client, msg *Message) error {
		// check rate limit
		if !client.checkChatRateLimit() {
			client.SendError("too_many_requests", "too many chat messages. maximum 20 per minute.", "")
			return ErrRateLimitExceeded
		}

		// parse payload
		var payload ChatMessagePayload

		if err := msg.UnmarshalPayload(&payload); err != nil {
			client.SendError("validation_error", "failed to parse chat message", err.Error())
			return err
		}

		// validate message size
		messageSize := len([]rune(payload.Message))

		if messageSize > maxChatMessageSize {
			client.SendError("bad_request", "message exceeds maximum size. maximum 5000 characters allowed.", "")
			return ErrCodeTooLarge
		}

		// validate message is not empty (after trimming whitespace)
		trimmedMessage := strings.TrimSpace(payload.Message)

		if trimmedMessage == "" {
			client.SendError("bad_request", "message cannot be empty", "")
			return ErrCodeTooLarge
		}

		// save chat message (goes to redis buffer via BufferedRepository)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := sessionRepo.AddChatMessage(ctx, client.SessionID, client.UserID, trimmedMessage, client.DisplayName, "")
		if err != nil {
			logger.ErrorErr(err, "failed to save chat message",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			// don't fail - broadcast is more important for real-time chat
		}

		// add display name to payload
		payload.DisplayName = client.DisplayName
		payload.Message = trimmedMessage

		// create broadcast message
		broadcastMsg, err := NewMessage(TypeChatMessage, client.SessionID, client.UserID, payload)
		if err != nil {
			logger.ErrorErr(err, "failed to create broadcast message",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			return err
		}

		// broadcast to all clients in the session (including sender)
		hub.BroadcastToSession(client.SessionID, broadcastMsg, "")

		return nil
	}
}

// sendPasteLockStatus sends a paste lock status message to the client
func sendPasteLockStatus(_ *Hub, client *Client, locked bool, reason string) {
	payload := PasteLockChangedPayload{
		Locked: locked,
		Reason: reason,
	}

	msg, err := NewMessage(TypePasteLockChanged, client.SessionID, client.UserID, payload)
	if err != nil {
		logger.ErrorErr(err, "failed to create paste lock message",
			"client_id", client.ID,
			"session_id", client.SessionID,
		)
		return
	}

	// send only to the affected client (not broadcast)
	if err := client.Send(msg); err != nil {
		logger.ErrorErr(err, "failed to send paste lock message",
			"client_id", client.ID,
			"session_id", client.SessionID,
		)
	} else {
		logger.Info("paste lock message sent",
			"client_id", client.ID,
			"session_id", client.SessionID,
			"locked", locked,
		)
	}
}

// handles ping messages from clients (keep-alive)
func PingHandler() MessageHandler {
	return func(_ *Hub, client *Client, _ *Message) error {
		// respond with pong
		pongMsg, err := NewMessage(TypePong, client.SessionID, client.UserID, nil)
		if err != nil {
			return err
		}
		client.Send(pongMsg) //nolint:errcheck,gosec // best-effort pong
		return nil
	}
}

// handles cursor position messages for collaboration
func CursorPositionHandler() MessageHandler {
	return func(hub *Hub, client *Client, msg *Message) error {
		// only hosts and co-authors can send cursor positions
		if !client.CanWrite() {
			// silently ignore cursor updates from viewers
			return nil
		}

		// parse payload (client sends line and col only)
		var payload CursorPositionPayload
		if err := msg.UnmarshalPayload(&payload); err != nil {
			// silently ignore malformed cursor updates
			return nil
		}

		// enrich payload with sender information
		payload.UserID = client.UserID
		payload.DisplayName = client.DisplayName
		payload.Role = client.Role

		// create broadcast message
		broadcastMsg, err := NewMessage(TypeCursorPosition, client.SessionID, client.UserID, payload)
		if err != nil {
			logger.ErrorErr(err, "failed to create cursor position broadcast message",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			return err
		}

		// broadcast to all other clients in the session (exclude sender)
		hub.BroadcastToSession(client.SessionID, broadcastMsg, client.ID)

		return nil
	}
}
