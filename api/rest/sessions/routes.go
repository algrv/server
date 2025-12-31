package sessions

import (
	"github.com/algoraveai/server/algorave/sessions"
	"github.com/algoraveai/server/algorave/strudels"
	"github.com/algoraveai/server/internal/auth"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, sessionRepo sessions.Repository, strudelRepo *strudels.Repository) {
	router.POST("/sessions/transfer", auth.AuthMiddleware(), TransferSessionHandler(sessionRepo, strudelRepo))
}
