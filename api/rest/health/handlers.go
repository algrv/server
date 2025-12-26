package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents the health check response
type Response struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version,omitempty"`
}

// returns the server health status
func Handler(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Status:  "healthy",
		Service: "algorave",
		Version: "1.0.0",
	})
}

// responds with pong for testing
func PingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
