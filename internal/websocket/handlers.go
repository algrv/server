package websocket

import (
	"context"
	"strings"
	"time"

	"codeberg.org/algorave/server/algorave/sessions"
	"codeberg.org/algorave/server/internal/ccsignals"
	"codeberg.org/algorave/server/internal/logger"
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
		if detector != nil {
			handlePasteDetection(ctx, hub, client, detector, previousCode, payload.Code)
		}

		// save code (goes to redis buffer via BufferedRepository)
		if err := sessionRepo.UpdateSessionCode(ctx, client.SessionID, payload.Code); err != nil {
			logger.ErrorErr(err, "failed to save code",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			// don't fail the request, broadcast still happens
		}

		// add display name to payload
		payload.DisplayName = client.DisplayName

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
	// check if this is a large delta (potential paste)
	if detector.IsLargeDelta(previousCode, newCode) {
		// detect paste and get result
		result, err := detector.DetectPaste(ctx, client.SessionID, client.UserID, previousCode, newCode)
		if err != nil {
			logger.ErrorErr(err, "paste detection failed", "session_id", client.SessionID)
			return
		}

		if result.ShouldLock {
			// check if already locked (preserve original baseline)
			alreadyLocked, err := detector.IsLocked(ctx, client.SessionID)
			if err != nil {
				logger.ErrorErr(err, "failed to check lock status", "session_id", client.SessionID)
				// continue assuming not locked (will set new lock)
			}

			if !alreadyLocked {
				// set the lock
				config := ccsignals.DefaultConfig()
				if err := detector.SetLock(ctx, client.SessionID, newCode, config.LockTTL); err != nil {
					logger.ErrorErr(err, "failed to set paste lock", "session_id", client.SessionID)
					return
				}

				// determine reason for logging and notification
				reason := "paste_detected"

				if result.FingerprintMatch != nil && !result.FingerprintMatch.Record.CCSignal.AllowsAI() {
					reason = "similar_to_protected"
					logger.Info("paste lock set (similar to protected content)",
						"session_id", client.SessionID,
						"matched_work_id", result.FingerprintMatch.Record.WorkID,
						"similarity_distance", result.FingerprintMatch.Distance,
					)
				} else if result.MatchedContent != nil && result.MatchedContent.CCSignal == ccsignals.SignalNoAI {
					reason = "parent_no_ai"
					logger.Info("paste lock set (parent has no-ai)",
						"session_id", client.SessionID,
					)
				} else {
					logger.Info("paste lock set (external paste)",
						"session_id", client.SessionID,
						"reason", result.Reason,
					)
				}

				// notify client
				sendPasteLockStatus(hub, client, true, reason)
			}
		} else {
			logger.Debug("paste detected but allowed",
				"session_id", client.SessionID,
				"reason", result.Reason,
			)
		}
	} else {
		// no large delta - check if edits are significant enough to unlock
		if err := detector.CheckUnlock(ctx, client.SessionID, newCode); err != nil {
			logger.ErrorErr(err, "failed to check unlock", "session_id", client.SessionID)
			return
		}

		// check if lock was removed
		locked, err := detector.IsLocked(ctx, client.SessionID)
		if err == nil && !locked {
			// lock was removed by CheckUnlock, notify client
			// note: we only send this if lock was previously active
			// CheckUnlock handles the state internally, we just notify
			// For now, we check by seeing if significant edit threshold was met
			if detector.IsSignificantEdit(previousCode, newCode) {
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
