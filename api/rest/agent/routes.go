package agent

import (
	"github.com/gin-gonic/gin"

	"codeberg.org/algojams/server/algojams/strudels"
	"codeberg.org/algojams/server/algojams/users"
	agentcore "codeberg.org/algojams/server/internal/agent"
	"codeberg.org/algojams/server/internal/attribution"
	"codeberg.org/algojams/server/internal/buffer"
	"codeberg.org/algojams/server/internal/llm"
)

func RegisterRoutes(router *gin.RouterGroup, agentClient *agentcore.Agent, platformLLM llm.LLM, strudelRepo *strudels.Repository, userRepo *users.Repository, attrService *attribution.Service, sessionBuffer *buffer.SessionBuffer) {
	agentGroup := router.Group("/agent")
	{
		agentGroup.POST("/generate", GenerateHandler(agentClient, platformLLM, strudelRepo, userRepo, attrService, sessionBuffer))
	}
}
