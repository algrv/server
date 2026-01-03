package websocket

type ConnectParams struct {
	SessionID         string `form:"session_id"`                     // optional - if not provided, creates new anonymous session
	PreviousSessionID string `form:"previous_session_id"`            // optional - copy code from this session when creating new one
	Token             string `form:"token"`                          // jwt token for authenticated users
	InviteToken       string `form:"invite"`                         // invite token for joining sessions
	DisplayName       string `form:"display_name" binding:"max=100"` // optional display name for anonymous users
}
