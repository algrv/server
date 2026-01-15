package collaboration

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"codeberg.org/algorave/server/algorave/sessions"
	"codeberg.org/algorave/server/api/rest/pagination"
	"codeberg.org/algorave/server/internal/auth"
	"codeberg.org/algorave/server/internal/errors"
	"codeberg.org/algorave/server/internal/logger"
)

// CreateSessionHandler godoc
// @Summary Create collaboration session
// @Description Create a new collaborative coding session (authenticated users only)
// @Tags sessions
// @Accept json
// @Produce json
// @Param request body CreateSessionRequest true "Session data"
// @Success 201 {object} CreateSessionResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions [post]
// @Security BearerAuth
func CreateSessionHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		// parse request
		var req CreateSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		// create session
		session, err := sessionRepo.CreateSession(c.Request.Context(), &sessions.CreateSessionRequest{
			HostUserID: userID,
			Title:      req.Title,
			Code:       req.Code,
		})
		if err != nil {
			errors.InternalError(c, "failed to create session", err)
			return
		}

		// Set discoverable if requested
		if req.IsDiscoverable != nil && *req.IsDiscoverable {
			if err := sessionRepo.SetDiscoverable(c.Request.Context(), session.ID, true); err != nil {
				logger.ErrorErr(err, "failed to set session discoverable", "session_id", session.ID)
			} else {
				session.IsDiscoverable = true
			}
		}

		c.JSON(http.StatusCreated, CreateSessionResponse{
			ID:             session.ID,
			HostUserID:     session.HostUserID,
			Title:          session.Title,
			Code:           session.Code,
			IsActive:       session.IsActive,
			IsDiscoverable: session.IsDiscoverable,
			CreatedAt:      session.CreatedAt,
			LastActivity:   session.LastActivity,
		})
	}
}

// GetSessionHandler godoc
// @Summary Get session details
// @Description Get session information including participants
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Success 200 {object} SessionResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id} [get]
// @Security BearerAuth
func GetSessionHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		// get session
		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			errors.SessionNotFound(c)
			return
		}

		// get participants (both authenticated and anonymous)
		participants, err := sessionRepo.ListAllParticipants(c.Request.Context(), sessionID)
		if err != nil {
			logger.ErrorErr(err, "failed to list participants",
				"session_id", sessionID,
			)
		}

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
			ID:             session.ID,
			HostUserID:     session.HostUserID,
			Title:          session.Title,
			Code:           session.Code,
			IsActive:       session.IsActive,
			IsDiscoverable: session.IsDiscoverable,
			CreatedAt:      session.CreatedAt,
			EndedAt:        session.EndedAt,
			LastActivity:   session.LastActivity,
			Participants:   participantResponses,
		})
	}
}

// ListUserSessionsHandler godoc
// @Summary List user's sessions
// @Description Get all sessions where user is host or participant
// @Tags sessions
// @Produce json
// @Param active_only query boolean false "Only return active sessions" default(false)
// @Success 200 {object} SessionsListResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions [get]
// @Security BearerAuth
func ListUserSessionsHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)

		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		activeOnly := c.DefaultQuery("active_only", "false") == "true"

		userSessions, err := sessionRepo.GetUserSessions(c.Request.Context(), userID, activeOnly)
		if err != nil {
			errors.InternalError(c, "failed to retrieve sessions", err)
			return
		}

		// convert to response format
		responses := make([]SessionResponse, 0, len(userSessions))

		for _, s := range userSessions {
			responses = append(responses, SessionResponse{
				ID:             s.ID,
				HostUserID:     s.HostUserID,
				Title:          s.Title,
				Code:           s.Code,
				IsActive:       s.IsActive,
				IsDiscoverable: s.IsDiscoverable,
				CreatedAt:      s.CreatedAt,
				EndedAt:        s.EndedAt,
				LastActivity:   s.LastActivity,
			})
		}

		c.JSON(http.StatusOK, SessionsListResponse{Sessions: responses})
	}
}

// UpdateSessionCodeHandler godoc
// @Summary Update session code
// @Description Update the code in a session (host or co-authors only)
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Param request body UpdateSessionCodeRequest true "Code update"
// @Success 200 {object} UpdateSessionCodeResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id} [put]
// @Security BearerAuth
func UpdateSessionCodeHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		_, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			errors.SessionNotFound(c)
			return
		}

		// check if authenticated user is host or co-author
		participant, err := sessionRepo.GetAuthenticatedParticipant(c.Request.Context(), sessionID, userID)
		if err != nil || (participant.Role != "host" && participant.Role != "co-author") {
			errors.Forbidden(c, "you don't have permission to edit this session")
			return
		}

		// parse request
		var req UpdateSessionCodeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		// update code
		if err := sessionRepo.UpdateSessionCode(c.Request.Context(), sessionID, req.Code); err != nil {
			errors.InternalError(c, "failed to update code", err)
			return
		}

		c.JSON(http.StatusOK, UpdateSessionCodeResponse{
			Message: "code updated successfully",
			Code:    req.Code,
		})
	}
}

// EndSessionHandler godoc
// @Summary End session
// @Description End a collaborative session (host only)
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id} [delete]
// @Security BearerAuth
func EndSessionHandler(sessionRepo sessions.Repository, sessionEnder SessionEnder) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			errors.SessionNotFound(c)
			return
		}

		// only authenticated user can end session
		if session.HostUserID != userID {
			errors.Forbidden(c, "only the host can end the session")
			return
		}

		if err := sessionRepo.EndSession(c.Request.Context(), sessionID); err != nil {
			errors.InternalError(c, "failed to end session", err)
			return
		}

		// notify all WebSocket clients and close their connections
		if sessionEnder != nil {
			sessionEnder.EndSession(sessionID, "session ended by host")
		}

		c.JSON(http.StatusOK, MessageResponse{Message: "session ended successfully"})
	}
}

// CreateInviteTokenHandler godoc
// @Summary Create invite token
// @Description Generate an invite link for joining the session (host only)
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Param request body CreateInviteTokenRequest true "Token settings"
// @Success 201 {object} InviteTokenResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id}/invite [post]
// @Security BearerAuth
func CreateInviteTokenHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			errors.SessionNotFound(c)
			return
		}

		if session.HostUserID != userID {
			errors.Forbidden(c, "only the host can create invite tokens")
			return
		}

		var req CreateInviteTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		token, err := sessionRepo.CreateInviteToken(c.Request.Context(), &sessions.CreateInviteTokenRequest{
			SessionID: sessionID,
			Role:      req.Role,
			MaxUses:   req.MaxUses,
			ExpiresAt: req.ExpiresAt,
		})
		if err != nil {
			errors.InternalError(c, "failed to create invite token", err)
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

// ListParticipantsHandler godoc
// @Summary List session participants
// @Description Get all participants in a session (authenticated and anonymous)
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Success 200 {object} ParticipantsListResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id}/participants [get]
// @Security BearerAuth
func ListParticipantsHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		// get participants (both authenticated and anonymous)
		participants, err := sessionRepo.ListAllParticipants(c.Request.Context(), sessionID)
		if err != nil {
			errors.InternalError(c, "failed to retrieve participants", err)
			return
		}

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

		c.JSON(http.StatusOK, ParticipantsListResponse{Participants: responses})
	}
}

// JoinSessionHandler godoc
// @Summary Join session with invite
// @Description Join a collaborative session using an invite token (authenticated or anonymous)
// @Tags sessions
// @Accept json
// @Produce json
// @Param request body JoinSessionRequest true "Join request with invite token"
// @Success 200 {object} JoinSessionResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse "Invalid or expired invite"
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/join [post]
// @Security BearerAuth
func JoinSessionHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req JoinSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		token, err := sessionRepo.ValidateInviteToken(c.Request.Context(), req.InviteToken)
		if err != nil {
			errors.InvalidInvite(c, "")
			return
		}

		userID, isAuthenticated := auth.GetUserID(c)
		displayName := req.DisplayName

		if displayName == "" {
			if isAuthenticated {
				displayName = "User"
			} else {
				displayName = "Anonymous"
			}
		}

		if isAuthenticated {
			_, err = sessionRepo.AddAuthenticatedParticipant(c.Request.Context(), token.SessionID, userID, displayName, token.Role)
			if err != nil {
				errors.InternalError(c, "failed to join session", err)
				return
			}
		} else {
			_, err = sessionRepo.AddAnonymousParticipant(c.Request.Context(), token.SessionID, displayName, token.Role)
			if err != nil {
				errors.InternalError(c, "failed to join session", err)
				return
			}
		}

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

// LeaveSessionHandler godoc
// @Summary Leave session
// @Description Leave a collaborative session
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id}/leave [post]
// @Security BearerAuth
func LeaveSessionHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		participant, err := sessionRepo.GetAuthenticatedParticipant(c.Request.Context(), sessionID, userID)
		if err != nil {
			errors.NotFound(c, "you are not a participant in this session")
			return
		}

		if err := sessionRepo.MarkAuthenticatedParticipantLeft(c.Request.Context(), participant.ID); err != nil {
			errors.InternalError(c, "failed to leave session", err)
			return
		}

		c.JSON(http.StatusOK, MessageResponse{Message: "successfully left session"})
	}
}

// GetSessionMessagesHandler godoc
// @Summary Get session chat messages
// @Description Retrieve chat messages from a session (AI conversations are strudel-scoped and fetched separately)
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Param limit query int false "Max messages to return (max 1000)" default(100)
// @Success 200 {object} MessagesResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id}/messages [get]
// @Security BearerAuth
func GetSessionMessagesHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		limit := 100

		if limitStr := c.Query("limit"); limitStr != "" {
			var parsedLimit int
			if _, err := fmt.Sscanf(limitStr, "%d", &parsedLimit); err == nil {
				if parsedLimit > 0 && parsedLimit <= 1000 {
					limit = parsedLimit
				}
			}
		}

		// get chat messages (AI conversations are strudel-scoped and not returned here)
		messages, err := sessionRepo.GetChatMessages(c.Request.Context(), sessionID, limit)
		if err != nil {
			errors.InternalError(c, "failed to retrieve messages", err)
			return
		}

		c.JSON(http.StatusOK, MessagesResponse{Messages: messages})
	}
}

// RemoveParticipantHandler godoc
// @Summary Remove participant
// @Description Remove a participant from the session (host only)
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Param participant_id path string true "Participant ID (UUID)"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id}/participants/{participant_id} [delete]
// @Security BearerAuth
func RemoveParticipantHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		participantID, ok := errors.ValidatePathUUID(c, "participant_id")
		if !ok {
			return
		}

		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			errors.SessionNotFound(c)
			return
		}

		if session.HostUserID != userID {
			errors.Forbidden(c, "only the host can remove participants")
			return
		}

		participant, err := sessionRepo.GetParticipantByID(c.Request.Context(), participantID)
		if err != nil || participant.SessionID != sessionID {
			errors.ParticipantNotFound(c)
			return
		}

		if participant.UserID != nil && *participant.UserID == userID {
			errors.InvalidOperation(c, "cannot remove yourself. use leave endpoint instead")
			return
		}

		if err := sessionRepo.RemoveParticipant(c.Request.Context(), participantID); err != nil {
			errors.InternalError(c, "failed to remove participant", err)
			return
		}

		c.JSON(http.StatusOK, MessageResponse{Message: "participant removed successfully"})
	}
}

func UpdateParticipantRoleHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		participantID, ok := errors.ValidatePathUUID(c, "participant_id")
		if !ok {
			return
		}

		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			errors.SessionNotFound(c)
			return
		}

		if session.HostUserID != userID {
			errors.Forbidden(c, "only the host can change participant roles")
			return
		}

		var req struct {
			Role string `json:"role" binding:"required,oneof=co-author viewer"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		participant, err := sessionRepo.GetParticipantByID(c.Request.Context(), participantID)
		if err != nil || participant.SessionID != sessionID {
			errors.ParticipantNotFound(c)
			return
		}

		if participant.Role == "host" {
			errors.InvalidOperation(c, "cannot change host role")
			return
		}

		if err := sessionRepo.UpdateParticipantRole(c.Request.Context(), participantID, req.Role); err != nil {
			errors.InternalError(c, "failed to update role", err)
			return
		}

		c.JSON(http.StatusOK, UpdateRoleResponse{
			Message: "role updated successfully",
			Role:    req.Role,
		})
	}
}

// ListInviteTokensHandler godoc
// @Summary List invite tokens
// @Description Get all invite tokens for a session (host only)
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Success 200 {object} InviteTokensListResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id}/invite [get]
// @Security BearerAuth
func ListInviteTokensHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			errors.SessionNotFound(c)
			return
		}

		if session.HostUserID != userID {
			errors.Forbidden(c, "only the host can view invite tokens")
			return
		}

		tokens, err := sessionRepo.ListInviteTokens(c.Request.Context(), sessionID)
		if err != nil {
			errors.InternalError(c, "failed to retrieve invite tokens", err)
			return
		}

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

		c.JSON(http.StatusOK, InviteTokensListResponse{Tokens: responses})
	}
}

// RevokeInviteTokenHandler godoc
// @Summary Revoke invite token
// @Description Revoke an invite token to prevent further use (host only)
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Param token_id path string true "Token ID (UUID)"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id}/invite/{token_id} [delete]
// @Security BearerAuth
func RevokeInviteTokenHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		tokenID, ok := errors.ValidatePathUUID(c, "token_id")
		if !ok {
			return
		}

		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			errors.SessionNotFound(c)
			return
		}

		if session.HostUserID != userID {
			errors.Forbidden(c, "only the host can revoke invite tokens")
			return
		}

		if err := sessionRepo.RevokeInviteToken(c.Request.Context(), tokenID); err != nil {
			errors.InternalError(c, "failed to revoke invite token", err)
			return
		}

		c.JSON(http.StatusOK, MessageResponse{Message: "invite token revoked successfully"})
	}
}

// ListLiveSessionsHandler godoc
// @Summary List live sessions
// @Description Get discoverable active sessions plus user's own active sessions (if authenticated)
// @Tags sessions
// @Produce json
// @Param limit query int false "Items per page (max 100)" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Success 200 {object} LiveSessionsListResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/live [get]
func ListLiveSessionsHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset := parsePaginationParams(c)
		params := pagination.DefaultParams(limit, offset, 20, 100)

		// check if user is authenticated (optional)
		userID, isAuthenticated := auth.GetUserID(c)

		// track user's session IDs for membership check
		userSessionIDs := make(map[string]bool)

		// track sessions where user is currently active (to exclude from results)
		currentSessionIDs := make(map[string]bool)

		// if authenticated, get user's active sessions first
		var memberSessions []LiveSessionResponse
		if isAuthenticated {
			userSessions, err := sessionRepo.GetUserSessions(c.Request.Context(), userID, true) // active_only=true
			if err == nil {
				for _, s := range userSessions {
					userSessionIDs[s.ID] = true

					participants, _ := sessionRepo.ListAllParticipants(c.Request.Context(), s.ID) //nolint:errcheck // best-effort count

					// count active participants and check if user is currently active in this session
					participantCount := 0
					userIsActive := false
					for _, p := range participants {
						if p.Status == "active" {
							participantCount++
							if p.UserID != nil && *p.UserID == userID {
								userIsActive = true
							}
						}
					}

					// skip sessions where user is currently active (they're already in it)
					if userIsActive {
						currentSessionIDs[s.ID] = true
						continue
					}

					// only include member sessions if they have other participants or are discoverable
					// this prevents showing empty private sessions the user happens to be a member of
					if participantCount <= 1 && !s.IsDiscoverable {
						continue
					}

					memberSessions = append(memberSessions, LiveSessionResponse{
						ID:               s.ID,
						Title:            s.Title,
						ParticipantCount: participantCount,
						IsMember:         true,
						IsDiscoverable:   s.IsDiscoverable,
						CreatedAt:        s.CreatedAt,
						LastActivity:     s.LastActivity,
					})
				}
			}
		}

		// get discoverable sessions
		liveSessions, total, err := sessionRepo.ListDiscoverableSessions(c.Request.Context(), params.Limit, params.Offset)
		if err != nil {
			errors.InternalError(c, "failed to retrieve live sessions", err)
			return
		}

		// build response: member sessions first, then other discoverable sessions
		responses := make([]LiveSessionResponse, 0, len(memberSessions)+len(liveSessions))
		responses = append(responses, memberSessions...)

		for _, s := range liveSessions {
			// skip if already included as member session
			if userSessionIDs[s.ID] {
				continue
			}

			// skip sessions where user is currently active
			if currentSessionIDs[s.ID] {
				continue
			}

			participants, _ := sessionRepo.ListAllParticipants(c.Request.Context(), s.ID) //nolint:errcheck // best-effort count
			participantCount := 0
			for _, p := range participants {
				if p.Status == "active" {
					participantCount++
				}
			}

			responses = append(responses, LiveSessionResponse{
				ID:               s.ID,
				Title:            s.Title,
				ParticipantCount: participantCount,
				IsMember:         false,
				IsDiscoverable:   s.IsDiscoverable,
				CreatedAt:        s.CreatedAt,
				LastActivity:     s.LastActivity,
			})
		}

		// adjust total to account for member sessions that might not be discoverable
		adjustedTotal := total + len(memberSessions)
		for _, s := range liveSessions {
			if userSessionIDs[s.ID] {
				adjustedTotal-- // don't double count
			}
		}

		c.JSON(http.StatusOK, LiveSessionsListResponse{
			Sessions:   responses,
			Pagination: pagination.NewMeta(params, adjustedTotal),
		})
	}
}

// SetDiscoverableHandler godoc
// @Summary Set session discoverability
// @Description Toggle whether a session appears in the live sessions list (host only)
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Param request body SetDiscoverableRequest true "Discoverability settings"
// @Success 200 {object} SessionResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id}/discoverable [put]
// @Security BearerAuth
func SetDiscoverableHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			errors.SessionNotFound(c)
			return
		}

		if session.HostUserID != userID {
			errors.Forbidden(c, "only the host can change discoverability")
			return
		}

		var req SetDiscoverableRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.ValidationError(c, err)
			return
		}

		if err := sessionRepo.SetDiscoverable(c.Request.Context(), sessionID, req.IsDiscoverable); err != nil {
			errors.InternalError(c, "failed to update discoverability", err)
			return
		}

		session.IsDiscoverable = req.IsDiscoverable

		c.JSON(http.StatusOK, SessionResponse{
			ID:             session.ID,
			HostUserID:     session.HostUserID,
			Title:          session.Title,
			Code:           session.Code,
			IsActive:       session.IsActive,
			IsDiscoverable: session.IsDiscoverable,
			CreatedAt:      session.CreatedAt,
			EndedAt:        session.EndedAt,
			LastActivity:   session.LastActivity,
		})
	}
}

func parsePaginationParams(c *gin.Context) (limit, offset int) {
	if limitStr := c.Query("limit"); limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			limit = 0
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
			offset = 0
		}
	}
	return limit, offset
}

// SoftEndSessionHandler godoc
// @Summary Soft-end a live session
// @Description Ends the live portion of a session: kicks all non-host participants, revokes all invite tokens,
// @Description sets discoverable to false. Host keeps access to the session and their code.
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Success 200 {object} SoftEndSessionResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id}/end-live [post]
// @Security BearerAuth
func SoftEndSessionHandler(sessionRepo sessions.Repository, sessionEnder SessionEnder) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			errors.SessionNotFound(c)
			return
		}

		// only host can soft-end the session
		if session.HostUserID != userID {
			errors.Forbidden(c, "only the host can end the live session")
			return
		}

		// count participants before kicking them
		participants, _ := sessionRepo.ListAllParticipants(c.Request.Context(), sessionID) //nolint:errcheck // best-effort count
		participantsKicked := 0

		for _, p := range participants {
			// count active non-host participants
			if p.Status == "active" && (p.UserID == nil || *p.UserID != userID) {
				participantsKicked++
			}
		}

		// 1. set discoverable to false
		if err := sessionRepo.SetDiscoverable(c.Request.Context(), sessionID, false); err != nil {
			logger.ErrorErr(err, "failed to set discoverable to false", "session_id", sessionID)
		}

		// 2. revoke all invite tokens
		invitesRevoked := false
		if err := sessionRepo.RevokeAllInviteTokens(c.Request.Context(), sessionID); err != nil {
			logger.ErrorErr(err, "failed to revoke invite tokens", "session_id", sessionID)
		} else {
			invitesRevoked = true
		}

		// 3. mark all non-host participants as left
		if err := sessionRepo.MarkAllNonHostParticipantsLeft(c.Request.Context(), sessionID, userID); err != nil {
			logger.ErrorErr(err, "failed to mark participants as left", "session_id", sessionID)
		}

		// 4. notify WebSocket clients (they will be disconnected)
		if sessionEnder != nil {
			sessionEnder.EndSession(sessionID, "live session ended by host")
		}

		c.JSON(http.StatusOK, SoftEndSessionResponse{
			Message:            "live session ended successfully",
			ParticipantsKicked: participantsKicked,
			InvitesRevoked:     invitesRevoked,
		})
	}
}

// GetLastUserSessionHandler godoc
// @Summary Get user's last active session for recovery
// @Description Returns the user's most recent active session where they are host, excluding sessions they're currently in
// @Tags sessions
// @Produce json
// @Success 200 {object} LiveSessionResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/last [get]
// @Security BearerAuth
func GetLastUserSessionHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		session, err := sessionRepo.GetLastUserSession(c.Request.Context(), userID)
		if err != nil {
			errors.NotFound(c, "no active session found")
			return
		}

		// get participants and check if user is currently active
		participants, _ := sessionRepo.ListAllParticipants(c.Request.Context(), session.ID) //nolint:errcheck // best-effort count
		participantCount := 0
		userIsActive := false
		for _, p := range participants {
			if p.Status == "active" {
				participantCount++
				if p.UserID != nil && *p.UserID == userID {
					userIsActive = true
				}
			}
		}

		// don't return session if user is currently active in it (they're already in it)
		if userIsActive {
			errors.NotFound(c, "no active session found")
			return
		}

		// don't return session if it's empty and private (no reason to rejoin)
		if participantCount <= 1 && !session.IsDiscoverable {
			errors.NotFound(c, "no active session found")
			return
		}

		c.JSON(http.StatusOK, LiveSessionResponse{
			ID:               session.ID,
			Title:            session.Title,
			ParticipantCount: participantCount,
			IsMember:         true,
			IsDiscoverable:   session.IsDiscoverable,
			CreatedAt:        session.CreatedAt,
			LastActivity:     session.LastActivity,
		})
	}
}

// GetSessionLiveStatusHandler godoc
// @Summary Check if session is live
// @Description Returns whether a session is currently live (has other participants or active invite tokens)
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID (UUID)"
// @Success 200 {object} IsLiveResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /api/v1/sessions/{id}/live-status [get]
// @Security BearerAuth
func GetSessionLiveStatusHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := errors.ValidatePathUUID(c, "id")
		if !ok {
			return
		}

		userID, exists := auth.GetUserID(c)
		if !exists {
			errors.Unauthorized(c, "")
			return
		}

		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			errors.SessionNotFound(c)
			return
		}

		// verify user is host or participant
		if session.HostUserID != userID {
			participant, err := sessionRepo.GetAuthenticatedParticipant(c.Request.Context(), sessionID, userID)
			if err != nil || participant.Status != "active" {
				errors.Forbidden(c, "you are not a member of this session")
				return
			}
		}

		// count active participants (excluding current user)
		participants, _ := sessionRepo.ListAllParticipants(c.Request.Context(), sessionID) //nolint:errcheck // best-effort count
		participantCount := 0
		for _, p := range participants {
			if p.Status == "active" {
				participantCount++
			}
		}

		// check for active invite tokens
		hasActiveTokens, err := sessionRepo.HasActiveInviteTokens(c.Request.Context(), sessionID)
		if err != nil {
			hasActiveTokens = false
		}

		// session is "live" if it has multiple participants OR has active invite tokens
		isLive := participantCount > 1 || hasActiveTokens

		c.JSON(http.StatusOK, IsLiveResponse{
			IsLive:                isLive,
			ParticipantCount:      participantCount,
			HasActiveInviteTokens: hasActiveTokens,
		})
	}
}
