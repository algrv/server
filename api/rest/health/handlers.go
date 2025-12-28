package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler godoc
// @Summary Health check
// @Description Get service health status
// @Tags health
// @Produce json
// @Success 200 {object} Response
// @Router /health [get]
func Handler(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Status:  "healthy",
		Service: "algorave",
		Version: "1.0.0",
	})
}

// PingHandler godoc
// @Summary Ping
// @Description Simple ping endpoint
// @Tags health
// @Produce json
// @Success 200 {object} PingResponse
// @Router /api/v1/ping [get]
func PingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, PingResponse{
		Message: "pong",
	})
}
