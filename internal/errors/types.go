package errors

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`             // error code (e.g., "unauthorized", "not_found")
	Message string `json:"message"`           // user-friendly message
	Details string `json:"details,omitempty"` // optional details (sanitized in production)
}

type ErrorInfo struct {
	category  string
	sanitized string
}
