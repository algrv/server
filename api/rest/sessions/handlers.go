package sessions

import (
	"net/http"

	"github.com/algoraveai/server/algorave/sessions"
	"github.com/algoraveai/server/algorave/strudels"
	"github.com/algoraveai/server/internal/agent"
	"github.com/algoraveai/server/internal/auth"
	"github.com/algoraveai/server/internal/errors"
	"github.com/gin-gonic/gin"
)

// TransferSessionHandler godoc
// @Summary Transfer anonymous session
// @Description Convert an anonymous session to a saved strudel in authenticated user's account
// @Tags sessions
// @Accept json
// @Produce json
// @Param request body TransferSessionRequest true "Transfer request"
// @Success 201 {object} TransferSessionResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/transfer [post]
// @Security BearerAuth
func TransferSessionHandler(sessionRepo sessions.Repository, strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		var req TransferSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		ctx := c.Request.Context()

		// get the session from DB
		session, err := sessionRepo.GetSession(ctx, req.SessionID)
		if err != nil {
			errors.NotFound(c, "session")
			return
		}

		// verify it's an anonymous session (owned by system user)
		if session.HostUserID != sessions.SystemUserID {
			errors.Forbidden(c, "only anonymous sessions can be transferred")
			return
		}

		// get messages from the session to build conversation history
		messages, err := sessionRepo.GetMessages(ctx, req.SessionID, 100)
		if err != nil {
			errors.InternalError(c, "failed to get session messages", err)
			return
		}

		// convert session messages to conversation history format
		history := make(strudels.ConversationHistory, 0, len(messages))
		for i := len(messages) - 1; i >= 0; i-- { // reverse order (oldest first)
			msg := messages[i]
			// only include user prompts and AI responses, not chat messages
			if msg.MessageType == sessions.MessageTypeUserPrompt || msg.MessageType == sessions.MessageTypeAIResponse {
				role := "user"
				if msg.MessageType == sessions.MessageTypeAIResponse {
					role = "assistant"
				}
				history = append(history, agent.Message{
					Role:    role,
					Content: msg.Content,
				})
			}
		}

		// create a new strudel with the session's conversation history
		strudelReq := strudels.CreateStrudelRequest{
			Title:               req.Title,
			Description:         "Transferred from anonymous session",
			Code:                session.Code,
			ConversationHistory: history,
			IsPublic:            false,
		}

		strudel, err := strudelRepo.Create(ctx, userID, strudelReq)
		if err != nil {
			errors.InternalError(c, "failed to create strudel", err)
			return
		}

		// end the session after successful transfer
		if err := sessionRepo.EndSession(ctx, req.SessionID); err != nil {
			// log but don't fail - the strudel was already created
			_ = err
		}

		c.JSON(http.StatusCreated, TransferSessionResponse{
			Message:   "session transferred successfully",
			Strudel:   strudel,
			StrudelID: strudel.ID,
		})
	}
}
