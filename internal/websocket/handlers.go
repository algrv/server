package websocket

import (
	"context"

	"github.com/algorave/server/algorave/sessions"
	"github.com/algorave/server/internal/agent"
	"github.com/algorave/server/internal/logger"
)

// handles code update messages
func CodeUpdateHandler(sessionRepo sessions.Repository) MessageHandler {
	return func(hub *Hub, client *Client, msg *Message) error {
		// check rate limit
		if !client.checkCodeUpdateRateLimit() {
			client.SendError("RATE_LIMIT_EXCEEDED", "Too many code updates. Maximum 10 per second.", "")
			return ErrRateLimitExceeded
		}

		// check if client has write permissions
		if !client.CanWrite() {
			client.SendError("FORBIDDEN", "You don't have permission to edit code", "")
			return ErrReadOnly
		}

		// parse payload
		var payload CodeUpdatePayload
		if err := msg.UnmarshalPayload(&payload); err != nil {
			client.SendError("INVALID_PAYLOAD", "failed to parse code update", err.Error())
			return err
		}

		// validate code size
		codeSize := len([]byte(payload.Code))
		if codeSize > maxCodeSize {
			client.SendError("CODE_TOO_LARGE", "Code exceeds maximum size. Maximum 100 KB allowed.", "")
			return ErrCodeTooLarge
		}

		// update session code in database
		ctx := context.Background()
		if err := sessionRepo.UpdateSessionCode(ctx, client.SessionID, payload.Code); err != nil {
			logger.ErrorErr(err, "failed to update session code",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			client.SendError("DATABASE_ERROR", "Failed to save code update", err.Error())
			return err
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
		logger.Info("code updated",
			"client_id", client.ID,
			"session_id", client.SessionID,
			"display_name", client.DisplayName,
		)

		return nil
	}
}

// handles code generation request messages
func GenerateHandler(agentClient *agent.Agent, sessionRepo sessions.Repository) MessageHandler {
	return func(hub *Hub, client *Client, msg *Message) error {
		// check rate limit
		if !client.checkAgentRequestRateLimit() {
			client.SendError("RATE_LIMIT_EXCEEDED", "Too many agent requests. Maximum 5 per minute.", "")
			return ErrRateLimitExceeded
		}

		// parse payload
		var payload AgentRequestPayload
		if err := msg.UnmarshalPayload(&payload); err != nil {
			client.SendError("INVALID_PAYLOAD", "failed to parse generation request", err.Error())
			return err
		}

		// convert conversation history to agent.Message format
		conversationHistory := make([]agent.Message, 0, len(payload.ConversationHistory))
		for _, msg := range payload.ConversationHistory {
			conversationHistory = append(conversationHistory, agent.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		// create agent request
		agentReq := agent.GenerateRequest{
			UserQuery:           payload.UserQuery,
			EditorState:         payload.EditorState,
			ConversationHistory: conversationHistory,
		}

		// generate code using agent
		ctx := context.Background()
		response, err := agentClient.Generate(ctx, agentReq)
		if err != nil {
			logger.ErrorErr(err, "failed to generate code",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
			client.SendError("GENERATION_ERROR", "Failed to generate code", err.Error())
			return err
		}

		// save messages to session history
		if payload.UserQuery != "" {
			_, err := sessionRepo.AddMessage(ctx, client.SessionID, client.UserID, "user", payload.UserQuery)
			if err != nil {
				logger.ErrorErr(err, "failed to save user message",
					"client_id", client.ID,
					"session_id", client.SessionID,
				)
			}
		}

		if response.Code != "" {
			_, err := sessionRepo.AddMessage(ctx, client.SessionID, "", "assistant", response.Code)
			if err != nil {
				logger.ErrorErr(err, "failed to save assistant message",
					"client_id", client.ID,
					"session_id", client.SessionID,
				)
			}
		}

		// update session code if generation was successful
		if response.IsActionable && response.Code != "" {
			if err := sessionRepo.UpdateSessionCode(ctx, client.SessionID, response.Code); err != nil {
				logger.ErrorErr(err, "failed to update session code",
					"client_id", client.ID,
					"session_id", client.SessionID,
				)
			}
		}

		// create response payload
		responsePayload := AgentResponsePayload{
			Code:                response.Code,
			DocsRetrieved:       response.DocsRetrieved,
			ExamplesRetrieved:   response.ExamplesRetrieved,
			Model:               response.Model,
			IsActionable:        response.IsActionable,
			ClarifyingQuestions: response.ClarifyingQuestions,
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

		// send response to all clients in the session (including requester)
		hub.BroadcastToSession(client.SessionID, responseMsg, "")

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
