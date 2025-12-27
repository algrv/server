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

// handles WebSocket connection upgrades
func WebSocketHandler(hub *ws.Hub, sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// parse query parameters
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

		// try JWT authentication first
		if params.Token != "" {
			claims, err := auth.ValidateJWT(params.Token)
			if err == nil {
				userID = claims.UserID
				// check if user is the host
				if userID == session.HostUserID {
					role = "host"
					displayName = "Host"
				} else {
					// user is authenticated but not host - default to viewer
					// they should use invite token for higher permissions
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

			// check max uses
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

		// check connection limits before accepting
		ipAddress := c.ClientIP()
		canAccept, reason := hub.CanAcceptConnection(userID, ipAddress)
		if !canAccept {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "connection_limit_exceeded", "message": reason})
			return
		}

		// generate client ID
		clientID, err := ws.GenerateClientID()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "Failed to generate client ID"})
			return
		}

		// track IP connection (will be untracked when client disconnects)
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

		// determine if user is authenticated
		isAuthenticated := userID != ""

		// create client
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
				// continue anyway - this might be a duplicate participant
			}
		} else {
			_, err = sessionRepo.AddAnonymousParticipant(ctx, params.SessionID, displayName, role)
			if err != nil {
				logger.Warn("failed to add anonymous participant",
					"session_id", params.SessionID,
					"error", err,
				)
				// continue anyway - participant might already exist
			}
		}

		// register client with hub
		hub.Register <- client

		// start client pumps
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
