package collaboration

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/algorave/server/algorave/sessions"
	"github.com/algorave/server/internal/auth"
	"github.com/algorave/server/internal/logger"
)

// creates a new collaborative session
func CreateSessionHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "authentication required"})
			return
		}

		// parse request
		var req CreateSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
			return
		}

		// create session
		session, err := sessionRepo.CreateSession(c.Request.Context(), &sessions.CreateSessionRequest{
			HostUserID: userID,
			Title:      req.Title,
			Code:       req.Code,
		})
		if err != nil {
			logger.ErrorErr(err, "failed to create session",
				"user_id", userID,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "Failed to create session"})
			return
		}

		// return response
		c.JSON(http.StatusCreated, CreateSessionResponse{
			ID:           session.ID,
			HostUserID:   session.HostUserID,
			Title:        session.Title,
			Code:         session.Code,
			IsActive:     session.IsActive,
			CreatedAt:    session.CreatedAt,
			LastActivity: session.LastActivity,
		})
	}
}

// retrieves a session by ID
func GetSessionHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		// get session
		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "session not found"})
			return
		}

		// get participants (both authenticated and anonymous)
		participants, err := sessionRepo.ListAllParticipants(c.Request.Context(), sessionID)
		if err != nil {
			logger.ErrorErr(err, "failed to list participants",
				"session_id", sessionID,
			)
		}

		// convert to response format
		participantResponses := make([]ParticipantResponse, 0, len(participants))

		for _, p := range participants {
			participantResponses = append(participantResponses, ParticipantResponse{
				ID:          p.ID,
				UserID:      p.UserID,
				DisplayName: &p.DisplayName,
				Role:        p.Role,
				Status:      p.Status,
				JoinedAt:    p.JoinedAt,
				LeftAt:      p.LeftAt,
			})
		}

		c.JSON(http.StatusOK, SessionResponse{
			ID:           session.ID,
			HostUserID:   session.HostUserID,
			Title:        session.Title,
			Code:         session.Code,
			IsActive:     session.IsActive,
			CreatedAt:    session.CreatedAt,
			EndedAt:      session.EndedAt,
			LastActivity: session.LastActivity,
			Participants: participantResponses,
		})
	}
}

// lists all sessions for the currently authenticated user
func ListUserSessionsHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "authentication required"})
			return
		}

		// get active_only query parameter
		activeOnly := c.DefaultQuery("active_only", "false") == "true"

		// get sessions
		userSessions, err := sessionRepo.GetUserSessions(c.Request.Context(), userID, activeOnly)
		if err != nil {
			logger.ErrorErr(err, "failed to get user sessions",
				"user_id", userID,
				"active_only", activeOnly,
			)

			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to retrieve sessions"})

			return
		}

		// convert to response format
		responses := make([]SessionResponse, 0, len(userSessions))

		for _, s := range userSessions {
			responses = append(responses, SessionResponse{
				ID:           s.ID,
				HostUserID:   s.HostUserID,
				Title:        s.Title,
				Code:         s.Code,
				IsActive:     s.IsActive,
				CreatedAt:    s.CreatedAt,
				EndedAt:      s.EndedAt,
				LastActivity: s.LastActivity,
			})
		}

		c.JSON(http.StatusOK, gin.H{"sessions": responses})
	}
}

// updates the strudel code in a session
func UpdateSessionCodeHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		// get authenticated user
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "authentication required"})
			return
		}

		// get session to verify it exists
		_, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "session not found"})
			return
		}

		// check if user is host or co-author (must be authenticated)
		participant, err := sessionRepo.GetAuthenticatedParticipant(c.Request.Context(), sessionID, userID)
		if err != nil || (participant.Role != "host" && participant.Role != "co-author") {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "you don't have permission to edit this session"})
			return
		}

		// parse request
		var req UpdateSessionCodeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
			return
		}

		// update code
		if err := sessionRepo.UpdateSessionCode(c.Request.Context(), sessionID, req.Code); err != nil {
			logger.ErrorErr(err, "failed to update session code",
				"session_id", sessionID,
				"user_id", userID,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to update code"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "code updated successfully", "code": req.Code})
	}
}

// ends a session
func EndSessionHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		// get authenticated user
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "authentication required"})
			return
		}

		// get session to verify host
		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "session not found"})
			return
		}

		// only host can end session
		if session.HostUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "only the host can end the session"})
			return
		}

		// end session
		if err := sessionRepo.EndSession(c.Request.Context(), sessionID); err != nil {
			logger.ErrorErr(err, "failed to end session",
				"session_id", sessionID,
				"user_id", userID,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to end session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "session ended successfully"})
	}
}

// creates an invite token for a session
func CreateInviteTokenHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		// get authenticated user
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "authentication required"})
			return
		}

		// get session to verify host
		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "session not found"})
			return
		}

		// only host can create invite tokens
		if session.HostUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "only the host can create invite tokens"})
			return
		}

		// parse request
		var req CreateInviteTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
			return
		}

		// create invite token
		token, err := sessionRepo.CreateInviteToken(c.Request.Context(), &sessions.CreateInviteTokenRequest{
			SessionID: sessionID,
			Role:      req.Role,
			MaxUses:   req.MaxUses,
			ExpiresAt: req.ExpiresAt,
		})
		if err != nil {
			logger.ErrorErr(err, "failed to create invite token",
				"session_id", sessionID,
				"user_id", userID,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to create invite token"})
			return
		}

		c.JSON(http.StatusCreated, InviteTokenResponse{
			ID:        token.ID,
			SessionID: token.SessionID,
			Token:     token.Token,
			Role:      token.Role,
			MaxUses:   token.MaxUses,
			UsesCount: token.UsesCount,
			ExpiresAt: token.ExpiresAt,
			CreatedAt: token.CreatedAt,
		})
	}
}

// lists all participants in a session
func ListParticipantsHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		// get participants (both authenticated and anonymous)
		participants, err := sessionRepo.ListAllParticipants(c.Request.Context(), sessionID)
		if err != nil {
			logger.ErrorErr(err, "failed to list participants",
				"session_id", sessionID,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to retrieve participants"})
			return
		}

		// convert to response format
		responses := make([]ParticipantResponse, 0, len(participants))
		for _, p := range participants {
			responses = append(responses, ParticipantResponse{
				ID:          p.ID,
				UserID:      p.UserID,
				DisplayName: &p.DisplayName,
				Role:        p.Role,
				Status:      p.Status,
				JoinedAt:    p.JoinedAt,
				LeftAt:      p.LeftAt,
			})
		}

		c.JSON(http.StatusOK, gin.H{"participants": responses})
	}
}

// joins a session using an invite token
func JoinSessionHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// parse request
		var req JoinSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
			return
		}

		// validate invite token
		token, err := sessionRepo.ValidateInviteToken(c.Request.Context(), req.InviteToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_invite", "message": "invalid or expired invite token"})
			return
		}

		// get optional authenticated user
		userID, _ := auth.GetUserID(c)

		// determine display name
		displayName := req.DisplayName
		if displayName == "" {
			if userID != "" {
				displayName = "User"
			} else {
				displayName = "Anonymous"
			}
		}

		// add participant (authenticated or anonymous based on userID)
		if userID != "" {
			// authenticated user
			_, err = sessionRepo.AddAuthenticatedParticipant(c.Request.Context(), token.SessionID, userID, displayName, token.Role)
			if err != nil {
				logger.ErrorErr(err, "failed to add authenticated participant",
					"session_id", token.SessionID,
					"user_id", userID,
				)

				c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to join session"})

				return
			}
		} else {
			// anonymous user
			_, err = sessionRepo.AddAnonymousParticipant(c.Request.Context(), token.SessionID, displayName, token.Role)
			if err != nil {
				logger.ErrorErr(err, "failed to add anonymous participant",
					"session_id", token.SessionID,
				)

				c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to join session"})

				return
			}
		}

		// increment token uses
		if err := sessionRepo.IncrementTokenUses(c.Request.Context(), token.ID); err != nil {
			logger.ErrorErr(err, "failed to increment token uses",
				"session_id", token.SessionID,
				"token_id", token.ID,
			)
		}

		c.JSON(http.StatusOK, JoinSessionResponse{
			SessionID:   token.SessionID,
			Role:        token.Role,
			DisplayName: displayName,
		})
	}
}

// leaves a session
func LeaveSessionHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		// get authenticated user
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "authentication required"})
			return
		}

		// get participant record
		participant, err := sessionRepo.GetAuthenticatedParticipant(c.Request.Context(), sessionID, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_participant", "message": "you are not a participant in this session"})
			return
		}

		// mark as left
		if err := sessionRepo.MarkAuthenticatedParticipantLeft(c.Request.Context(), participant.ID); err != nil {
			logger.ErrorErr(err, "failed to mark participant as left",
				"session_id", sessionID,
				"participant_id", participant.ID,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to leave session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "successfully left session"})
	}
}

// gets a session's conversation history
func GetSessionMessagesHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		limit := 100

		if limitStr := c.Query("limit"); limitStr != "" {
			var parsedLimit int
			if _, err := fmt.Sscanf(limitStr, "%d", &parsedLimit); err == nil {
				if parsedLimit > 0 && parsedLimit <= 1000 {
					limit = parsedLimit
				}
			}
		}

		// get messages
		messages, err := sessionRepo.GetMessages(c.Request.Context(), sessionID, limit)
		if err != nil {
			logger.ErrorErr(err, "failed to get messages",
				"session_id", sessionID,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to retrieve messages"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"messages": messages})
	}
}

// removes a participant from a session (kick)
func RemoveParticipantHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")
		participantID := c.Param("participant_id")

		// get authenticated user
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "authentication required"})
			return
		}

		// get session to verify host
		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "session not found"})
			return
		}

		// only host can remove participants
		if session.HostUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "only the host can remove participants"})
			return
		}

		// get participant to ensure they're in this session
		participant, err := sessionRepo.GetParticipantByID(c.Request.Context(), participantID)
		if err != nil || participant.SessionID != sessionID {
			c.JSON(http.StatusNotFound, gin.H{"error": "participant_not_found", "message": "participant not found in this session"})
			return
		}

		// can't remove yourself
		if participant.UserID != nil && *participant.UserID == userID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_operation", "message": "cannot remove yourself. use leave endpoint instead"})
			return
		}

		// remove participant
		if err := sessionRepo.RemoveParticipant(c.Request.Context(), participantID); err != nil {
			logger.ErrorErr(err, "failed to remove participant",
				"session_id", sessionID,
				"participant_id", participantID,
				"user_id", userID,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to remove participant"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "participant removed successfully"})
	}
}

// updates a participant's role
func UpdateParticipantRoleHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")
		participantID := c.Param("participant_id")

		// get authenticated user
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "authentication required"})
			return
		}

		// get session to verify host
		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "session not found"})
			return
		}

		// only host can change roles
		if session.HostUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "only the host can change participant roles"})
			return
		}

		// parse request
		var req struct {
			Role string `json:"role" binding:"required,oneof=co-author viewer"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
			return
		}

		// get participant to ensure they're in this session
		participant, err := sessionRepo.GetParticipantByID(c.Request.Context(), participantID)
		if err != nil || participant.SessionID != sessionID {
			c.JSON(http.StatusNotFound, gin.H{"error": "participant_not_found", "message": "participant not found in this session"})
			return
		}

		// can't change host role
		if participant.Role == "host" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_operation", "message": "cannot change host role"})
			return
		}

		// update role
		if err := sessionRepo.UpdateParticipantRole(c.Request.Context(), participantID, req.Role); err != nil {
			logger.ErrorErr(err, "failed to update participant role",
				"session_id", sessionID,
				"participant_id", participantID,
				"user_id", userID,
				"new_role", req.Role,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to update role"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "role updated successfully", "role": req.Role})
	}
}

// lists all invite tokens for a session
func ListInviteTokensHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		// get authenticated user
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "authentication required"})
			return
		}

		// get session to verify host
		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "session not found"})
			return
		}

		// only host can view invite tokens
		if session.HostUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "only the host can view invite tokens"})
			return
		}

		// get invite tokens
		tokens, err := sessionRepo.ListInviteTokens(c.Request.Context(), sessionID)
		if err != nil {
			logger.ErrorErr(err, "failed to list invite tokens",
				"session_id", sessionID,
				"user_id", userID,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to retrieve invite tokens"})
			return
		}

		// convert to response format
		responses := make([]InviteTokenResponse, 0, len(tokens))
		for _, t := range tokens {
			responses = append(responses, InviteTokenResponse{
				ID:        t.ID,
				SessionID: t.SessionID,
				Token:     t.Token,
				Role:      t.Role,
				MaxUses:   t.MaxUses,
				UsesCount: t.UsesCount,
				ExpiresAt: t.ExpiresAt,
				CreatedAt: t.CreatedAt,
			})
		}

		c.JSON(http.StatusOK, gin.H{"tokens": responses})
	}
}

// revokes an invite token
func RevokeInviteTokenHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")
		tokenID := c.Param("token_id")

		// get authenticated user
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "authentication required"})
			return
		}

		// get session to verify host
		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "session not found"})
			return
		}

		// only host can revoke invite tokens
		if session.HostUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "only the host can revoke invite tokens"})
			return
		}

		// revoke token
		if err := sessionRepo.RevokeInviteToken(c.Request.Context(), tokenID); err != nil {
			logger.ErrorErr(err, "failed to revoke invite token",
				"session_id", sessionID,
				"token_id", tokenID,
				"user_id", userID,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "failed to revoke invite token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "invite token revoked successfully"})
	}
}
