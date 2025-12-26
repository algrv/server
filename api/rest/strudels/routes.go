package strudels

import (
	"github.com/algorave/server/algorave/strudels"
	"github.com/algorave/server/internal/auth"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all strudel routes
func RegisterRoutes(router *gin.RouterGroup, strudelRepo *strudels.Repository) {
	// Protected strudel routes (require authentication)
	strudelsGroup := router.Group("/strudels")
	strudelsGroup.Use(auth.AuthMiddleware())
	{
		strudelsGroup.GET("", ListStrudelsHandler(strudelRepo))
		strudelsGroup.POST("", CreateStrudelHandler(strudelRepo))
		strudelsGroup.GET("/:id", GetStrudelHandler(strudelRepo))
		strudelsGroup.PUT("/:id", UpdateStrudelHandler(strudelRepo))
		strudelsGroup.DELETE("/:id", DeleteStrudelHandler(strudelRepo))
	}

	// Public strudels (no auth required)
	router.GET("/public/strudels", ListPublicStrudelsHandler(strudelRepo))
}
