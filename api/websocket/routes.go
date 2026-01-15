package websocket

import (
	"github.com/gin-gonic/gin"

	"codeberg.org/algorave/server/algorave/sessions"
	ws "codeberg.org/algorave/server/internal/websocket"
)

func RegisterRoutes(router *gin.RouterGroup, hub *ws.Hub, sessionRepo sessions.Repository) {
	router.GET("/ws", WebSocketHandler(hub, sessionRepo))
}
