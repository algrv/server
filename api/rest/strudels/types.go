package strudels

import (
	"github.com/algrv/server/algorave/strudels"
	"github.com/algrv/server/api/rest/pagination"
)

// StrudelsListResponse wraps a list of strudels with pagination
type StrudelsListResponse struct {
	Strudels   []strudels.Strudel `json:"strudels"`
	Pagination pagination.Meta    `json:"pagination"`
}

// MessageResponse for simple success messages
type MessageResponse struct {
	Message string `json:"message"`
}
