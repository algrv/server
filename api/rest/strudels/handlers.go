package strudels

import (
	"fmt"
	"net/http"

	"github.com/algorave/server/algorave/strudels"
	"github.com/algorave/server/internal/auth"
	"github.com/gin-gonic/gin"
)

// CreateStrudelHandler creates a new strudel for the authenticated user
func CreateStrudelHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		var req strudels.CreateStrudelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		strudel, err := strudelRepo.Create(c.Request.Context(), userID, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create strudel"})
			return
		}

		c.JSON(http.StatusCreated, strudel)
	}
}

// ListStrudelsHandler lists all strudels for the authenticated user
func ListStrudelsHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		strudelsList, err := strudelRepo.List(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list strudels"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"strudels": strudelsList})
	}
}

// GetStrudelHandler gets a single strudel by ID
func GetStrudelHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		strudelID := c.Param("id")
		strudel, err := strudelRepo.Get(c.Request.Context(), strudelID, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "strudel not found"})
			return
		}

		c.JSON(http.StatusOK, strudel)
	}
}

// UpdateStrudelHandler updates a strudel
func UpdateStrudelHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		strudelID := c.Param("id")
		var req strudels.UpdateStrudelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		strudel, err := strudelRepo.Update(c.Request.Context(), strudelID, userID, req)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "strudel not found"})
			return
		}

		c.JSON(http.StatusOK, strudel)
	}
}

// DeleteStrudelHandler deletes a strudel
func DeleteStrudelHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		strudelID := c.Param("id")
		err := strudelRepo.Delete(c.Request.Context(), strudelID, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "strudel not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "strudel deleted"})
	}
}

// ListPublicStrudelsHandler lists public strudels (no auth required)
func ListPublicStrudelsHandler(strudelRepo *strudels.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 50 // default limit
		if l, ok := c.GetQuery("limit"); ok {
			if parsedLimit, err := parseInt(l); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}

		strudelsList, err := strudelRepo.ListPublic(c.Request.Context(), limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list public strudels"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"strudels": strudelsList})
	}
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
