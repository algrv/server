package generate

import (
	"github.com/algorave/server/internal/agent"
	"github.com/algorave/server/internal/auth"
	"github.com/gin-gonic/gin"
)

// registers code generation routes
func RegisterRoutes(router *gin.RouterGroup, agentClient *agent.Agent) {
	router.POST("/generate", auth.OptionalAuthMiddleware(), Handler(agentClient))
}
