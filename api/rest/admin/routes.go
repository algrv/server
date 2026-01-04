package admin

import (
	"github.com/algrv/server/algorave/strudels"
	"github.com/algrv/server/internal/auth"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, strudelRepo *strudels.Repository) {
	admin := router.Group("/admin")
	admin.Use(auth.AdminAuthMiddleware())

	admin.GET("/strudels/:id", GetStrudel(strudelRepo))
	admin.PUT("/strudels/:id/use-in-training", SetUseInTraining(strudelRepo))
}
