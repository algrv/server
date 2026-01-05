package main

import (
	"github.com/algrv/server/api/rest/admin"
	"github.com/algrv/server/api/rest/agent"
	"github.com/algrv/server/api/rest/auth"
	"github.com/algrv/server/api/rest/collaboration"
	"github.com/algrv/server/api/rest/health"
	"github.com/algrv/server/api/rest/strudels"
	"github.com/algrv/server/api/rest/users"
	"github.com/algrv/server/api/websocket"
	"github.com/gin-gonic/gin"
)

// sets up all API routes and middleware
func RegisterRoutes(router *gin.Engine, server *Server) {
	router.Use(CORSMiddleware())
	router.GET("/health", health.Handler)

	v1 := router.Group("/api/v1")

	{
		v1.GET("/ping", health.PingHandler)

		auth.RegisterRoutes(v1, server.userRepo)
		strudels.RegisterRoutes(v1, server.strudelRepo)
		collaboration.RegisterRoutes(v1, server.sessionRepo, server.hub)
		users.RegisterRoutes(v1, server.db)
		admin.RegisterRoutes(v1, server.strudelRepo)
		agent.RegisterRoutes(v1, server.services.Agent, server.services.LLM)
		websocket.RegisterRoutes(v1, server.hub, server.sessionRepo)
	}
}
