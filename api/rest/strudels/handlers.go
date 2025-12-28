package strudels

import (
	"fmt"
	"net/http"

	"github.com/algorave/server/algorave/strudels"
	"github.com/algorave/server/internal/auth"
	"github.com/algorave/server/internal/errors"
	"github.com/gin-gonic/gin"
)

// CreateStrudelHandler godoc
// @Summary Create strudel
// @Description Save a new Strudel pattern with code, title, and metadata
// @Tags strudels
// @Accept json
// @Produce json
// @Param request body strudels.CreateStrudelRequest true "Strudel data"
// @Success 201 {object} strudels.Strudel
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/strudels [post]
// @Security BearerAuth
func CreateStrudelHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		var req strudels.CreateStrudelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		strudel, err := strudelRepo.Create(c.Request.Context(), userID, req)
		if err != nil {
			errors.InternalError(c, "failed to create strudel", err)
			return
		}

		c.JSON(http.StatusCreated, strudel)
	}
}

// ListStrudelsHandler godoc
// @Summary List user's strudels
// @Description Get all strudels owned by the authenticated user
// @Tags strudels
// @Produce json
// @Success 200 {object} StrudelsListResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/strudels [get]
// @Security BearerAuth
func ListStrudelsHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		strudelsList, err := strudelRepo.List(c.Request.Context(), userID)
		if err != nil {
			errors.InternalError(c, "failed to list strudels", err)
			return
		}

		c.JSON(http.StatusOK, StrudelsListResponse{Strudels: strudelsList})
	}
}

// GetStrudelHandler godoc
// @Summary Get strudel by ID
// @Description Get a specific strudel by ID (must be owner)
// @Tags strudels
// @Produce json
// @Param id path string true "Strudel ID (UUID)"
// @Success 200 {object} strudels.Strudel
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/strudels/{id} [get]
// @Security BearerAuth
func GetStrudelHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		strudelID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		strudel, err := strudelRepo.Get(c.Request.Context(), strudelID, userID)
		if err != nil {
			errors.NotFound(c, "strudel")
			return
		}

		c.JSON(http.StatusOK, strudel)
	}
}

// UpdateStrudelHandler godoc
// @Summary Update strudel
// @Description Update a strudel's properties (must be owner)
// @Tags strudels
// @Accept json
// @Produce json
// @Param id path string true "Strudel ID (UUID)"
// @Param request body strudels.UpdateStrudelRequest true "Update data"
// @Success 200 {object} strudels.Strudel
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/strudels/{id} [put]
// @Security BearerAuth
func UpdateStrudelHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		strudelID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		var req strudels.UpdateStrudelRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		strudel, err := strudelRepo.Update(c.Request.Context(), strudelID, userID, req)
		if err != nil {
			errors.NotFound(c, "strudel")
			return
		}

		c.JSON(http.StatusOK, strudel)
	}
}

// DeleteStrudelHandler godoc
// @Summary Delete strudel
// @Description Delete a strudel (must be owner)
// @Tags strudels
// @Produce json
// @Param id path string true "Strudel ID (UUID)"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/strudels/{id} [delete]
// @Security BearerAuth
func DeleteStrudelHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		strudelID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		err := strudelRepo.Delete(c.Request.Context(), strudelID, userID)
		if err != nil {
			errors.NotFound(c, "strudel")
			return
		}

		c.JSON(http.StatusOK, MessageResponse{Message: "strudel deleted"})
	}
}

// ListPublicStrudelsHandler godoc
// @Summary List public strudels
// @Description Get publicly shared strudels from all users
// @Tags strudels
// @Produce json
// @Param limit query int false "Limit results (max 100)" default(50)
// @Success 200 {object} StrudelsListResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/public/strudels [get]
func ListPublicStrudelsHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 50

		if l, ok := c.GetQuery("limit"); ok {
			if parsedLimit, err := parseInt(l); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}

		strudelsList, err := strudelRepo.ListPublic(c.Request.Context(), limit)
		if err != nil {
			errors.InternalError(c, "failed to list public strudels", err)
			return
		}

		c.JSON(http.StatusOK, StrudelsListResponse{Strudels: strudelsList})
	}
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)

	return i, err
}
