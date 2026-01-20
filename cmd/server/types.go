package main

import (
	"codeberg.org/algojams/server/algojams/sessions"
	"codeberg.org/algojams/server/algojams/strudels"
	"codeberg.org/algojams/server/algojams/users"
	"codeberg.org/algojams/server/internal/agent"
	"codeberg.org/algojams/server/internal/attribution"
	"codeberg.org/algojams/server/internal/botdefense"
	"codeberg.org/algojams/server/internal/buffer"
	"codeberg.org/algojams/server/internal/config"
	"codeberg.org/algojams/server/internal/llm"
	"codeberg.org/algojams/server/internal/retriever"
	"codeberg.org/algojams/server/internal/storage"
	"codeberg.org/algojams/server/internal/strudel"
	ws "codeberg.org/algojams/server/internal/websocket"
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
	botDefense     *botdefense.Defense
}

// holds all external service clients (LLM, storage, retriever, agent)
type Services struct {
	Agent       *agent.Agent
	Attribution *attribution.Service
	LLM         llm.LLM
	Retriever   *retriever.Client
	Storage     *storage.Client
	Validator   *strudel.Validator
}
