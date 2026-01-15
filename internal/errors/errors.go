package errors

import (
	"fmt"
	"net/http"
	"strings"

	"codeberg.org/algorave/server/internal/logger"
	"github.com/gin-gonic/gin"
)

// @AGENTS & @CONTRIBUTORS: RECIPE FOR ERROR HANDLING:
//
// for HTTP REST handlers:
//   - use errors.InternalError(), errors.BadRequest(), etc. for critical errors
//     these functions handle both logging and HTTP response automatically
//   - use logger.ErrorErr() only for non-critical errors where processing continues
//   - never call both logger.ErrorErr() and errors.InternalError() for the same error
//
// for WebSocket handlers:
//   - use logger.ErrorErr() + client.SendError() + return err
//   - this provides both server-side logging and client-side error notification
//
// for services/repositories/internal packages:
//   - return wrapped errors with context using fmt.Errorf("context: %w", err)
//   - let the caller (handler) decide how to log and respond
//   - do not log errors in non-handler code (avoid double logging)
//
// error classification:
//   - errors are automatically classified for logging (adds "error_category" field)
//   - classification happens in InternalError() for better log filtering
//   - categories: database, network, validation, auth, not_found, timeout, unknown

// Unauthorized returns a 401 unauthorized error
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "authentication required"
	}

	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Error:   CodeUnauthorized,
		Message: message,
	})
}

// Forbidden returns a 403 forbidden error
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "permission denied"
	}

	c.JSON(http.StatusForbidden, ErrorResponse{
		Error:   CodeForbidden,
		Message: message,
	})
}

// NotFound returns a 404 not found error
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

// BadRequest returns a 400 bad request error
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
		info := classifyError(err)
		response.Details = info.sanitized
	}

	c.JSON(http.StatusBadRequest, response)
}

// ValidationError returns a 400 bad request error for validation failures
func ValidationError(c *gin.Context, err error) {
	message := "validation failed"
	details := ""

	if err != nil {
		info := classifyError(err)
		details = info.sanitized
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

// InternalError returns a 500 internal server error
func InternalError(c *gin.Context, message string, err error) {
	if message == "" {
		message = "an error occurred"
	}

	// classify error once (single pass for both logging and response)
	info := classifyError(err)

	// log with category
	logger.ErrorErr(err, message,
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"user_id", c.GetString("user_id"),
		"error_category", info.category,
	)

	// return sanitized error to client
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:   CodeServerError,
		Message: message,
		Details: info.sanitized,
	})
}

// Conflict returns a 409 conflict error
func Conflict(c *gin.Context, message string) {
	if message == "" {
		message = "resource conflict"
	}

	c.JSON(http.StatusConflict, ErrorResponse{
		Error:   CodeConflict,
		Message: message,
	})
}

// TooManyRequests returns a 429 too many requests error
func TooManyRequests(c *gin.Context, message string) {
	if message == "" {
		message = "too many requests"
	}

	c.JSON(http.StatusTooManyRequests, ErrorResponse{
		Error:   CodeTooManyRequests,
		Message: message,
	})
}

// InvalidOperation returns a 400 bad request error for invalid operations
func InvalidOperation(c *gin.Context, message string) {
	if message == "" {
		message = "invalid operation"
	}

	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   CodeInvalidOperation,
		Message: message,
	})
}

// SessionNotFound returns a 404 error for session not found
func SessionNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, ErrorResponse{
		Error:   CodeSessionNotFound,
		Message: "session not found",
	})
}

// InvalidInvite returns a 401 error for invalid invite tokens
func InvalidInvite(c *gin.Context, message string) {
	if message == "" {
		message = "invalid or expired invite token"
	}

	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Error:   CodeInvalidInvite,
		Message: message,
	})
}

// ParticipantNotFound returns a 404 error for participant not found
func ParticipantNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, ErrorResponse{
		Error:   CodeParticipantNotFound,
		Message: "participant not found in this session",
	})
}

// IsValidUUID validates a UUID string format
func IsValidUUID(id string) bool {
	if id == "" {
		return false
	}

	return uuidRegex.MatchString(strings.ToLower(id))
}

// ValidateUUID validates a UUID string and returns 404 if invalid
func ValidateUUID(c *gin.Context, id string, resourceName string) bool {
	if id != "" && !IsValidUUID(id) {
		NotFound(c, resourceName)
		return false
	}

	return true
}

// ValidatePathUUID validates a UUID parameter from the request path
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

// ErrUnsupportedProvider returns an error for unsupported provider
func ErrUnsupportedProvider(provider string) error {
	return fmt.Errorf("unsupported provider: %s (supported: openai, claude)", provider)
}
