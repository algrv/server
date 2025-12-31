package main

import (
	"github.com/algoraveai/server/algorave/sessions"
	"github.com/algoraveai/server/algorave/strudels"
	"github.com/algoraveai/server/algorave/users"
	"github.com/algoraveai/server/internal/agent"
	"github.com/algoraveai/server/internal/config"
	"github.com/algoraveai/server/internal/llm"
	"github.com/algoraveai/server/internal/retriever"
	"github.com/algoraveai/server/internal/storage"
	ws "github.com/algoraveai/server/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// holds all dependencies and state for the API server
type Server struct {
	db          *pgxpool.Pool
	config      *config.Config
	userRepo    *users.Repository
	strudelRepo *strudels.Repository
	sessionRepo sessions.Repository
	services    *Services
	hub         *ws.Hub
	router      *gin.Engine
}

// holds all external service clients (LLM, storage, retriever, agent)
type Services struct {
	Agent     *agent.Agent
	LLM       llm.LLM
	Retriever *retriever.Client
	Storage   *storage.Client
}
