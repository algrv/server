package main

import (
	"github.com/algrv/server/algorave/sessions"
	"github.com/algrv/server/algorave/strudels"
	"github.com/algrv/server/algorave/users"
	"github.com/algrv/server/internal/agent"
	"github.com/algrv/server/internal/attribution"
	"github.com/algrv/server/internal/buffer"
	"github.com/algrv/server/internal/config"
	"github.com/algrv/server/internal/llm"
	"github.com/algrv/server/internal/notifications"
	"github.com/algrv/server/internal/retriever"
	"github.com/algrv/server/internal/storage"
	"github.com/algrv/server/internal/strudel"
	ws "github.com/algrv/server/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// holds all dependencies and state for the API server
type Server struct {
	db             *pgxpool.Pool
	config         *config.Config
	userRepo       *users.Repository
	strudelRepo    *strudels.Repository
	sessionRepo    sessions.Repository
	services       *Services
	hub            *ws.Hub
	router         *gin.Engine
	buffer         *buffer.SessionBuffer
	flusher        *buffer.Flusher
	cleanupService *sessions.CleanupService
	ccSignals      *CCSignalsSystem
}

// holds all external service clients (LLM, storage, retriever, agent)
type Services struct {
	Agent         *agent.Agent
	Attribution   *attribution.Service
	Notifications *notifications.Service
	LLM           llm.LLM
	Retriever     *retriever.Client
	Storage       *storage.Client
	Validator     *strudel.Validator
}
