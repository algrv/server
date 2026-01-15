package websocket

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"codeberg.org/algorave/server/algorave/sessions"
	"codeberg.org/algorave/server/internal/auth"
	"codeberg.org/algorave/server/internal/errors"
	"codeberg.org/algorave/server/internal/logger"
	ws "codeberg.org/algorave/server/internal/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     ws.CheckOrigin,
}

// handles WebSocket connections for real-time collaboration.
// see docs/websocket/API.md for usage documentation.
func WebSocketHandler(hub *ws.Hub, sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params ConnectParams
		if err := c.ShouldBindQuery(&params); err != nil {
			errors.BadRequest(c, "invalid parameters", err)
			return
		}

		// use timeout context for DB operations to prevent hanging
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		var session *sessions.Session
		var userID string
		var displayName string
		var role string

		// case 1: No session_id provided - create new anonymous session
		if params.SessionID == "" {
			// check for JWT token first (authenticated user creating session)
			if params.Token != "" {
				claims, err := auth.ValidateJWT(params.Token)
				if err == nil {
					userID = claims.UserID

					// create session with authenticated user as host
					newSession, err := sessionRepo.CreateSession(ctx, &sessions.CreateSessionRequest{
						HostUserID: userID,
						Title:      "New Session",
						Code:       "",
					})
					if err != nil {
						errors.InternalError(c, "failed to create session", err)
						return
					}

					session = newSession
					displayName = "Host"
					role = "host"

					// copy code from previous session if provided
					if params.PreviousSessionID != "" {
						oldSession, err := sessionRepo.GetSession(ctx, params.PreviousSessionID)
						if err == nil && oldSession.Code != "" {
							session.Code = oldSession.Code
							if updateErr := sessionRepo.UpdateSessionCode(ctx, session.ID, oldSession.Code); updateErr != nil {
								logger.Warn("failed to copy code from previous session",
									"new_session_id", session.ID,
									"previous_session_id", params.PreviousSessionID,
									"error", updateErr,
								)
							}
						}
					}
				}
			}

			// no valid JWT - create anonymous session
			if session == nil {
				newSession, err := sessionRepo.CreateAnonymousSession(ctx)
				if err != nil {
					errors.InternalError(c, "failed to create anonymous session", err)
					return
				}

				session = newSession
				role = "host" // anonymous user is "host" of their own session

				if params.DisplayName != "" {
					displayName = params.DisplayName
				} else {
					displayName = "Anonymous"
				}
			}

			params.SessionID = session.ID
		} else {
			// case 2: session_id provided - validate and join existing session
			if !errors.IsValidUUID(params.SessionID) {
				errors.BadRequest(c, "invalid session_id format", nil)
				return
			}

			var err error
			session, err = sessionRepo.GetSession(ctx, params.SessionID)
			if err != nil {
				errors.SessionNotFound(c)
				return
			}

			if !session.IsActive {
				errors.Forbidden(c, "session has ended")
				return
			}

			// try JWT authentication
			if params.Token != "" {
				claims, err := auth.ValidateJWT(params.Token)
				if err == nil {
					userID = claims.UserID

					if userID == session.HostUserID {
						role = "host"
						displayName = "Host"
					} else {
						role = "viewer"
						displayName = "Viewer"
					}
				}
			}

			// try invite token if no JWT or JWT failed
			if role == "" && params.InviteToken != "" {
				inviteToken, err := sessionRepo.ValidateInviteToken(ctx, params.InviteToken)
				if err != nil {
					errors.InvalidInvite(c, "")
					return
				}

				if inviteToken.SessionID != params.SessionID {
					errors.InvalidInvite(c, "invite token is for a different session")
					return
				}

				if inviteToken.MaxUses != nil && inviteToken.UsesCount >= *inviteToken.MaxUses {
					errors.Forbidden(c, "invite token has reached maximum uses")
					return
				}

				role = inviteToken.Role

				if params.DisplayName != "" {
					displayName = params.DisplayName
				} else {
					displayName = fmt.Sprintf("Anonymous %s", inviteToken.Role)
				}

				// increment invite token usage count
				if err := sessionRepo.IncrementTokenUses(ctx, inviteToken.ID); err != nil {
					logger.Warn("failed to increment invite token uses",
						"session_id", params.SessionID,
						"token_id", inviteToken.ID,
						"error", err,
					)
				}
			}

			// if still no role, reject connection
			if role == "" {
				errors.Unauthorized(c, "valid authentication required")
				return
			}
		}

		// check connection limits before accepting new connection
		ipAddress := c.ClientIP()
		canAccept, reason := hub.CanAcceptConnection(userID, ipAddress)

		if !canAccept {
			errors.TooManyRequests(c, reason)
			return
		}

		clientID, err := ws.GenerateClientID()
		if err != nil {
			errors.InternalError(c, "failed to generate client ID", err)
			return
		}

		// upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.ErrorErr(err, "failed to upgrade connection",
				"session_id", params.SessionID,
				"ip", ipAddress,
			)

			return
		}

		// track IP connection only after successful upgrade
		hub.TrackIPConnection(ipAddress)

		isAuthenticated := userID != ""
		initialCode := session.Code

		// fetch chat history for the session (chat is session-scoped)
		var chatHistory []ws.SessionStateChatMessage
		messages, err := sessionRepo.GetChatMessages(ctx, params.SessionID, 50)
		if err != nil {
			logger.Warn("failed to fetch chat history",
				"session_id", params.SessionID,
				"error", err,
			)
		} else {
			for _, msg := range messages {
				msgDisplayName := ""
				if msg.DisplayName != nil {
					msgDisplayName = *msg.DisplayName
				}
				avatarURL := ""
				if msg.AvatarURL != nil {
					avatarURL = *msg.AvatarURL
				}
				chatHistory = append(chatHistory, ws.SessionStateChatMessage{
					DisplayName: msgDisplayName,
					AvatarURL:   avatarURL,
					Content:     msg.Content,
					Timestamp:   msg.CreatedAt.UnixMilli(),
				})
			}
		}

		client := ws.NewClient(clientID, params.SessionID, userID, displayName, role, ipAddress, initialCode, chatHistory, isAuthenticated, conn, hub)

		// add participant to session (authenticated or anonymous)
		// note: anonymous hosts are not added to participants table as they're already tracked via the session itself
		if isAuthenticated {
			_, err = sessionRepo.AddAuthenticatedParticipant(ctx, params.SessionID, userID, displayName, role)
			if err != nil {
				logger.Warn("failed to add authenticated participant",
					"session_id", params.SessionID,
					"user_id", userID,
					"error", err,
				)
			}
		} else if role != "host" {
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
