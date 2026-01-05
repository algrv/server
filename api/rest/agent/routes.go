package agent

import (
	"github.com/gin-gonic/gin"

	agentcore "github.com/algrv/server/internal/agent"
	"github.com/algrv/server/internal/llm"
)

func RegisterRoutes(router *gin.RouterGroup, agentClient *agentcore.Agent, platformLLM llm.LLM) {
	agentGroup := router.Group("/agent")
	{
		agentGroup.POST("/generate", GenerateHandler(agentClient, platformLLM))
	}
}
