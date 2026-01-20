package main

import (
	"codeberg.org/algopatterns/server/algopatterns/sessions"
	"codeberg.org/algopatterns/server/algopatterns/strudels"
	"codeberg.org/algopatterns/server/algopatterns/users"
	"codeberg.org/algopatterns/server/internal/agent"
	"codeberg.org/algopatterns/server/internal/attribution"
	"codeberg.org/algopatterns/server/internal/botdefense"
	"codeberg.org/algopatterns/server/internal/buffer"
	"codeberg.org/algopatterns/server/internal/config"
	"codeberg.org/algopatterns/server/internal/llm"
	"codeberg.org/algopatterns/server/internal/retriever"
	"codeberg.org/algopatterns/server/internal/storage"
	"codeberg.org/algopatterns/server/internal/strudel"
	ws "codeberg.org/algopatterns/server/internal/websocket"
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
