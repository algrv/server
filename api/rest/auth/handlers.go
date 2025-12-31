package auth

import (
	"net/http"

	"slices"

	"github.com/algoraveai/server/algorave/users"
	"github.com/algoraveai/server/internal/auth"
	"github.com/algoraveai/server/internal/errors"
	"github.com/algoraveai/server/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
)

// BeginAuthHandler godoc
// @Summary Start OAuth authentication
// @Description Begin OAuth authentication flow with specified provider (google, github, apple)
// @Tags auth
// @Param provider path string true "OAuth provider" Enums(google, github, apple)
// @Success 302 {string} string "Redirect to OAuth provider"
// @Failure 400 {object} errors.ErrorResponse
// @Router /api/v1/auth/{provider} [get]
func BeginAuthHandler(_ *users.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Param("provider")

		if !isValidProvider(provider) {
			errors.BadRequest(c, "invalid provider", nil)
			return
		}

		// set provider in query for gothic
		q := c.Request.URL.Query()
		q.Add("provider", provider)
		c.Request.URL.RawQuery = q.Encode()

		gothic.BeginAuthHandler(c.Writer, c.Request)
	}
}

// CallbackHandler godoc
// @Summary OAuth callback
// @Description OAuth provider callback. Returns user data and JWT token
// @Tags auth
// @Produce json
// @Param provider path string true "OAuth provider" Enums(google, github, apple)
// @Success 200 {object} AuthResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/auth/{provider}/callback [get]
func CallbackHandler(userRepo *users.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Param("provider")

		q := c.Request.URL.Query()
		q.Add("provider", provider)
		c.Request.URL.RawQuery = q.Encode()

		gothUser, err := gothic.CompleteUserAuth(c.Writer, c.Request)
		if err != nil {
			errors.InternalError(c, "authentication failed", err)
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
			errors.InternalError(c, "failed to create user", err)
			return
		}

		token, err := auth.GenerateJWT(user.ID, user.Email)
		if err != nil {
			errors.InternalError(c, "failed to generate token", err)
			return
		}

		c.JSON(http.StatusOK, AuthResponse{
			User:  user,
			Token: token,
		})
	}
}

// GetCurrentUserHandler godoc
// @Summary Get current user
// @Description Get authenticated user's profile
// @Tags auth
// @Produce json
// @Success 200 {object} UserResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/auth/me [get]
// @Security BearerAuth
func GetCurrentUserHandler(userRepo *users.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)

		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		user, err := userRepo.FindByID(c.Request.Context(), userID)
		if err != nil {
			errors.NotFound(c, "user")
			return
		}

		c.JSON(http.StatusOK, UserResponse{User: user})
	}
}

// UpdateProfileHandler godoc
// @Summary Update user profile
// @Description Update authenticated user's name and avatar
// @Tags auth
// @Accept json
// @Produce json
// @Param request body UpdateProfileRequest true "Profile update"
// @Success 200 {object} UserResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/auth/me [put]
// @Security BearerAuth
func UpdateProfileHandler(userRepo *users.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		var req UpdateProfileRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		user, err := userRepo.UpdateProfile(c.Request.Context(), userID, req.Name, req.AvatarURL)
		if err != nil {
			errors.InternalError(c, "failed to update profile", err)
			return
		}

		c.JSON(http.StatusOK, UserResponse{User: user})
	}
}

// LogoutHandler godoc
// @Summary Logout
// @Description Clear authentication session
// @Tags auth
// @Produce json
// @Success 200 {object} MessageResponse
// @Router /api/v1/auth/logout [post]
func LogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := gothic.Logout(c.Writer, c.Request); err != nil {
			logger.ErrorErr(err, "failed to logout user from gothic session")
		}
		c.JSON(http.StatusOK, MessageResponse{Message: "logged out successfully"})
	}
}

func isValidProvider(provider string) bool {
	validProviders := []string{"google", "github", "apple"}
	return slices.Contains(validProviders, provider)
}
