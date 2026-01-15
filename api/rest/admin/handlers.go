package admin

import (
	"net/http"

	"codeberg.org/algorave/server/algorave/strudels"
	"codeberg.org/algorave/server/internal/errors"
	"github.com/gin-gonic/gin"
)

// SetUseInTraining godoc
// @Summary Set use_in_training flag on a strudel
// @Description Admin-only endpoint to mark a strudel for use in training data
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "Strudel ID"
// @Param request body SetUseInTrainingRequest true "Use in training data"
// @Success 200 {object} StrudelAdminResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/admin/strudels/{id}/use-in-training [put]
// @Security AdminKeyAuth
func SetUseInTraining(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		strudelID := c.Param("id")
		if strudelID == "" {
			errors.BadRequest(c, "strudel id required", nil)
			return
		}

		var req SetUseInTrainingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		strudel, err := strudelRepo.AdminSetUseInTraining(c.Request.Context(), strudelID, req.UseInTraining)
		if err != nil {
			errors.InternalError(c, "failed to update use_in_training", err)
			return
		}

		c.JSON(http.StatusOK, StrudelAdminResponse{
			ID:            strudel.ID,
			UserID:        strudel.UserID,
			Title:         strudel.Title,
			Code:          strudel.Code,
			IsPublic:      strudel.IsPublic,
			UseInTraining: strudel.UseInTraining,
			Description:   strudel.Description,
			Tags:          strudel.Tags,
		})
	}
}

// GetStrudel godoc
// @Summary Get any strudel by ID (admin)
// @Description Admin-only endpoint to get any strudel regardless of ownership
// @Tags admin
// @Produce json
// @Param id path string true "Strudel ID"
// @Success 200 {object} StrudelAdminResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/admin/strudels/{id} [get]
// @Security AdminKeyAuth
func GetStrudel(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		strudelID := c.Param("id")
		if strudelID == "" {
			errors.BadRequest(c, "strudel id required", nil)
			return
		}

		strudel, err := strudelRepo.AdminGetStrudel(c.Request.Context(), strudelID)
		if err != nil {
			errors.NotFound(c, "strudel not found")
			return
		}

		c.JSON(http.StatusOK, StrudelAdminResponse{
			ID:            strudel.ID,
			UserID:        strudel.UserID,
			Title:         strudel.Title,
			Code:          strudel.Code,
			IsPublic:      strudel.IsPublic,
			UseInTraining: strudel.UseInTraining,
			Description:   strudel.Description,
			Tags:          strudel.Tags,
		})
	}
}
