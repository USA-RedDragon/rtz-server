package websocket

import (
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/gin-gonic/gin"
)

// CreateHandlerWithTimings is CreateHandler with custom ping timings.
func CreateHandlerWithTimings(ws Websocket, config *config.Config, pongWait, pingPeriod time.Duration) func(*gin.Context) {
	return createHandler(ws, config, pongWait, pingPeriod)
}
