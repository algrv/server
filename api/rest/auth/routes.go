package auth

import (
	"github.com/algorave/server/algorave/users"
	"github.com/algorave/server/internal/auth"
	"github.com/gin-gonic/gin"
)

// registers all authentication routes
func RegisterRoutes(router *gin.RouterGroup, userRepo *users.Repository) {
	authGroup := router.Group("/auth")
	{
		authGroup.GET("/:provider", BeginAuthHandler(userRepo))
		authGroup.GET("/:provider/callback", CallbackHandler(userRepo))
		authGroup.POST("/logout", LogoutHandler())
		authGroup.GET("/me", auth.AuthMiddleware(), GetCurrentUserHandler(userRepo))
	}
}
