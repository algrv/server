package generate

import (
	"github.com/algorave/server/internal/agent"
	"github.com/algorave/server/internal/auth"
	"github.com/algorave/server/internal/sessions"
	"github.com/gin-gonic/gin"
)

// registers code generation routes
func RegisterRoutes(router *gin.RouterGroup, agentClient *agent.Agent, strudelRepo StrudelGetter, sessionMgr *sessions.Manager) {
	router.POST("/generate", auth.OptionalAuthMiddleware(), Handler(agentClient, strudelRepo, sessionMgr))
}
