package sessions

import (
	"net/http"

	"github.com/algorave/server/algorave/strudels"
	"github.com/algorave/server/internal/auth"
	"github.com/algorave/server/internal/sessions"
	"github.com/gin-gonic/gin"
)

// creates a handler to transfer an anonymous session to an authenticated user's account
func TransferSessionHandler(sessionMgr *sessions.Manager, strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// check if user is authenticated
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		var req TransferSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// get the session
		session, exists := sessionMgr.GetSession(req.SessionID)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found or expired"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create strudel"})
			return
		}

		// delete the session after successful transfer
		sessionMgr.DeleteSession(req.SessionID)

		c.JSON(http.StatusCreated, gin.H{
			"message":    "session transferred successfully",
			"strudel":    strudel,
			"strudel_id": strudel.ID,
		})
	}
}
