package auth

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"slices"

	"codeberg.org/algorave/server/algorave/users"
	"codeberg.org/algorave/server/internal/auth"
	"codeberg.org/algorave/server/internal/errors"
	"codeberg.org/algorave/server/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
)

// BeginAuthHandler godoc
// @Summary Start OAuth authentication
// @Description Begin OAuth authentication flow with specified provider (google, github, apple)
// @Tags auth
// @Param provider path string true "OAuth provider" Enums(google, github, apple)
// @Param redirect_url query string false "URL to redirect to after authentication"
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

		q := c.Request.URL.Query()
		q.Add("provider", provider)

		// encode redirect URL in OAuth state parameter (survives the OAuth redirect)
		if redirectURL := c.Query("redirect_url"); redirectURL != "" {
			state := base64.URLEncoding.EncodeToString([]byte(redirectURL))
			q.Set("state", state)
		}

		c.Request.URL.RawQuery = q.Encode()

		gothic.BeginAuthHandler(c.Writer, c.Request)
	}
}

// CallbackHandler godoc
// @Summary OAuth callback
// @Description OAuth provider callback. Redirects to original URL with token, or returns JSON if no redirect URL
// @Tags auth
// @Produce json
// @Param provider path string true "OAuth provider" Enums(google, github, apple)
// @Success 302 {string} string "Redirect to original URL with token"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/auth/{provider}/callback [get]
func CallbackHandler(userRepo *users.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Param("provider")

		// extract redirect URL from state before gothic processes it
		redirectURL := extractRedirectURL(c.Query("state"))

		q := c.Request.URL.Query()
		q.Add("provider", provider)
		c.Request.URL.RawQuery = q.Encode()

		gothUser, err := gothic.CompleteUserAuth(c.Writer, c.Request)
		if err != nil {
			handleAuthError(c, redirectURL, "authentication failed", err)
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
			handleAuthError(c, redirectURL, "failed to create user", err)
			return
		}

		token, err := auth.GenerateJWT(user.ID, user.Email, user.IsAdmin)
		if err != nil {
			handleAuthError(c, redirectURL, "failed to generate token", err)
			return
		}

		// redirect to original URL with token if available
		if redirectURL != "" {
			parsedURL, err := url.Parse(redirectURL)
			if err == nil {
				query := parsedURL.Query()
				query.Set("token", token)
				parsedURL.RawQuery = query.Encode()
				c.Redirect(http.StatusTemporaryRedirect, parsedURL.String())
				return
			}
		}

		// fallback to JSON response if no redirect URL
		c.JSON(http.StatusOK, AuthResponse{
			User:  user,
			Token: token,
		})
	}
}

// extractRedirectURL decodes the redirect URL from the OAuth state parameter
func extractRedirectURL(state string) string {
	if state == "" {
		return ""
	}
	decoded, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return ""
	}
	return string(decoded)
}

// handleAuthError redirects to the original URL with error params, or returns JSON error
func handleAuthError(c *gin.Context, redirectURL, message string, err error) {
	if redirectURL != "" {
		parsedURL, parseErr := url.Parse(redirectURL)
		if parseErr == nil {
			query := parsedURL.Query()
			query.Set("error", message)
			parsedURL.RawQuery = query.Encode()
			c.Redirect(http.StatusTemporaryRedirect, parsedURL.String())
			return
		}
	}

	errors.InternalError(c, message, err)
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
