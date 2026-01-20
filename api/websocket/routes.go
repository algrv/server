package websocket

import (
	"github.com/gin-gonic/gin"

	"codeberg.org/algopatterns/server/algopatterns/sessions"
	"codeberg.org/algopatterns/server/algopatterns/users"
	ws "codeberg.org/algopatterns/server/internal/websocket"
)

func RegisterRoutes(router *gin.RouterGroup, hub *ws.Hub, sessionRepo sessions.Repository, userRepo *users.Repository) {
	router.GET("/ws", WebSocketHandler(hub, sessionRepo, userRepo))
}
