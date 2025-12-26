package generate

import (
	"log"
	"net/http"

	"github.com/algorave/server/internal/agent"
	"github.com/gin-gonic/gin"
)

// Handler creates a handler for code generation
func Handler(agentClient *agent.Agent) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Request

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "validation failed",
				"details": err.Error(),
			})
			return
		}

		resp, err := agentClient.Generate(c.Request.Context(), agent.GenerateRequest{
			UserQuery:           req.UserQuery,
			EditorState:         req.EditorState,
			ConversationHistory: req.ConversationHistory,
		})

		if err != nil {
			log.Printf("failed to generate code: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "generation_failed",
				"message": "failed to generate code",
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Code:                resp.Code,
			DocsRetrieved:       resp.DocsRetrieved,
			ExamplesRetrieved:   resp.ExamplesRetrieved,
			Model:               resp.Model,
			IsActionable:        resp.IsActionable,
			ClarifyingQuestions: resp.ClarifyingQuestions,
		})
	}
}
