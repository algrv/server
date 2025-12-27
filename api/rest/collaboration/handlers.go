package collaboration

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/algorave/server/algorave/sessions"
	"github.com/algorave/server/internal/auth"
)

// creates a new collaborative session
func CreateSessionHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// get authenticated user
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "Authentication required"})
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
			log.Printf("Failed to create session: %v", err)
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
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "Session not found"})
			return
		}

		// get participants (both authenticated and anonymous)
		participants, err := sessionRepo.ListAllParticipants(c.Request.Context(), sessionID)
		if err != nil {
			log.Printf("Failed to list participants: %v", err)
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

// lists all sessions for the authenticated user
func ListUserSessionsHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// get authenticated user
		userID, exists := auth.GetUserID(c)

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "Authentication required"})
			return
		}

		// get active_only query parameter
		activeOnly := c.DefaultQuery("active_only", "false") == "true"

		// get sessions
		userSessions, err := sessionRepo.GetUserSessions(c.Request.Context(), userID, activeOnly)
		if err != nil {
			log.Printf("Failed to get user sessions: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "Failed to retrieve sessions"})

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

// updates the code in a session
func UpdateSessionCodeHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		// get authenticated user
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "Authentication required"})
			return
		}

		// get session to verify it exists
		_, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "Session not found"})
			return
		}

		// check if user is host or co-author (must be authenticated)
		participant, err := sessionRepo.GetAuthenticatedParticipant(c.Request.Context(), sessionID, userID)
		if err != nil || (participant.Role != "host" && participant.Role != "co-author") {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "You don't have permission to edit this session"})
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
			log.Printf("Failed to update session code: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "Failed to update code"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Code updated successfully", "code": req.Code})
	}
}

// ends a session
func EndSessionHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		// get authenticated user
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "Authentication required"})
			return
		}

		// get session to verify host
		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "Session not found"})
			return
		}

		// only host can end session
		if session.HostUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Only the host can end the session"})
			return
		}

		// end session
		if err := sessionRepo.EndSession(c.Request.Context(), sessionID); err != nil {
			log.Printf("Failed to end session: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "Failed to end session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Session ended successfully"})
	}
}

// creates an invite token for a session
func CreateInviteTokenHandler(sessionRepo sessions.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		// get authenticated user
		userID, exists := auth.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "message": "Authentication required"})
			return
		}

		// get session to verify host
		session, err := sessionRepo.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found", "message": "Session not found"})
			return
		}

		// only host can create invite tokens
		if session.HostUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "Only the host can create invite tokens"})
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
			log.Printf("Failed to create invite token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "Failed to create invite token"})
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
			log.Printf("Failed to list participants: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "Failed to retrieve participants"})
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_invite", "message": "Invalid or expired invite token"})
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
				log.Printf("Failed to add authenticated participant: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "Failed to join session"})
				return
			}
		} else {
			// anonymous user
			_, err = sessionRepo.AddAnonymousParticipant(c.Request.Context(), token.SessionID, displayName, token.Role)
			if err != nil {
				log.Printf("Failed to add anonymous participant: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "message": "Failed to join session"})
				return
			}
		}

		// increment token uses
		if err := sessionRepo.IncrementTokenUses(c.Request.Context(), token.ID); err != nil {
			log.Printf("Failed to increment token uses: %v", err)
		}

		c.JSON(http.StatusOK, JoinSessionResponse{
			SessionID:   token.SessionID,
			Role:        token.Role,
			DisplayName: displayName,
		})
	}
}
