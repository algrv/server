package sessions

type TransferSessionRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Title     string `json:"title" binding:"required"`
}
