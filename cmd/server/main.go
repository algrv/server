package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codeberg.org/algorave/server/internal/auth"
	"codeberg.org/algorave/server/internal/config"
	"codeberg.org/algorave/server/internal/logger"
)

// @title Algorave API
// @version 1.0
// @description AI-powered Strudel code generation and collaborative live coding platform
// @description
// @description Features:
// @description - AI-powered Strudel code generation from natural language
// @description - Real-time collaborative editing via WebSockets
// @description - OAuth authentication (Google, GitHub)
// @description - Anonymous session support
// @description - Save and share Strudel patterns

// @contact.name API Support
// @contact.url https://codeberg.org/algorave/server

// @license.name GPL-3.0
// @license.url https://www.gnu.org/licenses/gpl-3.0.html

// @host algorave.dev

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT token for authenticated requests. Format: Bearer {token}

func main() {
	logger.Info("starting algorave server")

	// load configuration from environment
	cfg, err := config.LoadEnvironmentVariables()
	if err != nil {
		logger.Fatal("failed to load configuration", "error", err)
	}

	// initialize OAuth providers
	if err := auth.InitializeProviders(); err != nil {
		logger.Fatal("failed to initialize OAuth providers", "error", err)
	}

	// create server with all dependencies
	srv, err := NewServer(cfg)
	if err != nil {
		logger.Fatal("failed to create server", "error", err)
	}

	// get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      srv.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// start server in goroutine
	go func() {
		logger.Info("server listening", "port", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed to start", "error", err)
		}
	}()

	// start websocket hub
	go srv.hub.Run()

	// start buffer flusher (Redis → Postgres)
	srv.flusher.Start()

	// start session cleanup service with cancellable context
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	go srv.cleanupService.Start(cleanupCtx)

	// wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// stop cleanup service
	cleanupCancel()

	logger.Info("shutting down server")

	// notify websocket clients and close connections first
	srv.hub.Shutdown()

	// stop flusher (flushes remaining data before stopping)
	srv.flusher.Stop()

	// graceful shutdown with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	// close validator if running
	if srv.services.Validator != nil {
		srv.services.Validator.Close() //nolint:errcheck,gosec // best-effort cleanup on shutdown
	}

	// close Redis connection
	srv.buffer.Close() //nolint:errcheck,gosec // best-effort cleanup on shutdown

	// close database connection
	srv.db.Close()

	logger.Info("server stopped")
}
