package strudels

import "github.com/algoraveai/server/algorave/strudels"

// StrudelsListResponse wraps a list of strudels
type StrudelsListResponse struct {
	Strudels []strudels.Strudel `json:"strudels"`
}

// MessageResponse for simple success messages
type MessageResponse struct {
	Message string `json:"message"`
}
