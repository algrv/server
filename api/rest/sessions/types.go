package sessions

import "github.com/algorave/server/algorave/strudels"

type TransferSessionRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Title     string `json:"title" binding:"required"`
}

// TransferSessionResponse returned after successful session transfer
type TransferSessionResponse struct {
	Message   string           `json:"message"`
	Strudel   *strudels.Strudel `json:"strudel"`
	StrudelID string           `json:"strudel_id"`
}
