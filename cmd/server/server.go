package main

import (
	"context"
	"fmt"
	"time"

	"github.com/algrv/server/algorave/sessions"
	"github.com/algrv/server/algorave/strudels"
	"github.com/algrv/server/algorave/users"
	"github.com/algrv/server/internal/buffer"
	"github.com/algrv/server/internal/config"
	"github.com/algrv/server/internal/logger"
	ws "github.com/algrv/server/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// how often the flusher writes buffered data to Postgres
	bufferFlushInterval = 5 * time.Second
)

// creates and configures a new server instance with all dependencies
func NewServer(cfg *config.Config) (*Server, error) {
	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(cfg.SupabaseConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// configure connection pool for supabase free tier pooler compatibility
	// free tier has ~10-15 pooler connections, so keep our pool small
	poolConfig.MaxConns = 5
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// CRITICAL: use simple protocol for supabase pooler (PgBouncer) compatibility
	// pgBouncer in transaction mode doesn't support prepared statements,
	// which causes connections to hang on subsequent queries
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	userRepo := users.NewRepository(db)
	strudelRepo := strudels.NewRepository(db)
	postgresSessionRepo := sessions.NewRepository(db)

	// initialize Redis buffer for WebSocket write operations
	sessionBuffer, err := buffer.NewSessionBuffer(cfg.RedisURL, bufferFlushInterval)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize redis buffer: %w", err)
	}

	// wrap session repo with buffering layer (writes go to Redis, reads go to Postgres)
	sessionRepo := buffer.NewBufferedRepository(postgresSessionRepo, sessionBuffer)

	// create flusher to periodically persist buffered data to Postgres
	flusher := buffer.NewFlusher(sessionBuffer, postgresSessionRepo, bufferFlushInterval)

	services, err := InitializeServices(cfg, db)
	if err != nil {
		sessionBuffer.Close() //nolint:errcheck,gosec // best-effort cleanup on init failure
		db.Close()
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	hub := ws.NewHub()

	// register websocket message handlers (handlers use sessionRepo interface, unaware of Redis)
	hub.RegisterHandler(ws.TypeCodeUpdate, ws.CodeUpdateHandler(sessionRepo, sessionBuffer, strudelRepo))
	hub.RegisterHandler(ws.TypeChatMessage, ws.ChatHandler(sessionRepo))
	hub.RegisterHandler(ws.TypePlay, ws.PlayHandler())
	hub.RegisterHandler(ws.TypeStop, ws.StopHandler())
	hub.RegisterHandler(ws.TypePing, ws.PingHandler())

	// flush buffer on client disconnect
	hub.OnClientDisconnect(func(client *ws.Client) {
		if !client.CanWrite() {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := flusher.FlushSession(ctx, client.SessionID); err != nil {
			logger.ErrorErr(err, "failed to flush buffer on disconnect",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
		} else {
			logger.Debug("buffer flushed on disconnect",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
		}
	})

	// send paste lock status on client connect (for session reconnects)
	hub.OnClientRegistered(func(client *ws.Client) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		locked, err := sessionBuffer.IsPasteLocked(ctx, client.SessionID)
		if err != nil {
			return // ignore errors, not critical
		}

		if locked {
			payload := ws.PasteLockChangedPayload{
				Locked: true,
				Reason: "session_reconnect",
			}

			msg, err := ws.NewMessage(ws.TypePasteLockChanged, client.SessionID, client.UserID, payload)
			if err != nil {
				return
			}

			client.Send(msg) //nolint:errcheck,gosec // best-effort
		}
	})

	router := gin.Default()

	server := &Server{
		db:          db,
		config:      cfg,
		userRepo:    userRepo,
		strudelRepo: strudelRepo,
		sessionRepo: sessionRepo,
		services:    services,
		hub:         hub,
		router:      router,
		buffer:      sessionBuffer,
		flusher:     flusher,
	}

	RegisterRoutes(router, server)

	return server, nil
}
