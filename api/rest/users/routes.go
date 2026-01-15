package users

import (
	"codeberg.org/algorave/server/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
	users := rg.Group("/users")
	users.Use(auth.AuthMiddleware()) // all user routes require authentication

	users.GET("/usage", GetUsage(db))
	users.PUT("/training-consent", UpdateTrainingConsent(db))
	users.PUT("/ai-features-enabled", UpdateAIFeaturesEnabled(db))
}
