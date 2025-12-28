package websocket

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/algorave/server/algorave/sessions"
	"github.com/algorave/server/internal/auth"
	"github.com/algorave/server/internal/logger"
	ws "github.com/algorave/server/internal/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     ws.CheckOrigin,
}

// WebSocketHandler godoc
// @Summary WebSocket connection
// @Description Establish WebSocket connection for real-time collaboration. Supports authentication via JWT token or invite token.
// @Description
// @Description Message Types:
// @Description - code_update: Real-time code changes
// @Description - agent_request: AI code generation requests
// @Description - chat_message: Chat messages
// @Description - user_joined: User join notifications
// @Description - user_left: User leave notifications
// @Tags websocket
// @Accept json
// @Produce json
// @Param session_id query string true "Session ID (UUID)"
// @Param token query string false "JWT authentication token"
// @Param invite_token query string false "Session invite token"
// @Param display_name query string false "Display name for anonymous users"
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 429 {object} map[string]string
// @Router /api/v1/ws [get]
func WebSocketHandler(hub *ws.Hub, sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params ConnectParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_params", "message": err.Error()})
			return
		}

		// verify session exists and is active
		ctx := c.Request.Context()
		session, err := sessionRepo.GetSession(ctx, params.SessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "Session does not exist"})
			return
		}

		if !session.IsActive {
			c.JSON(http.StatusForbidden, gin.H{"error": "session_inactive", "message": "Session has ended"})
			return
		}

		// authenticate and determine user info
		var userID string
		var displayName string
		var role string

		if params.Token != "" {
			claims, err := auth.ValidateJWT(params.Token)
			if err == nil {
				userID = claims.UserID
				if userID == session.HostUserID {
					role = "host"
					displayName = "Host"
				} else {
					// user is authenticated but not host - default to viewer
					role = "viewer"
					displayName = "Viewer"
				}
			}
		}

		// try invite token authentication if no JWT or JWT failed
		if role == "" && params.InviteToken != "" {
			inviteToken, err := sessionRepo.ValidateInviteToken(ctx, params.InviteToken)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_invite", "message": "Invalid or expired invite token"})
				return
			}

			if inviteToken.SessionID != params.SessionID {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong_session", "message": "Invite token is for a different session"})
				return
			}

			if inviteToken.MaxUses != nil && inviteToken.UsesCount >= *inviteToken.MaxUses {
				c.JSON(http.StatusForbidden, gin.H{"error": "invite_expired", "message": "Invite token has reached maximum uses"})
				return
			}

			role = inviteToken.Role

			if params.DisplayName != "" {
				displayName = params.DisplayName
			} else {
				displayName = fmt.Sprintf("Anonymous %s", inviteToken.Role)
			}
		}

		// if still no role, reject connection
		if role == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "Valid authentication required"})
			return
		}

		// check connection limits before accepting new connection
		ipAddress := c.ClientIP()
		canAccept, reason := hub.CanAcceptConnection(userID, ipAddress)

		if !canAccept {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "connection_limit_exceeded", "message": reason})
			return
		}

		clientID, err := ws.GenerateClientID()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "Failed to generate client ID"})
			return
		}

		hub.TrackIPConnection(ipAddress)

		// upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.ErrorErr(err, "failed to upgrade connection",
				"session_id", params.SessionID,
				"ip", ipAddress,
			)

			return
		}

		isAuthenticated := userID != ""
		client := ws.NewClient(clientID, params.SessionID, userID, displayName, role, ipAddress, isAuthenticated, conn, hub)

		// add participant to session (authenticated or anonymous)
		if isAuthenticated {
			_, err = sessionRepo.AddAuthenticatedParticipant(ctx, params.SessionID, userID, displayName, role)
			if err != nil {
				logger.Warn("failed to add authenticated participant",
					"session_id", params.SessionID,
					"user_id", userID,
					"error", err,
				)
			}
		} else {
			_, err = sessionRepo.AddAnonymousParticipant(ctx, params.SessionID, displayName, role)
			if err != nil {
				logger.Warn("failed to add anonymous participant",
					"session_id", params.SessionID,
					"error", err,
				)
			}
		}

		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()

		logger.Info("websocket connection established",
			"client_id", clientID,
			"session_id", params.SessionID,
			"role", role,
			"user_id", userID,
			"ip", ipAddress,
		)
	}
}
