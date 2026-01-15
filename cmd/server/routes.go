package main

import (
	"codeberg.org/algorave/server/api/rest/admin"
	"codeberg.org/algorave/server/api/rest/agent"
	"codeberg.org/algorave/server/api/rest/auth"
	"codeberg.org/algorave/server/api/rest/collaboration"
	"codeberg.org/algorave/server/api/rest/health"
	"codeberg.org/algorave/server/api/rest/strudels"
	"codeberg.org/algorave/server/api/rest/users"
	"codeberg.org/algorave/server/api/websocket"
	"github.com/gin-gonic/gin"
)

// sets up all API routes and middleware
func RegisterRoutes(router *gin.Engine, server *Server) {
	router.Use(CORSMiddleware())

	// bot defense middleware - runs after CORS, before other routes
	if server.botDefense != nil {
		router.Use(server.botDefense.Middleware())
	}

	router.GET("/health", health.Handler)

	v1 := router.Group("/api/v1")

	{
		v1.GET("/ping", health.PingHandler)

		auth.RegisterRoutes(v1, server.userRepo)
		strudels.RegisterRoutes(v1, server.strudelRepo, server.services.Attribution, server.ccSignals)
		collaboration.RegisterRoutes(v1, server.sessionRepo, server.hub)
		users.RegisterRoutes(v1, server.db)
		admin.RegisterRoutes(v1, server.strudelRepo)
		agent.RegisterRoutes(v1, server.services.Agent, server.services.LLM, server.strudelRepo, server.services.Attribution, server.buffer)
		websocket.RegisterRoutes(v1, server.hub, server.sessionRepo)
	}
}
