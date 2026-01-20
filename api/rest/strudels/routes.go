package strudels

import (
	"codeberg.org/algojams/server/algojams/strudels"
	"codeberg.org/algojams/server/internal/attribution"
	"codeberg.org/algojams/server/internal/auth"
	"codeberg.org/algojams/server/internal/ccsignals"
	"github.com/gin-gonic/gin"
)

// provides methods to index/remove strudels from the fingerprint index
type FingerprintIndexer interface {
	IndexStrudel(strudelID, creatorID, code string, ccSignal ccsignals.CCSignal)
	UpdateStrudel(strudelID, creatorID, code string, ccSignal ccsignals.CCSignal) // only rehashes if content changed
	RemoveStrudel(strudelID string)
}

func RegisterRoutes(router *gin.RouterGroup, strudelRepo *strudels.Repository, attrService *attribution.Service, fpIndexer FingerprintIndexer) {
	// GET strudel by ID - allows owner OR public access (optional auth)
	router.GET("/strudels/:id", auth.OptionalAuthMiddleware(), GetStrudelHandler(strudelRepo))

	// authenticated strudel operations
	strudelsGroup := router.Group("/strudels")
	strudelsGroup.Use(auth.AuthMiddleware())
	{
		strudelsGroup.GET("", ListStrudelsHandler(strudelRepo))
		strudelsGroup.POST("", CreateStrudelHandler(strudelRepo, fpIndexer))
		strudelsGroup.GET("/tags", ListUserTagsHandler(strudelRepo))
		strudelsGroup.PUT("/:id", UpdateStrudelHandler(strudelRepo, fpIndexer))
		strudelsGroup.DELETE("/:id", DeleteStrudelHandler(strudelRepo, fpIndexer))
	}

	// public strudels (no auth required)
	router.GET("/public/strudels", ListPublicStrudelsHandler(strudelRepo))
	router.GET("/public/strudels/tags", ListPublicTagsHandler(strudelRepo))
	router.GET("/public/strudels/:id", GetPublicStrudelHandler(strudelRepo))
	router.GET("/public/strudels/:id/stats", GetStrudelStatsHandler(strudelRepo, attrService))
}
