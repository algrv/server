package collaboration

import (
	"github.com/gin-gonic/gin"

	"codeberg.org/algorave/server/algorave/sessions"
	"codeberg.org/algorave/server/internal/auth"
)

func RegisterRoutes(router *gin.RouterGroup, sessionRepo sessions.Repository, sessionEnder SessionEnder) {
	// live sessions (optional auth - includes user's sessions if authenticated)
	router.GET("/sessions/live", auth.OptionalAuthMiddleware(), ListLiveSessionsHandler(sessionRepo))

	// user's last active session (for recovery)
	router.GET("/sessions/last", auth.AuthMiddleware(), GetLastUserSessionHandler(sessionRepo))

	// session management (authenticated)
	router.POST("/sessions", auth.AuthMiddleware(), CreateSessionHandler(sessionRepo))
	router.GET("/sessions", auth.AuthMiddleware(), ListUserSessionsHandler(sessionRepo))
	router.GET("/sessions/:id", auth.AuthMiddleware(), GetSessionHandler(sessionRepo))
	router.PUT("/sessions/:id", auth.AuthMiddleware(), UpdateSessionCodeHandler(sessionRepo))
	router.DELETE("/sessions/:id", auth.AuthMiddleware(), EndSessionHandler(sessionRepo, sessionEnder))
	router.POST("/sessions/:id/leave", auth.AuthMiddleware(), LeaveSessionHandler(sessionRepo))
	router.PUT("/sessions/:id/discoverable", auth.AuthMiddleware(), SetDiscoverableHandler(sessionRepo))

	// soft-end live session (kicks participants, revokes invites, keeps code)
	router.POST("/sessions/:id/end-live", auth.AuthMiddleware(), SoftEndSessionHandler(sessionRepo, sessionEnder))

	// check live status
	router.GET("/sessions/:id/live-status", auth.AuthMiddleware(), GetSessionLiveStatusHandler(sessionRepo))

	// session messages
	router.GET("/sessions/:id/messages", auth.AuthMiddleware(), GetSessionMessagesHandler(sessionRepo))

	// invite tokens (host only)
	router.POST("/sessions/:id/invite", auth.AuthMiddleware(), CreateInviteTokenHandler(sessionRepo))
	router.GET("/sessions/:id/invite", auth.AuthMiddleware(), ListInviteTokensHandler(sessionRepo))
	router.DELETE("/sessions/:id/invite/:token_id", auth.AuthMiddleware(), RevokeInviteTokenHandler(sessionRepo))

	// participants
	router.GET("/sessions/:id/participants", auth.AuthMiddleware(), ListParticipantsHandler(sessionRepo))
	router.DELETE("/sessions/:id/participants/:participant_id", auth.AuthMiddleware(), RemoveParticipantHandler(sessionRepo))
	router.PATCH("/sessions/:id/participants/:participant_id", auth.AuthMiddleware(), UpdateParticipantRoleHandler(sessionRepo))

	// join session (optional auth)
	router.POST("/sessions/join", auth.OptionalAuthMiddleware(), JoinSessionHandler(sessionRepo))
}
