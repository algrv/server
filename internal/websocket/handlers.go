package websocket

import (
	"context"
	"strings"
	"time"

	"github.com/algrv/server/algorave/sessions"
	"github.com/algrv/server/internal/logger"
)

// handles code update messages
func CodeUpdateHandler(sessionRepo sessions.Repository) MessageHandler {
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

		// save code (goes to redis buffer via BufferedRepository)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

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
