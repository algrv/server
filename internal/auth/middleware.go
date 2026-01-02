package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// validates JWT tokens and adds user info to context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := ValidateJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("is_admin", claims.IsAdmin)

		c.Next()
	}
}

// extracts user_id from context after AuthMiddleware
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")

	if !exists {
		return "", false
	}

	uid, ok := userID.(string) //nolint:errcheck
	if !ok {
		return "", false
	}

	return uid, true
}

// validates JWT if present but doesn't require it
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.Split(authHeader, " ")

		if len(parts) == 2 && parts[0] == "Bearer" {
			token := parts[1]
			claims, err := ValidateJWT(token)

			if err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("user_email", claims.Email)
				c.Set("is_admin", claims.IsAdmin)
			}
		}

		c.Next()
	}
}

// requires authenticated user with admin role
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check if user is authenticated
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := ValidateJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Check admin role
		if !claims.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("is_admin", claims.IsAdmin)

		c.Next()
	}
}
