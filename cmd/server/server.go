package main

import (
	"context"
	"fmt"
	"time"

	"github.com/algoraveai/server/algorave/sessions"
	"github.com/algoraveai/server/algorave/strudels"
	"github.com/algoraveai/server/algorave/users"
	"github.com/algoraveai/server/internal/config"
	ws "github.com/algoraveai/server/internal/websocket"
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
	hub.RegisterHandler(ws.TypeCodeUpdate, ws.CodeUpdateHandler(sessionRepo))
	hub.RegisterHandler(ws.TypeAgentRequest, ws.GenerateHandler(services.Agent, sessionRepo, userRepo))
	hub.RegisterHandler(ws.TypeChatMessage, ws.ChatHandler(sessionRepo))

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
