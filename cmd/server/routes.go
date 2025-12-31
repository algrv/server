package main

import (
	"github.com/algoraveai/server/api/rest/auth"
	"github.com/algoraveai/server/api/rest/collaboration"
	"github.com/algoraveai/server/api/rest/health"
	restSessions "github.com/algoraveai/server/api/rest/sessions"
	"github.com/algoraveai/server/api/rest/strudels"
	"github.com/algoraveai/server/api/rest/users"
	"github.com/algoraveai/server/api/websocket"
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
		restSessions.RegisterRoutes(v1, server.sessionRepo, server.strudelRepo)
		collaboration.RegisterRoutes(v1, server.sessionRepo)
		users.RegisterRoutes(v1, server.db)
		websocket.RegisterRoutes(v1, server.hub, server.sessionRepo, server.userRepo)
	}
}
