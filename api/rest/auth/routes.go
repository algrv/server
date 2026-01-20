package auth

import (
	"codeberg.org/algopatterns/server/algopatterns/users"
	"codeberg.org/algopatterns/server/internal/auth"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, userRepo *users.Repository) {
	authGroup := router.Group("/auth")
	{
		authGroup.GET("/:provider", BeginAuthHandler(userRepo))
		authGroup.GET("/:provider/callback", CallbackHandler(userRepo))
		authGroup.POST("/logout", LogoutHandler())
		authGroup.GET("/me", auth.AuthMiddleware(), GetCurrentUserHandler(userRepo))
		authGroup.PUT("/me", auth.AuthMiddleware(), UpdateProfileHandler(userRepo))
	}
}
