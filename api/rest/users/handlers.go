package users

import (
	"net/http"

	"codeberg.org/algorave/server/algorave/users"
	"codeberg.org/algorave/server/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetUsage godoc
// @Summary Get user's usage statistics
// @Description Returns usage statistics for the authenticated user including today's count, daily limit, and usage history
// @Tags users
// @Produce json
// @Success 200 {object} UsageResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/users/usage [get]
// @Security BearerAuth
func GetUsage(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		if userID == "" {
			errors.Unauthorized(c, "user not authenticated")
			return
		}

		var todayCount int

		err := db.QueryRow(c.Request.Context(), `
			SELECT get_user_usage_today($1)
		`, userID).Scan(&todayCount)
		if err != nil {
			errors.InternalError(c, "failed to fetch usage data", err)
			return
		}

		var tier string
		err = db.QueryRow(c.Request.Context(), `
			SELECT tier FROM users WHERE id = $1
		`, userID).Scan(&tier)
		if err != nil {
			errors.InternalError(c, "failed to fetch user tier", err)
			return
		}

		limit := 100
		if tier == "payg" || tier == "byok" {
			limit = -1
		}

		rows, err := db.Query(c.Request.Context(), `
			SELECT DATE(created_at) as date, COUNT(*) as count
			FROM usage_logs
			WHERE user_id = $1
			AND is_byok = false
			AND created_at >= CURRENT_DATE - INTERVAL '30 days'
			GROUP BY DATE(created_at)
			ORDER BY date DESC
		`, userID)
		if err != nil {
			errors.InternalError(c, "failed to fetch usage history", err)
			return
		}

		defer rows.Close()

		history := []DailyUsage{}

		for rows.Next() {
			var du DailyUsage
			var date string
			if err := rows.Scan(&date, &du.Count); err != nil {
				continue
			}

			du.Date = date
			history = append(history, du)
		}

		remaining := limit - todayCount

		if limit == -1 {
			remaining = -1
		}

		c.JSON(http.StatusOK, UsageResponse{
			Tier:      tier,
			Today:     todayCount,
			Limit:     limit,
			Remaining: remaining,
			History:   history,
		})
	}
}

// UpdateTrainingConsent godoc
// @Summary Update user's training consent
// @Description Toggle whether the user's public strudels can be used for AI training
// @Tags users
// @Accept json
// @Produce json
// @Param request body TrainingConsentRequest true "Consent data"
// @Success 200 {object} users.User
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/users/training-consent [put]
// @Security BearerAuth
func UpdateTrainingConsent(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		if userID == "" {
			errors.Unauthorized(c, "user not authenticated")
			return
		}

		var req TrainingConsentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		repo := users.NewRepository(db)
		user, err := repo.UpdateTrainingConsent(c.Request.Context(), userID, req.TrainingConsent)
		if err != nil {
			errors.InternalError(c, "failed to update training consent", err)
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

// UpdateAIFeaturesEnabled godoc
// @Summary Update user's AI features setting
// @Description Toggle whether AI features (prompt bar, code generation) are enabled for the user
// @Tags users
// @Accept json
// @Produce json
// @Param request body AIFeaturesEnabledRequest true "AI features enabled data"
// @Success 200 {object} users.User
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/users/ai-features-enabled [put]
// @Security BearerAuth
func UpdateAIFeaturesEnabled(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		if userID == "" {
			errors.Unauthorized(c, "user not authenticated")
			return
		}

		var req AIFeaturesEnabledRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		repo := users.NewRepository(db)
		user, err := repo.UpdateAIFeaturesEnabled(c.Request.Context(), userID, req.AIFeaturesEnabled)
		if err != nil {
			errors.InternalError(c, "failed to update AI features enabled setting", err)
			return
		}

		c.JSON(http.StatusOK, user)
	}
}
