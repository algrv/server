package errors

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36 characters)
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

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

// error categories for classification
const (
	CategoryDatabase   = "database"
	CategoryNetwork    = "network"
	CategoryValidation = "validation"
	CategoryAuth       = "auth"
	CategoryNotFound   = "not_found"
	CategoryTimeout    = "timeout"
	CategoryUnknown    = "unknown"
)

// analyzes an error and returns its category and sanitized message
func classifyError(err error) ErrorInfo {
	if err == nil {
		return ErrorInfo{CategoryUnknown, ""}
	}

	env := os.Getenv("ENVIRONMENT")
	isProduction := env == "production"

	// database errors (pgx-specific)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return ErrorInfo{
			category:  CategoryDatabase,
			sanitized: ternary(isProduction, "database operation failed", err.Error()),
		}
	}

	// no rows found
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrorInfo{
			category:  CategoryNotFound,
			sanitized: ternary(isProduction, "resource not found", err.Error()),
		}
	}

	// context errors
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorInfo{
			category:  CategoryTimeout,
			sanitized: ternary(isProduction, "request timed out", err.Error()),
		}
	}

	if errors.Is(err, context.Canceled) {
		return ErrorInfo{
			category:  CategoryTimeout,
			sanitized: ternary(isProduction, "request canceled", err.Error()),
		}
	}

	// fallback to string matching for unknown error types
	errMsg := strings.ToLower(err.Error())

	// timeout/deadline
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline") {
		return ErrorInfo{
			category:  CategoryTimeout,
			sanitized: ternary(isProduction, "request timed out", err.Error()),
		}
	}

	// not found
	if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no rows") {
		return ErrorInfo{
			category:  CategoryNotFound,
			sanitized: ternary(isProduction, "resource not found", err.Error()),
		}
	}

	// database (fallback for non-pgx database errors)
	if strings.Contains(errMsg, "database") || strings.Contains(errMsg, "sql") ||
		strings.Contains(errMsg, "postgres") || strings.Contains(errMsg, "pgx") {
		return ErrorInfo{
			category:  CategoryDatabase,
			sanitized: ternary(isProduction, "database operation failed", err.Error()),
		}
	}

	// network
	if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "network") ||
		strings.Contains(errMsg, "dial") {
		return ErrorInfo{
			category:  CategoryNetwork,
			sanitized: ternary(isProduction, "connection error occurred", err.Error()),
		}
	}

	// validation
	if strings.Contains(errMsg, "validation") || strings.Contains(errMsg, "binding") ||
		strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "required") {
		return ErrorInfo{
			category:  CategoryValidation,
			sanitized: ternary(isProduction, "validation failed", err.Error()),
		}
	}

	// auth
	if strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "forbidden") ||
		strings.Contains(errMsg, "permission") || strings.Contains(errMsg, "auth") {
		return ErrorInfo{
			category:  CategoryAuth,
			sanitized: ternary(isProduction, "permission denied", err.Error()),
		}
	}

	// unknown - generic response
	return ErrorInfo{
		category:  CategoryUnknown,
		sanitized: ternary(isProduction, "an error occurred", err.Error()),
	}
}

// ternary helper for cleaner conditional assignment
func ternary(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}

	return falseVal
}
