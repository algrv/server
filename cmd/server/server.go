package main

import (
	"context"
	"fmt"
	"time"

	"github.com/algrv/server/algorave/sessions"
	"github.com/algrv/server/algorave/strudels"
	"github.com/algrv/server/algorave/users"
	"github.com/algrv/server/internal/config"
	"github.com/algrv/server/internal/logger"
	ws "github.com/algrv/server/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	sessionRepo := sessions.NewRepository(db)

	services, err := InitializeServices(cfg, db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	hub := ws.NewHub()

	// register websocket message handlers
	hub.RegisterHandler(ws.TypeCodeUpdate, ws.CodeUpdateHandler())
	hub.RegisterHandler(ws.TypeAutoSave, ws.AutoSaveHandler(sessionRepo))
	hub.RegisterHandler(ws.TypeAgentRequest, ws.GenerateHandler(services.Agent, sessionRepo, userRepo))
	hub.RegisterHandler(ws.TypeChatMessage, ws.ChatHandler(sessionRepo))
	hub.RegisterHandler(ws.TypePlay, ws.PlayHandler())
	hub.RegisterHandler(ws.TypeStop, ws.StopHandler())
	hub.RegisterHandler(ws.TypeSwitchStrudel, ws.SwitchStrudelHandler(strudelRepo))

	// save code on client disconnect
	hub.OnClientDisconnect(func(client *ws.Client) {
		if !client.CanWrite() {
			return
		}

		lastCode := client.GetLastCode()
		if lastCode == "" {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := sessionRepo.UpdateSessionCode(ctx, client.SessionID, lastCode); err != nil {
			logger.ErrorErr(err, "failed to save code on disconnect",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
		} else {
			logger.Debug("code saved on disconnect",
				"client_id", client.ID,
				"session_id", client.SessionID,
			)
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
	}

	RegisterRoutes(router, server)

	return server, nil
}
