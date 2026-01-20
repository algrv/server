package agent

import (
	"github.com/gin-gonic/gin"

	"codeberg.org/algopatterns/server/algopatterns/strudels"
	"codeberg.org/algopatterns/server/algopatterns/users"
	agentcore "codeberg.org/algopatterns/server/internal/agent"
	"codeberg.org/algopatterns/server/internal/attribution"
	"codeberg.org/algopatterns/server/internal/buffer"
	"codeberg.org/algopatterns/server/internal/llm"
)

func RegisterRoutes(router *gin.RouterGroup, agentClient *agentcore.Agent, platformLLM llm.LLM, strudelRepo *strudels.Repository, userRepo *users.Repository, attrService *attribution.Service, sessionBuffer *buffer.SessionBuffer) {
	agentGroup := router.Group("/agent")
	{
		agentGroup.POST("/generate", GenerateHandler(agentClient, platformLLM, strudelRepo, userRepo, attrService, sessionBuffer))
	}
}
