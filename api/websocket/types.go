package websocket

type ConnectParams struct {
	SessionID   string `form:"session_id" binding:"required"`
	Token       string `form:"token"`                          // jwt token for authenticated users
	InviteToken string `form:"invite"`                         // invite token for joining sessions
	DisplayName string `form:"display_name" binding:"max=100"` // optional display name for anonymous users
}
