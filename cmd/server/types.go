package main

import (
	"codeberg.org/algorave/server/algorave/sessions"
	"codeberg.org/algorave/server/algorave/strudels"
	"codeberg.org/algorave/server/algorave/users"
	"codeberg.org/algorave/server/internal/agent"
	"codeberg.org/algorave/server/internal/attribution"
	"codeberg.org/algorave/server/internal/botdefense"
	"codeberg.org/algorave/server/internal/buffer"
	"codeberg.org/algorave/server/internal/config"
	"codeberg.org/algorave/server/internal/llm"
	"codeberg.org/algorave/server/internal/retriever"
	"codeberg.org/algorave/server/internal/storage"
	"codeberg.org/algorave/server/internal/strudel"
	ws "codeberg.org/algorave/server/internal/websocket"
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
