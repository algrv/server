package main

import (
	"github.com/algorave/server/algorave/anonsessions"
	"github.com/algorave/server/algorave/sessions"
	"github.com/algorave/server/algorave/strudels"
	"github.com/algorave/server/algorave/users"
	"github.com/algorave/server/internal/agent"
	"github.com/algorave/server/internal/config"
	"github.com/algorave/server/internal/llm"
	"github.com/algorave/server/internal/retriever"
	"github.com/algorave/server/internal/storage"
	ws "github.com/algorave/server/internal/websocket"
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
	sessionMgr  *anonsessions.Manager
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
