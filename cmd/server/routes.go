package main

import (
	"codeberg.org/algopatterns/server/api/rest/admin"
	"codeberg.org/algopatterns/server/api/rest/agent"
	"codeberg.org/algopatterns/server/api/rest/auth"
	"codeberg.org/algopatterns/server/api/rest/collaboration"
	"codeberg.org/algopatterns/server/api/rest/health"
	"codeberg.org/algopatterns/server/api/rest/strudels"
	"codeberg.org/algopatterns/server/api/rest/users"
	"codeberg.org/algopatterns/server/api/websocket"
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
		agent.RegisterRoutes(v1, server.services.Agent, server.services.LLM, server.strudelRepo, server.userRepo, server.services.Attribution, server.buffer)
		websocket.RegisterRoutes(v1, server.hub, server.sessionRepo, server.userRepo)
	}
}
