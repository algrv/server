package auth

import (
	"net/http"
	"os"

	"slices"

	"github.com/algorave/server/algorave/users"
	"github.com/algorave/server/internal/auth"
	"github.com/algorave/server/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth/gothic"
)

var (
	sessionStore = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
)

func init() {
	// configure session options
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600, // 1 hour
		HttpOnly: true,
		Secure:   os.Getenv("ENVIRONMENT") == "production",
		SameSite: http.SameSiteLaxMode,
	}
}

// starts the OAuth flow for a provider
func BeginAuthHandler(userRepo *users.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Param("provider")

		if !isValidProvider(provider) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider"})
			return
		}

		// set provider in query for gothic
		q := c.Request.URL.Query()
		q.Add("provider", provider)
		c.Request.URL.RawQuery = q.Encode()

		gothic.BeginAuthHandler(c.Writer, c.Request)
	}
}

// handles OAuth callbacks
func CallbackHandler(userRepo *users.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Param("provider")

		q := c.Request.URL.Query()
		q.Add("provider", provider)
		c.Request.URL.RawQuery = q.Encode()

		// complete OAuth flow
		gothUser, err := gothic.CompleteUserAuth(c.Writer, c.Request)
		if err != nil {
			logger.ErrorErr(err, "OAuth authentication failed",
				"provider", provider,
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
			return
		}

		user, err := userRepo.FindOrCreateByProvider(
			c.Request.Context(),
			gothUser.Provider,
			gothUser.UserID,
			gothUser.Email,
			gothUser.Name,
			gothUser.AvatarURL,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		// generate JWT token
		token, err := auth.GenerateJWT(user.ID, user.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		// return user + token
		c.JSON(http.StatusOK, gin.H{
			"user":  user,
			"token": token,
		})
	}
}

// returns the current authenticated user
func GetCurrentUserHandler(userRepo *users.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		user, err := userRepo.FindByID(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}

// updates the current user's profile
func UpdateProfileHandler(userRepo *users.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		var req struct {
			Name      string `json:"name" binding:"required"`
			AvatarURL string `json:"avatar_url"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		user, err := userRepo.UpdateProfile(c.Request.Context(), userID, req.Name, req.AvatarURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}

// handles logout (client-side token deletion mainly)
func LogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		gothic.Logout(c.Writer, c.Request)
		c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
	}
}

// checks if provider is supported
func isValidProvider(provider string) bool {
	validProviders := []string{"google", "github", "apple"}
	return slices.Contains(validProviders, provider)
}
