package sessions

import (
	"net/http"

	"github.com/algorave/server/algorave/anonsessions"
	"github.com/algorave/server/algorave/strudels"
	"github.com/algorave/server/internal/auth"
	"github.com/algorave/server/internal/errors"
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
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/sessions/transfer [post]
// @Security BearerAuth
func TransferSessionHandler(sessionMgr *anonsessions.Manager, strudelRepo *strudels.Repository) gin.HandlerFunc {
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

		session, exists := sessionMgr.GetSession(req.SessionID)
		if !exists {
			errors.NotFound(c, "session")
			return
		}

		// create a new strudel with the session's conversation history
		strudelReq := strudels.CreateStrudelRequest{
			Title:               req.Title,
			Description:         "Transferred from anonymous session",
			Code:                session.EditorState,
			ConversationHistory: session.ConversationHistory,
			IsPublic:            false,
		}

		strudel, err := strudelRepo.Create(c.Request.Context(), userID, strudelReq)
		if err != nil {
			errors.InternalError(c, "failed to create strudel", err)
			return
		}

		// delete the session after successful transfer
		sessionMgr.DeleteSession(req.SessionID)

		c.JSON(http.StatusCreated, TransferSessionResponse{
			Message:   "session transferred successfully",
			Strudel:   strudel,
			StrudelID: strudel.ID,
		})
	}
}
