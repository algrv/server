package sessions

import (
	"github.com/algorave/server/algorave/strudels"
	"github.com/algorave/server/internal/auth"
	"github.com/algorave/server/internal/sessions"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, sessionMgr *sessions.Manager, strudelRepo *strudels.Repository) {
	router.POST("/sessions/transfer", auth.AuthMiddleware(), TransferSessionHandler(sessionMgr, strudelRepo))
}
