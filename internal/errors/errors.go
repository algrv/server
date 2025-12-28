package errors

import (
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/algorave/server/internal/logger"
	"github.com/gin-gonic/gin"
)

// Error Handling Guidelines:
//
// For HTTP REST handlers:
//   - Use errors.InternalError(), errors.BadRequest(), etc. for critical errors
//     These functions handle both logging and HTTP response automatically
//   - Use logger.ErrorErr() only for non-critical errors where processing continues
//   - Never call both logger.ErrorErr() and errors.InternalError() for the same error
//
// For WebSocket handlers:
//   - Use logger.ErrorErr() + client.SendError() + return err
//   - This provides both server-side logging and client-side error notification
//
// For services/repositories/internal packages:
//   - Return wrapped errors with context using fmt.Errorf("context: %w", err)
//   - Let the caller (handler) decide how to log and respond
//   - Do not log errors in non-handler code (avoid double logging)

// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36 characters)
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`             // error code (e.g., "unauthorized", "not_found")
	Message string `json:"message"`           // user-friendly message
	Details string `json:"details,omitempty"` // optional details (sanitized in production)
}

// standard error codes
const (
	CodeUnauthorized        = "unauthorized"
	CodeForbidden           = "forbidden"
	CodeNotFound            = "not_found"
	CodeValidationError     = "validation_error"
	CodeServerError         = "server_error"
	CodeBadRequest          = "bad_request"
	CodeConflict            = "conflict"
	CodeTooManyRequests     = "too_many_requests"
	CodeInvalidOperation    = "invalid_operation"
	CodeSessionNotFound     = "session_not_found"
	CodeInvalidInvite       = "invalid_invite"
	CodeParticipantNotFound = "participant_not_found"
)

// returns a 401 unauthorized error
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "authentication required"
	}

	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Error:   CodeUnauthorized,
		Message: message,
	})
}

// returns a 403 forbidden error
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "permission denied"
	}

	c.JSON(http.StatusForbidden, ErrorResponse{
		Error:   CodeForbidden,
		Message: message,
	})
}

// returns a 404 not found error
func NotFound(c *gin.Context, resource string) {
	message := "resource not found"

	if resource != "" {
		message = resource + " not found"
	}

	c.JSON(http.StatusNotFound, ErrorResponse{
		Error:   CodeNotFound,
		Message: message,
	})
}

// returns a 400 bad request error
func BadRequest(c *gin.Context, message string, err error) {
	if message == "" {
		message = "invalid request"
	}

	response := ErrorResponse{
		Error:   CodeBadRequest,
		Message: message,
	}

	// add details if error provided
	if err != nil {
		response.Details = sanitizeError(err)
	}

	c.JSON(http.StatusBadRequest, response)
}

// returns a 400 bad request error for validation failures
func ValidationError(c *gin.Context, err error) {
	message := "validation failed"
	details := ""

	if err != nil {
		details = sanitizeError(err)
		// extract a more specific message from validation errors if available
		if strings.Contains(err.Error(), "binding") || strings.Contains(err.Error(), "validation") {
			message = "request validation failed"
		}
	}

	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   CodeValidationError,
		Message: message,
		Details: details,
	})
}

// returns a 500 internal server error
func InternalError(c *gin.Context, message string, err error) {
	if message == "" {
		message = "an error occurred"
	}

	// log full error server-side with context
	logger.ErrorErr(err, message,
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"user_id", c.GetString("user_id"),
	)

	// return sanitized error to client
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:   CodeServerError,
		Message: message,
		Details: sanitizeError(err),
	})
}

// returns a 409 conflict error
func Conflict(c *gin.Context, message string) {
	if message == "" {
		message = "resource conflict"
	}

	c.JSON(http.StatusConflict, ErrorResponse{
		Error:   CodeConflict,
		Message: message,
	})
}

// returns a 429 too many requests error
func TooManyRequests(c *gin.Context, message string) {
	if message == "" {
		message = "too many requests"
	}

	c.JSON(http.StatusTooManyRequests, ErrorResponse{
		Error:   CodeTooManyRequests,
		Message: message,
	})
}

// returns a 400 bad request error for invalid operations
func InvalidOperation(c *gin.Context, message string) {
	if message == "" {
		message = "invalid operation"
	}

	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   CodeInvalidOperation,
		Message: message,
	})
}

// returns a 404 error for session not found
func SessionNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, ErrorResponse{
		Error:   CodeSessionNotFound,
		Message: "session not found",
	})
}

// returns a 401 error for invalid invite tokens
func InvalidInvite(c *gin.Context, message string) {
	if message == "" {
		message = "invalid or expired invite token"
	}

	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Error:   CodeInvalidInvite,
		Message: message,
	})
}

// returns a 404 error for participant not found
func ParticipantNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, ErrorResponse{
		Error:   CodeParticipantNotFound,
		Message: "participant not found in this session",
	})
}

// sanitizes error messages for production
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()
	env := os.Getenv("ENVIRONMENT")

	if env != "production" {
		return errMsg
	}

	if strings.Contains(errMsg, "database") || strings.Contains(errMsg, "sql") {
		return "database operation failed"
	}

	if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "network") {
		return "connection error occurred"
	}

	if strings.Contains(errMsg, "timeout") {
		return "request timed out"
	}

	if strings.Contains(errMsg, "permission") || strings.Contains(errMsg, "unauthorized") {
		return "permission denied"
	}

	if strings.Contains(errMsg, "not found") {
		return "resource not found"
	}

	return "an error occurred"
}

// validates a UUID string format
func IsValidUUID(id string) bool {
	if id == "" {
		return false
	}

	return uuidRegex.MatchString(strings.ToLower(id))
}

// validates a UUID string and returns 404 if invalid
func ValidateUUID(c *gin.Context, id string, resourceName string) bool {
	if id != "" && !IsValidUUID(id) {
		NotFound(c, resourceName)
		return false
	}

	return true
}

// validates a UUID parameter from the request path
func ValidatePathUUID(c *gin.Context, paramName string) (string, bool) {
	id := c.Param(paramName)

	if id == "" {
		BadRequest(c, "missing "+paramName, nil)
		return "", false
	}

	if !IsValidUUID(id) {
		NotFound(c, "resource")
		return "", false
	}

	return id, true
}
