package strudels

import (
	"fmt"
	"net/http"
	"strings"

	"codeberg.org/algorave/server/algorave/strudels"
	"codeberg.org/algorave/server/api/rest/pagination"
	"codeberg.org/algorave/server/internal/attribution"
	"codeberg.org/algorave/server/internal/auth"
	"codeberg.org/algorave/server/internal/ccsignals"
	"codeberg.org/algorave/server/internal/errors"
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
func CreateStrudelHandler(strudelRepo *strudels.Repository, fpIndexer FingerprintIndexer) gin.HandlerFunc {
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

		// index fingerprint if no-ai signal (for paste protection)
		if fpIndexer != nil && strudel.CCSignal != nil {
			fpIndexer.IndexStrudel(strudel.ID, strudel.UserID, strudel.Code, ccsignals.CCSignal(*strudel.CCSignal))
		}

		c.JSON(http.StatusCreated, strudel)
	}
}

// ListStrudelsHandler godoc
// @Summary List user's strudels
// @Description Get strudels owned by the authenticated user with pagination, search, and filtering
// @Tags strudels
// @Produce json
// @Param limit query int false "Items per page (max 100)" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Param search query string false "Search in title and description"
// @Param tags query []string false "Filter by tags (comma-separated)"
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

		limit, offset := parsePaginationParams(c)
		params := pagination.DefaultParams(limit, offset, 20, 100)
		filter := parseFilterParams(c)

		strudelsList, total, err := strudelRepo.List(c.Request.Context(), userID, params.Limit, params.Offset, filter)
		if err != nil {
			errors.InternalError(c, "failed to list strudels", err)
			return
		}

		c.JSON(http.StatusOK, StrudelsListResponse{
			Strudels:   strudelsList,
			Pagination: pagination.NewMeta(params, total),
		})
	}
}

// GetStrudelHandler godoc
// @Summary Get strudel by ID
// @Description Get a specific strudel by ID (owner or public)
// @Tags strudels
// @Produce json
// @Param id path string true "Strudel ID (UUID)"
// @Success 200 {object} StrudelDetailResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/strudels/{id} [get]
func GetStrudelHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		strudelID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		var strudel *strudels.Strudel

		// try to get as owner first if authenticated
		if userID, exists := auth.GetUserID(c); exists {
			strudel, _ = strudelRepo.Get(c.Request.Context(), strudelID, userID) //nolint:errcheck // fallback to public below
		}

		// fall back to public strudel if not owner
		if strudel == nil {
			var err error
			strudel, err = strudelRepo.GetPublic(c.Request.Context(), strudelID)
			if err != nil {
				errors.NotFound(c, "strudel")
				return
			}
		}

		// fetch full conversation history from strudel_messages
		messages, err := strudelRepo.GetStrudelMessages(c.Request.Context(), strudelID, 100)
		if err != nil {
			// non-fatal, return strudel with empty conversation
			messages = []*strudels.StrudelMessage{}
		}

		// convert to DTO (reverse order since DB returns DESC)
		conversationHistory := make([]ConversationMessageDTO, len(messages))
		for i, msg := range messages {
			// convert strudel references
			strudelRefs := make([]StrudelReferenceDTO, len(msg.StrudelReferences))
			for j, ref := range msg.StrudelReferences {
				strudelRefs[j] = StrudelReferenceDTO{
					ID:         ref.ID,
					Title:      ref.Title,
					AuthorName: ref.AuthorName,
					URL:        ref.URL,
				}
			}
			// convert doc references
			docRefs := make([]DocReferenceDTO, len(msg.DocReferences))
			for j, ref := range msg.DocReferences {
				docRefs[j] = DocReferenceDTO{
					PageName:     ref.PageName,
					SectionTitle: ref.SectionTitle,
					URL:          ref.URL,
				}
			}

			conversationHistory[len(messages)-1-i] = ConversationMessageDTO{
				ID:                  msg.ID,
				Role:                msg.Role,
				Content:             msg.Content,
				IsActionable:        msg.IsActionable,
				IsCodeResponse:      msg.IsCodeResponse,
				ClarifyingQuestions: msg.ClarifyingQuestions,
				StrudelReferences:   strudelRefs,
				DocReferences:       docRefs,
				CreatedAt:           msg.CreatedAt,
			}
		}

		// fetch parent CC signal if this is a fork
		var parentCCSignal *strudels.CCSignal
		if strudel.ForkedFrom != nil {
			parentCCSignal, _ = strudelRepo.GetParentCCSignal(c.Request.Context(), *strudel.ForkedFrom) //nolint:errcheck // parent may have been deleted
		}

		c.JSON(http.StatusOK, StrudelDetailResponse{
			ID:                  strudel.ID,
			UserID:              strudel.UserID,
			Title:               strudel.Title,
			Code:                strudel.Code,
			IsPublic:            strudel.IsPublic,
			CCSignal:            strudel.CCSignal,
			ForkedFrom:          strudel.ForkedFrom,
			ParentCCSignal:      parentCCSignal,
			Description:         strudel.Description,
			Tags:                strudel.Tags,
			Categories:          strudel.Categories,
			ConversationHistory: conversationHistory,
			CreatedAt:           strudel.CreatedAt,
			UpdatedAt:           strudel.UpdatedAt,
		})
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
func UpdateStrudelHandler(strudelRepo *strudels.Repository, fpIndexer FingerprintIndexer) gin.HandlerFunc {
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

		// update fingerprint index (remove old, add new if no-ai)
		if fpIndexer != nil {
			fpIndexer.RemoveStrudel(strudel.ID)
			if strudel.CCSignal != nil {
				fpIndexer.IndexStrudel(strudel.ID, strudel.UserID, strudel.Code, ccsignals.CCSignal(*strudel.CCSignal))
			}
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
func DeleteStrudelHandler(strudelRepo *strudels.Repository, fpIndexer FingerprintIndexer) gin.HandlerFunc {
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

		// remove from fingerprint index
		if fpIndexer != nil {
			fpIndexer.RemoveStrudel(strudelID)
		}

		c.JSON(http.StatusOK, MessageResponse{Message: "strudel deleted"})
	}
}

// ListPublicStrudelsHandler godoc
// @Summary List public strudels
// @Description Get publicly shared strudels from all users with pagination, search, and filtering
// @Tags strudels
// @Produce json
// @Param limit query int false "Items per page (max 100)" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Param search query string false "Search in title and description"
// @Param tags query []string false "Filter by tags (comma-separated)"
// @Success 200 {object} StrudelsListResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/public/strudels [get]
func ListPublicStrudelsHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset := parsePaginationParams(c)
		params := pagination.DefaultParams(limit, offset, 20, 100)
		filter := parseFilterParams(c)

		strudelsList, total, err := strudelRepo.ListPublic(c.Request.Context(), params.Limit, params.Offset, filter)
		if err != nil {
			errors.InternalError(c, "failed to list public strudels", err)
			return
		}

		c.JSON(http.StatusOK, StrudelsListResponse{
			Strudels:   strudelsList,
			Pagination: pagination.NewMeta(params, total),
		})
	}
}

// GetPublicStrudelHandler godoc
// @Summary Get public strudel by ID
// @Description Get a publicly shared strudel by its ID (for forking)
// @Tags strudels
// @Produce json
// @Param id path string true "Strudel ID (UUID)"
// @Success 200 {object} strudels.Strudel
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/public/strudels/{id} [get]
func GetPublicStrudelHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		strudelID := c.Param("id")

		if !errors.IsValidUUID(strudelID) {
			errors.BadRequest(c, "invalid strudel ID format", nil)
			return
		}

		strudel, err := strudelRepo.GetPublic(c.Request.Context(), strudelID)
		if err != nil {
			errors.NotFound(c, "strudel")
			return
		}

		c.JSON(http.StatusOK, strudel)
	}
}

// ListPublicTagsHandler godoc
// @Summary List public tags
// @Description Get all unique tags from public strudels
// @Tags strudels
// @Produce json
// @Success 200 {object} TagsListResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/public/strudels/tags [get]
func ListPublicTagsHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tags, err := strudelRepo.ListPublicTags(c.Request.Context())
		if err != nil {
			errors.InternalError(c, "failed to list tags", err)
			return
		}

		if tags == nil {
			tags = []string{}
		}

		c.JSON(http.StatusOK, TagsListResponse{Tags: tags})
	}
}

// ListUserTagsHandler godoc
// @Summary List user's tags
// @Description Get all unique tags from the authenticated user's strudels
// @Tags strudels
// @Produce json
// @Success 200 {object} TagsListResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/strudels/tags [get]
// @Security BearerAuth
func ListUserTagsHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		tags, err := strudelRepo.ListUserTags(c.Request.Context(), userID)
		if err != nil {
			errors.InternalError(c, "failed to list tags", err)
			return
		}

		if tags == nil {
			tags = []string{}
		}

		c.JSON(http.StatusOK, TagsListResponse{Tags: tags})
	}
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)

	return i, err
}

func parsePaginationParams(c *gin.Context) (limit, offset int) {
	if l, ok := c.GetQuery("limit"); ok {
		if parsedLimit, err := parseInt(l); err == nil {
			limit = parsedLimit
		}
	}
	if o, ok := c.GetQuery("offset"); ok {
		if parsedOffset, err := parseInt(o); err == nil {
			offset = parsedOffset
		}
	}
	return limit, offset
}

func parseFilterParams(c *gin.Context) strudels.ListFilter {
	filter := strudels.ListFilter{}

	if search, ok := c.GetQuery("search"); ok {
		filter.Search = search
	}

	if tagsStr, ok := c.GetQuery("tags"); ok && tagsStr != "" {
		filter.Tags = strings.Split(tagsStr, ",")

		// trim whitespace from each tag
		for i, tag := range filter.Tags {
			filter.Tags[i] = strings.TrimSpace(tag)
		}
	}

	return filter
}

// GetStrudelStatsHandler godoc
// @Summary Get strudel usage stats
// @Description Get attribution stats for a public strudel (how many times it was used as RAG context)
// @Tags strudels
// @Produce json
// @Param id path string true "Strudel ID (UUID)"
// @Success 200 {object} attribution.StrudelStatsResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/public/strudels/{id}/stats [get]
func GetStrudelStatsHandler(strudelRepo *strudels.Repository, attrService *attribution.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		strudelID := c.Param("id")

		if !errors.IsValidUUID(strudelID) {
			errors.BadRequest(c, "invalid strudel ID format", nil)
			return
		}

		// verify strudel exists and is public
		_, err := strudelRepo.GetPublic(c.Request.Context(), strudelID)
		if err != nil {
			errors.NotFound(c, "strudel")
			return
		}

		stats, err := attrService.GetStrudelStatsResponse(c.Request.Context(), strudelID)
		if err != nil {
			errors.InternalError(c, "failed to get strudel stats", err)
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}
