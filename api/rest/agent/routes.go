package agent

import (
	"github.com/gin-gonic/gin"

	"codeberg.org/algorave/server/algorave/strudels"
	"codeberg.org/algorave/server/algorave/users"
	agentcore "codeberg.org/algorave/server/internal/agent"
	"codeberg.org/algorave/server/internal/attribution"
	"codeberg.org/algorave/server/internal/buffer"
	"codeberg.org/algorave/server/internal/llm"
)

func RegisterRoutes(router *gin.RouterGroup, agentClient *agentcore.Agent, platformLLM llm.LLM, strudelRepo *strudels.Repository, userRepo *users.Repository, attrService *attribution.Service, sessionBuffer *buffer.SessionBuffer) {
	agentGroup := router.Group("/agent")
	{
		agentGroup.POST("/generate", GenerateHandler(agentClient, platformLLM, strudelRepo, userRepo, attrService, sessionBuffer))
	}
}
