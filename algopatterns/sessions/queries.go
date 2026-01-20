package sessions

const (
	// session queries
	queryCreateSession = `
		INSERT INTO sessions (host_user_id, title, code)
		VALUES ($1, $2, $3)
		RETURNING id, host_user_id, title, code, is_active, is_discoverable, created_at, ended_at, last_activity
	`

	queryCreateAnonymousSession = `
		INSERT INTO sessions (host_user_id, title, code)
		VALUES ('00000000-0000-0000-0000-000000000000', 'Anonymous Session', '')
		RETURNING id, host_user_id, title, code, is_active, is_discoverable, created_at, ended_at, last_activity
	`

	queryGetSession = `
		SELECT id, host_user_id, title, code, is_active, is_discoverable, created_at, ended_at, last_activity
		FROM sessions
		WHERE id = $1
	`

	queryGetUserSessions = `
		SELECT DISTINCT s.id, s.host_user_id, s.title, s.code, s.is_active, s.is_discoverable, s.created_at, s.ended_at, s.last_activity
		FROM sessions s
		LEFT JOIN session_participants sp ON s.id = sp.session_id
		WHERE s.host_user_id = $1 OR sp.user_id = $1
		ORDER BY s.last_activity DESC
	`

	queryGetUserSessionsActiveOnly = `
		SELECT DISTINCT s.id, s.host_user_id, s.title, s.code, s.is_active, s.is_discoverable, s.created_at, s.ended_at, s.last_activity
		FROM sessions s
		LEFT JOIN session_participants sp ON s.id = sp.session_id
		WHERE (s.host_user_id = $1 OR sp.user_id = $1) AND s.is_active = true
		ORDER BY s.last_activity DESC
	`

	queryListDiscoverableSessions = `
		SELECT id, host_user_id, title, code, is_active, is_discoverable, created_at, ended_at, last_activity
		FROM sessions
		WHERE is_discoverable = true AND is_active = true
		ORDER BY last_activity DESC
		LIMIT $1 OFFSET $2
	`

	queryCountDiscoverableSessions = `
		SELECT COUNT(*)
		FROM sessions
		WHERE is_discoverable = true AND is_active = true
	`

	queryUpdateSessionCode = `
		UPDATE sessions
		SET code = $1, last_activity = NOW()
		WHERE id = $2
	`

	querySetDiscoverable = `
		UPDATE sessions
		SET is_discoverable = $1
		WHERE id = $2
	`

	queryEndSession = `
		UPDATE sessions
		SET is_active = false, ended_at = NOW()
		WHERE id = $1
	`

	// authenticated participant queries
	queryAddAuthenticatedParticipant = `
		INSERT INTO session_participants (session_id, user_id, display_name, role)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (session_id, user_id) DO UPDATE
		SET status = 'active', left_at = NULL
		RETURNING id, session_id, user_id, display_name, role, status, joined_at, left_at
	`

	queryGetAuthenticatedParticipant = `
		SELECT id, session_id, user_id, display_name, role, status, joined_at, left_at
		FROM session_participants
		WHERE session_id = $1 AND user_id = $2
	`

	queryMarkAuthenticatedParticipantLeft = `
		UPDATE session_participants
		SET status = 'left', left_at = NOW()
		WHERE id = $1
	`

	queryListAuthenticatedParticipants = `
		SELECT id, session_id, user_id, display_name, role, status, joined_at, left_at
		FROM session_participants
		WHERE session_id = $1
		ORDER BY joined_at ASC
	`

	queryGetParticipantByID = `
		SELECT id, session_id, user_id, display_name, role, status, joined_at, left_at
		FROM session_participants
		WHERE id = $1
	`

	queryRemoveAuthenticatedParticipant = `
		UPDATE session_participants
		SET status = 'left', left_at = NOW()
		WHERE id = $1
	`

	queryUpdateParticipantRole = `
		UPDATE session_participants
		SET role = $1
		WHERE id = $2
	`

	// anonymous participant queries
	queryAddAnonymousParticipant = `
		INSERT INTO anonymous_participants (session_id, display_name, role)
		VALUES ($1, $2, $3)
		RETURNING id, session_id, display_name, role, status, joined_at, left_at, expires_at
	`

	queryListAnonymousParticipants = `
		SELECT id, session_id, display_name, role, status, joined_at, left_at, expires_at
		FROM anonymous_participants
		WHERE session_id = $1
		ORDER BY joined_at ASC
	`

	queryRemoveAnonymousParticipant = `
		UPDATE anonymous_participants
		SET status = 'left', left_at = NOW()
		WHERE id = $1
	`

	queryGetAnonymousParticipantByID = `
		SELECT id, session_id, display_name, role, status, joined_at, left_at, expires_at
		FROM anonymous_participants
		WHERE id = $1
	`

	// invite token queries
	queryCreateInviteToken = `
		INSERT INTO invite_tokens (session_id, token, role, max_uses, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, session_id, token, role, max_uses, uses_count, expires_at, created_at
	`

	queryListInviteTokens = `
		SELECT id, session_id, token, role, max_uses, uses_count, expires_at, created_at
		FROM invite_tokens
		WHERE session_id = $1
		ORDER BY created_at DESC
	`

	queryValidateInviteToken = `
		SELECT id, session_id, token, role, max_uses, uses_count, expires_at, created_at
		FROM invite_tokens
		WHERE token = $1
		AND (expires_at IS NULL OR expires_at > NOW())
		AND (max_uses IS NULL OR uses_count < max_uses)
	`

	queryIncrementTokenUses = `
		UPDATE invite_tokens
		SET uses_count = uses_count + 1
		WHERE id = $1
	`

	queryRevokeInviteToken = `
		DELETE FROM invite_tokens
		WHERE id = $1
	`

	// chat message queries (session-scoped)
	queryGetChatMessages = `
		SELECT id, session_id, user_id, role, content, display_name, avatar_url, created_at
		FROM session_messages
		WHERE session_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	queryAddChatMessage = `
		INSERT INTO session_messages (session_id, user_id, role, content, display_name, avatar_url, message_type)
		VALUES ($1, $2, 'user', $3, $4, $5, 'chat')
		RETURNING id, session_id, user_id, role, content, display_name, avatar_url, created_at
	`

	queryUpdateLastActivity = `
		UPDATE sessions
		SET last_activity = NOW()
		WHERE id = $1
	`

	// soft-end session queries
	queryRevokeAllInviteTokens = `
		DELETE FROM invite_tokens
		WHERE session_id = $1
	`

	queryMarkAllAuthParticipantsLeft = `
		UPDATE session_participants
		SET status = 'left', left_at = NOW()
		WHERE session_id = $1 AND user_id != $2 AND status = 'active'
	`

	queryMarkAllAnonParticipantsLeft = `
		UPDATE anonymous_participants
		SET status = 'left', left_at = NOW()
		WHERE session_id = $1 AND status = 'active'
	`

	// get last user session (most recent active session where user is host)
	// only returns host sessions - co-authors/viewers can rejoin via invite link or live sessions list
	queryGetLastUserSession = `
		SELECT s.id, s.host_user_id, s.title, s.code, s.is_active, s.is_discoverable, s.created_at, s.ended_at, s.last_activity
		FROM sessions s
		WHERE s.is_active = true
			AND s.host_user_id = $1
		ORDER BY s.last_activity DESC
		LIMIT 1
	`

	// cleanup queries for stale sessions
	queryListStaleSessions = `
		SELECT id, host_user_id, title, code, is_active, is_discoverable, created_at, ended_at, last_activity
		FROM sessions
		WHERE is_active = true AND last_activity < $1
	`

	queryCountActiveParticipants = `
		SELECT
			(SELECT COUNT(*) FROM session_participants WHERE session_id = $1 AND status = 'active') +
			(SELECT COUNT(*) FROM anonymous_participants WHERE session_id = $1 AND status = 'active')
	`

	queryHasActiveInviteTokens = `
		SELECT EXISTS(
			SELECT 1 FROM invite_tokens
			WHERE session_id = $1
			AND (expires_at IS NULL OR expires_at > NOW())
			AND (max_uses IS NULL OR uses_count < max_uses)
		)
	`
)
