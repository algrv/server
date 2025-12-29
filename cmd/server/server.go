package main

import (
	"context"
	"fmt"

	"github.com/algorave/server/algorave/anonsessions"
	"github.com/algorave/server/algorave/sessions"
	"github.com/algorave/server/algorave/strudels"
	"github.com/algorave/server/algorave/users"
	"github.com/algorave/server/internal/config"
	ws "github.com/algorave/server/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// creates and configures a new server instance with all dependencies
func NewServer(cfg *config.Config) (*Server, error) {
	ctx := context.Background()

	db, err := pgxpool.New(ctx, cfg.SupabaseConnString)
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
	sessionMgr := anonsessions.NewManager()

	services, err := InitializeServices(cfg, db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	hub := ws.NewHub()
	router := gin.Default()

	server := &Server{
		db:          db,
		config:      cfg,
		userRepo:    userRepo,
		strudelRepo: strudelRepo,
		sessionRepo: sessionRepo,
		sessionMgr:  sessionMgr,
		services:    services,
		hub:         hub,
		router:      router,
	}

	RegisterRoutes(router, server)

	return server, nil
}
