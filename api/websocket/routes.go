package websocket

import (
	"github.com/gin-gonic/gin"

	"github.com/algrv/server/algorave/sessions"
	"github.com/algrv/server/algorave/users"
	ws "github.com/algrv/server/internal/websocket"
)

func RegisterRoutes(router *gin.RouterGroup, hub *ws.Hub, sessionRepo sessions.Repository, userRepo *users.Repository) {
	router.GET("/ws", WebSocketHandler(hub, sessionRepo, userRepo))
}
