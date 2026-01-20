package websocket

import (
	"github.com/gin-gonic/gin"

	"codeberg.org/algojams/server/algojams/sessions"
	"codeberg.org/algojams/server/algojams/users"
	ws "codeberg.org/algojams/server/internal/websocket"
)

func RegisterRoutes(router *gin.RouterGroup, hub *ws.Hub, sessionRepo sessions.Repository, userRepo *users.Repository) {
	router.GET("/ws", WebSocketHandler(hub, sessionRepo, userRepo))
}
