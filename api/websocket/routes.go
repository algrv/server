package websocket

import (
	"github.com/gin-gonic/gin"

	"github.com/algoraveai/server/algorave/sessions"
	"github.com/algoraveai/server/algorave/users"
	ws "github.com/algoraveai/server/internal/websocket"
)

func RegisterRoutes(router *gin.RouterGroup, hub *ws.Hub, sessionRepo sessions.Repository, userRepo *users.Repository) {
	router.GET("/ws", WebSocketHandler(hub, sessionRepo, userRepo))
}
