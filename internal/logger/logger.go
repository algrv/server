package logger

import (
	"context"
	"log/slog"
	"os"
)

var (
	// default logger instance
	defaultLogger *slog.Logger
)

// initializes the logger based on environment
func init() {
	env := os.Getenv("ENVIRONMENT")

	var handler slog.Handler

	if env == "production" {
		// production: JSON output for structured logging
		opts := &slog.HandlerOptions{
			Level: slog.LevelInfo, // INFO and above in production
		}
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		// development: human-readable text output
		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug, // DEBUG and above in development
		}
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	defaultLogger = slog.New(handler)
}

// returns the default logger instance
func Default() *slog.Logger {
	return defaultLogger
}

// creates a logger with additional context fields
func With(args ...any) *slog.Logger {
	return defaultLogger.With(args...)
}

// creates a logger with context
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return defaultLogger
	}

	// extract any logger from context if present
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}

	return defaultLogger
}

// adds logger to context
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// helper type for context key
type loggerKey struct{}

// convenience functions for common log levels

// logs a debug message
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// logs an info message
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// logs a warning message
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// logs an error message
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// logs an error with context
func ErrorErr(err error, msg string, args ...any) {
	args = append(args, "error", err)
	defaultLogger.Error(msg, args...)
}

// logs a fatal error and exits (for CLI tools)
func Fatal(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
	os.Exit(1)
}

// logs a fatal error with error and exits (for CLI tools)
func FatalErr(err error, msg string, args ...any) {
	args = append(args, "error", err)
	defaultLogger.Error(msg, args...)
	os.Exit(1)
}
