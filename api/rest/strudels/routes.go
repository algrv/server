package strudels

import (
	"github.com/algrv/server/algorave/strudels"
	"github.com/algrv/server/internal/attribution"
	"github.com/algrv/server/internal/auth"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, strudelRepo *strudels.Repository, attrService *attribution.Service) {
	strudelsGroup := router.Group("/strudels")
	strudelsGroup.Use(auth.AuthMiddleware())
	{
		strudelsGroup.GET("", ListStrudelsHandler(strudelRepo))
		strudelsGroup.POST("", CreateStrudelHandler(strudelRepo))
		strudelsGroup.GET("/tags", ListUserTagsHandler(strudelRepo))
		strudelsGroup.GET("/:id", GetStrudelHandler(strudelRepo))
		strudelsGroup.PUT("/:id", UpdateStrudelHandler(strudelRepo))
		strudelsGroup.DELETE("/:id", DeleteStrudelHandler(strudelRepo))
	}

	// public strudels (no auth required)
	router.GET("/public/strudels", ListPublicStrudelsHandler(strudelRepo))
	router.GET("/public/strudels/tags", ListPublicTagsHandler(strudelRepo))
	router.GET("/public/strudels/:id", GetPublicStrudelHandler(strudelRepo))
	router.GET("/public/strudels/:id/stats", GetStrudelStatsHandler(strudelRepo, attrService))
}
