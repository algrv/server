package websocket

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/algrv/server/algorave/sessions"
	"github.com/algrv/server/algorave/strudels"
	"github.com/algrv/server/algorave/users"
	"github.com/algrv/server/internal/agent"
	"github.com/algrv/server/internal/llm"
	"github.com/algrv/server/internal/logger"
)

const dbTimeout = 10 * time.Second

// handles code update messages (broadcast only, no DB write)
func CodeUpdateHandler() MessageHandler {
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

		// track latest code for save on disconnect
		client.SetLastCode(payload.Code)

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
		logger.Info("code updated",
			"client_id", client.ID,
			"session_id", client.SessionID,
			"display_name", client.DisplayName,
		)

		return nil
	}
}

// handles code generation request messages
func GenerateHandler(agentClient *agent.Agent, sessionRepo sessions.Repository, userRepo *users.Repository) MessageHandler {
	return func(hub *Hub, client *Client, msg *Message) error {
		// check if client has write permissions (viewers cannot use agent)
		if !client.CanWrite() {
			client.SendError("forbidden", "only host and co-authors can use the AI assistant", "")
			return ErrReadOnly
		}

		// check per-minute rate limit
		if !client.checkAgentRequestRateLimit() {
			client.SendError("too_many_requests", "too many agent requests. maximum 10 per minute.", "")
			return ErrRateLimitExceeded
		}

		// parse payload
		var payload AgentRequestPayload
		if err := msg.UnmarshalPayload(&payload); err != nil {
			client.SendError("validation_error", "failed to parse generation request", err.Error())
			return err
		}

		isBYOK := payload.ProviderAPIKey != ""
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // longer timeout for AI generation
		defer cancel()

		// check daily rate limit
		if client.IsAuthenticated && client.UserID != "" {
			result, err := userRepo.CheckUserRateLimit(ctx, client.UserID, isBYOK)
			if err != nil {
				logger.ErrorErr(err, "failed to check rate limit",
					"client_id", client.ID,
					"user_id", client.UserID,
				)
				client.SendError("server_error", "failed to check rate limit", "")
				return err
			}

			if !result.Allowed {
				client.SendError("too_many_requests", "daily generation limit exceeded", "")
				return ErrRateLimitExceeded
			}
		} else {
			// anonymous user - check session rate limit
			result, err := userRepo.CheckSessionRateLimit(ctx, client.SessionID)
			if err != nil {
				logger.ErrorErr(err, "failed to check session rate limit",
					"client_id", client.ID,
					"session_id", client.SessionID,
				)
				// continue anyway for anonymous users if rate limit check fails
			} else if !result.Allowed {
				client.SendError("too_many_requests", "daily generation limit exceeded for anonymous users", "")
				return ErrRateLimitExceeded
			}
		}

		// broadcast the user's prompt to writers only (sanitized - no private data)
		broadcastPayload := AgentRequestPayload{
			UserQuery:   payload.UserQuery,
			DisplayName: client.DisplayName,
		}

		broadcastMsg, err := NewMessage(TypeAgentRequest, client.SessionID, client.UserID, broadcastPayload)
		if err == nil {
			hub.BroadcastToWriters(client.SessionID, broadcastMsg, "")
		} else {
			logger.Warn("failed to create broadcast message for agent request",
				"client_id", client.ID,
				"session_id", client.SessionID,
				"error", err,
			)
		}

		// convert conversation history to agent.Message format
		conversationHistory := make([]agent.Message, 0, len(payload.ConversationHistory))
		for _, m := range payload.ConversationHistory {
			conversationHistory = append(conversationHistory, agent.Message{
				Role:    m.Role,
				Content: m.Content,
			})
		}

		// create custom generator if BYOK
		var customGenerator llm.TextGenerator
		if isBYOK {
			if payload.Provider == "" {
				client.SendError("bad_request", "provider is required when using provider_api_key", "")
				return ErrInvalidMessage
			}

			var err error
			customGenerator, err = createBYOKGenerator(payload.Provider, payload.ProviderAPIKey)
			if err != nil {
				client.SendError("bad_request", err.Error(), "")
				return err
			}
		}

		// create agent request
		agentReq := agent.GenerateRequest{
			UserQuery:           payload.UserQuery,
			EditorState:         payload.EditorState,
			ConversationHistory: conversationHistory,
			CustomGenerator:     customGenerator,
		}

		// generate code using agent
		response, err := agentClient.Generate(ctx, agentReq)

		if err != nil {
			logger.ErrorErr(err, "failed to generate code",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			client.SendError("server_error", "failed to generate code", err.Error())
			return err
		}

		// log usage after successful generation
		var userIDPtr *string
		if client.IsAuthenticated && client.UserID != "" {
			userIDPtr = &client.UserID
		}

		usageReq := &users.UsageLogRequest{
			UserID:       userIDPtr,
			SessionID:    client.SessionID,
			Provider:     "anthropic",
			Model:        response.Model,
			InputTokens:  response.InputTokens,
			OutputTokens: response.OutputTokens,
			IsBYOK:       isBYOK,
		}

		if err := userRepo.LogUsage(ctx, usageReq); err != nil {
			logger.ErrorErr(err, "failed to log usage",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			// don't fail the request if logging fails
		}

		// save messages to session history
		if payload.UserQuery != "" {
			_, err := sessionRepo.AddMessage(ctx, client.SessionID, client.UserID, "user", sessions.MessageTypeUserPrompt, payload.UserQuery, false, false, client.DisplayName, "")
			if err != nil {
				logger.ErrorErr(err, "failed to save user message",
					"client_id", client.ID,
					"session_id", client.SessionID,
				)
			}
		}

		if response.Code != "" {
			_, err := sessionRepo.AddMessage(ctx, client.SessionID, "", "assistant", sessions.MessageTypeAIResponse, response.Code, response.IsActionable, response.IsCodeResponse, "Assistant", "")
			if err != nil {
				logger.ErrorErr(err, "failed to save assistant message",
					"client_id", client.ID,
					"session_id", client.SessionID,
				)
			}
		}

		// update session code if generation was successful and is a code response
		if response.IsCodeResponse && response.Code != "" {
			if err := sessionRepo.UpdateSessionCode(ctx, client.SessionID, response.Code); err != nil {
				logger.ErrorErr(err, "failed to update session code",
					"client_id", client.ID,
					"session_id", client.SessionID,
				)
			}
		}

		// create response payload with rate limit info
		responsePayload := AgentResponsePayload{
			Code:                response.Code,
			DocsRetrieved:       response.DocsRetrieved,
			ExamplesRetrieved:   response.ExamplesRetrieved,
			Model:               response.Model,
			IsActionable:        response.IsActionable,
			IsCodeResponse:      response.IsCodeResponse,
			ClarifyingQuestions: response.ClarifyingQuestions,
			RateLimit:           client.GetAgentRateLimitStatus(),
		}

		// create response message
		responseMsg, err := NewMessage(TypeAgentResponse, client.SessionID, client.UserID, responsePayload)
		if err != nil {
			logger.ErrorErr(err, "failed to create response message",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			return err
		}

		// send response to writers only (host and co-authors)
		hub.BroadcastToWriters(client.SessionID, responseMsg, "")

		// update last activity
		if err := sessionRepo.UpdateLastActivity(ctx, client.SessionID); err != nil {
			logger.ErrorErr(err, "failed to update last activity",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
		}

		logger.Info("code generated",
			"client_id", client.ID,
			"session_id", client.SessionID,
			"display_name", client.DisplayName,
			"model", response.Model,
			"docs_retrieved", response.DocsRetrieved,
			"examples_retrieved", response.ExamplesRetrieved,
		)

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

		// save to database
		ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
		defer cancel()

		_, err := sessionRepo.AddMessage(ctx, client.SessionID, client.UserID, "user", sessions.MessageTypeChat, trimmedMessage, false, false, client.DisplayName, "")
		if err != nil {
			logger.ErrorErr(err, "failed to save chat message",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)

			client.SendError("server_error", "failed to save message", err.Error())
			return err
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

		// update last activity
		if err := sessionRepo.UpdateLastActivity(ctx, client.SessionID); err != nil {
			logger.ErrorErr(err, "failed to update last activity",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
		}

		logger.Info("chat message sent",
			"client_id", client.ID,
			"session_id", client.SessionID,
			"display_name", client.DisplayName,
		)

		return nil
	}
}

// handles auto-save messages to persist code to database
func AutoSaveHandler(sessionRepo sessions.Repository) MessageHandler {
	return func(_ *Hub, client *Client, msg *Message) error {
		// check if client has write permissions
		if !client.CanWrite() {
			return ErrReadOnly
		}

		// parse payload
		var payload AutoSavePayload
		if err := msg.UnmarshalPayload(&payload); err != nil {
			client.SendError("validation_error", "failed to parse auto save", err.Error())
			return err
		}

		// validate code size
		codeSize := len([]byte(payload.Code))
		if codeSize > maxCodeSize {
			client.SendError("bad_request", "code exceeds maximum size. maximum 100 KB allowed.", "")
			return ErrCodeTooLarge
		}

		// persist to database
		ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
		defer cancel()

		if err := sessionRepo.UpdateSessionCode(ctx, client.SessionID, payload.Code); err != nil {
			logger.ErrorErr(err, "failed to auto-save session code",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			return err
		}

		logger.Debug("code auto-saved",
			"client_id", client.ID,
			"session_id", client.SessionID,
		)

		return nil
	}
}

// handles switch strudel messages to change context without reconnecting
func SwitchStrudelHandler(strudelRepo *strudels.Repository) MessageHandler {
	return func(_ *Hub, client *Client, msg *Message) error {
		// check if client has write permissions
		if !client.CanWrite() {
			client.SendError("forbidden", "only host and co-authors can switch strudel context", "")
			return ErrReadOnly
		}

		// parse payload
		var payload SwitchStrudelPayload
		if err := msg.UnmarshalPayload(&payload); err != nil {
			client.SendError("validation_error", "failed to parse switch strudel", err.Error())
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
		defer cancel()

		var code string
		var conversationHistory []SessionStateMessage

		if payload.StrudelID != nil {
			// switching to a saved strudel - fetch from DB
			if !client.IsAuthenticated || client.UserID == "" {
				client.SendError("unauthorized", "must be authenticated to load saved strudels", "")
				return ErrUnauthorized
			}

			strudel, err := strudelRepo.Get(ctx, *payload.StrudelID, client.UserID)
			if err != nil {
				logger.ErrorErr(err, "failed to fetch strudel",
					"client_id", client.ID,
					"strudel_id", *payload.StrudelID,
				)
				client.SendError("not_found", "strudel not found or access denied", "")
				return err
			}

			code = strudel.Code

			// convert strudel conversation history to session state format
			conversationHistory = convertAgentMessagesToSessionState(strudel.ConversationHistory)
		} else {
			// scratch/anonymous context - use provided data or empty
			code = payload.Code
			conversationHistory = payload.ConversationHistory

			// validate code size for provided code
			if len([]byte(code)) > maxCodeSize {
				client.SendError("bad_request", "code exceeds maximum size. maximum 100 KB allowed.", "")
				return ErrCodeTooLarge
			}
		}

		// filter conversation history for viewers (they get empty array)
		if client.Role == "viewer" {
			conversationHistory = []SessionStateMessage{}
		}

		// update client's current strudel context
		client.SetCurrentStrudelID(payload.StrudelID)
		client.SetLastCode(code)

		// send session_state with the context (only to requesting client)
		sessionStateMsg, err := NewMessage(TypeSessionState, client.SessionID, client.UserID, SessionStatePayload{
			Code:                code,
			YourRole:            client.Role,
			Participants:        []SessionStateParticipant{}, // not re-sending participants on switch
			ConversationHistory: conversationHistory,
			ChatHistory:         []SessionStateChatMessage{}, // chat is session-scoped, not strudel-scoped
		})
		if err != nil {
			logger.ErrorErr(err, "failed to create session state message",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			client.SendError("server_error", "failed to switch strudel context", "")
			return err
		}

		if err := client.Send(sessionStateMsg); err != nil {
			logger.ErrorErr(err, "failed to send session state",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			return err
		}

		strudelIDStr := "scratch"
		if payload.StrudelID != nil {
			strudelIDStr = *payload.StrudelID
		}

		logger.Info("strudel context switched",
			"client_id", client.ID,
			"session_id", client.SessionID,
			"strudel_id", strudelIDStr,
		)

		return nil
	}
}

// converts agent.Message slice to SessionStateMessage slice
func convertAgentMessagesToSessionState(messages []agent.Message) []SessionStateMessage {
	result := make([]SessionStateMessage, 0, len(messages))
	for i, m := range messages {
		displayName := "User"
		isCodeResponse := false
		if m.Role == "assistant" {
			displayName = "Assistant"
			isCodeResponse = true // assume assistant messages are code responses
		}

		result = append(result, SessionStateMessage{
			ID:             fmt.Sprintf("strudel-msg-%d", i),
			Role:           m.Role,
			Content:        m.Content,
			IsCodeResponse: isCodeResponse,
			DisplayName:    displayName,
			Timestamp:      0, // not stored in strudel
		})
	}
	return result
}
