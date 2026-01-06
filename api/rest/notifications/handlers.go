package notifications

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/algrv/server/internal/errors"
	"github.com/algrv/server/internal/notifications"
)

func ListHandler(svc *notifications.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			errors.Unauthorized(c, "authentication required")
			return
		}

		userID, ok := userIDVal.(string)
		if !ok {
			errors.Unauthorized(c, "invalid user ID")
			return
		}

		limit := 50
		if l := c.Query("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}

		unreadOnly := c.Query("unread") == "true"
		notifs, err := svc.ListForUser(c.Request.Context(), userID, limit, unreadOnly)
		if err != nil {
			errors.InternalError(c, "failed to fetch notifications", err)
			return
		}

		unreadCount, err := svc.GetUnreadCount(c.Request.Context(), userID)
		if err != nil {
			unreadCount = 0
		}

		response := make([]NotificationResponse, 0, len(notifs))

		for _, n := range notifs {
			response = append(response, NotificationResponse{
				ID:        n.ID,
				Type:      n.Type,
				Title:     n.Title,
				Body:      n.Body,
				Data:      n.Data,
				Read:      n.Read,
				CreatedAt: n.CreatedAt,
			})
		}

		c.JSON(http.StatusOK, ListResponse{
			Notifications: response,
			UnreadCount:   unreadCount,
		})
	}
}

func MarkReadHandler(svc *notifications.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			errors.Unauthorized(c, "authentication required")
			return
		}
		userID, ok := userIDVal.(string)
		if !ok {
			errors.Unauthorized(c, "invalid user ID")
			return
		}

		notificationID := c.Param("id")
		if notificationID == "" {
			errors.BadRequest(c, "notification ID required", nil)
			return
		}

		if err := svc.MarkRead(c.Request.Context(), userID, notificationID); err != nil {
			errors.InternalError(c, "failed to mark notification as read", err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func MarkAllReadHandler(svc *notifications.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			errors.Unauthorized(c, "authentication required")
			return
		}

		userID, ok := userIDVal.(string)
		if !ok {
			errors.Unauthorized(c, "invalid user ID")
			return
		}

		if err := svc.MarkAllRead(c.Request.Context(), userID); err != nil {
			errors.InternalError(c, "failed to mark notifications as read", err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func UnreadCountHandler(svc *notifications.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			errors.Unauthorized(c, "authentication required")
			return
		}

		userID, ok := userIDVal.(string)
		if !ok {
			errors.Unauthorized(c, "invalid user ID")
			return
		}

		count, err := svc.GetUnreadCount(c.Request.Context(), userID)
		if err != nil {
			errors.InternalError(c, "failed to get unread count", err)
			return
		}

		c.JSON(http.StatusOK, UnreadCountResponse{Count: count})
	}
}
