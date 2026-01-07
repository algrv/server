package websocket

import (
	"context"
	"strings"
	"time"

	"github.com/algrv/server/algorave/sessions"
	"github.com/algrv/server/algorave/strudels"
	"github.com/algrv/server/internal/buffer"
	"github.com/algrv/server/internal/logger"
)

// handles code update messages
func CodeUpdateHandler(sessionRepo sessions.Repository, sessionBuffer *buffer.SessionBuffer, strudelRepo *strudels.Repository) MessageHandler {
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

		// paste detection: server-side behavioral detection (independent of frontend source field)
		// only process paste detection if there's a large delta
		if buffer.IsLargeDelta(previousCode, payload.Code) {
			// large delta detected - validate if it's from a legitimate source
			shouldLock := true

			// check 1: does code match session's existing code? (reconnection/sync)
			if payload.Code == previousCode {
				shouldLock = false
			}

			// check 2: does code match any of user's own strudels? (loading own work)
			if shouldLock && client.UserID != "" && strudelRepo != nil {
				owns, err := strudelRepo.UserOwnsStrudelWithCode(ctx, client.UserID, payload.Code)
				if err == nil && owns {
					shouldLock = false
					logger.Info("large delta from own strudel, skipping paste lock",
						"session_id", client.SessionID,
						"user_id", client.UserID,
					)
				}
			}

			// check 3: does code match any public strudel that allows AI? (legitimate fork)
			// note: public strudels with no-ai CC signal will NOT bypass the lock
			if shouldLock && strudelRepo != nil {
				exists, err := strudelRepo.PublicStrudelExistsWithCodeAllowsAI(ctx, payload.Code)
				if err == nil && exists {
					shouldLock = false
					logger.Info("large delta from public strudel (fork, allows AI), skipping paste lock",
						"session_id", client.SessionID,
					)
				}
			}

			// if still shouldLock, this is likely an external paste
			if shouldLock {
				// only set lock if not already locked - preserve original baseline
				// (prevents bypass via duplicate-then-remove)
				alreadyLocked, _ := sessionBuffer.IsPasteLocked(ctx, client.SessionID) //nolint:errcheck // best-effort
				if !alreadyLocked {
					if err := sessionBuffer.SetPasteLock(ctx, client.SessionID, payload.Code); err != nil {
						logger.ErrorErr(err, "failed to set paste lock", "session_id", client.SessionID)
					} else {
						logger.Info("paste lock set",
							"session_id", client.SessionID,
							"source", payload.Source,
							"delta_len", len(payload.Code)-len(previousCode),
						)
						// notify client that paste lock is active
						sendPasteLockStatus(hub, client, true, "paste_detected")
					}
				}
			}
		} else {
			// no large delta - check if session is locked and if edits are significant enough to unlock
			locked, err := sessionBuffer.IsPasteLocked(ctx, client.SessionID)
			if err == nil && locked {
				baseline, err := sessionBuffer.GetPasteBaseline(ctx, client.SessionID)
				if err == nil && buffer.IsSignificantEdit(baseline, payload.Code) {
					// significant edits detected, remove lock
					if err := sessionBuffer.RemovePasteLock(ctx, client.SessionID); err != nil {
						logger.ErrorErr(err, "failed to remove paste lock", "session_id", client.SessionID)
					} else {
						logger.Info("paste lock removed due to significant edits",
							"session_id", client.SessionID,
						)
						// notify client that paste lock is lifted
						sendPasteLockStatus(hub, client, false, "edits_sufficient")
					}
				} else {
					// refresh TTL while still locked
					sessionBuffer.RefreshPasteLockTTL(ctx, client.SessionID) //nolint:errcheck,gosec // best-effort
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
