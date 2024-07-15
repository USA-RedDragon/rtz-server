package controllers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	"github.com/USA-RedDragon/rtz-server/internal/server/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

func HandleRPC(c *gin.Context) {
	dongleID, ok := c.Params.Get("dongle_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dongle_id is required"})
		return
	}

	config, ok := c.MustGet("config").(*config.Config)
	if !ok {
		slog.Error("Failed to get config from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	metrics, ok := c.MustGet("metrics").(*metrics.Metrics)
	if !ok {
		slog.Error("Failed to get metrics from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	maybeNats, ok := c.Get("nats")
	if !ok && config.NATS.Enabled {
		slog.Error("Failed to get NATS from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	nats, ok := maybeNats.(*nats.Conn)
	if !ok {
		nats = nil
	}

	var inboundCall apimodels.InboundRPCCall
	if err := c.BindJSON(&inboundCall); err != nil {
		slog.Error("Failed to bind RPC call", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	rpcCaller, ok := c.MustGet("rpcWebsocket").(*websocket.RPCWebsocket)
	if !ok {
		slog.Error("Failed to get rpc from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	// The frontend seemingly always provides a 0 id, but we need to track it through the system
	uuid, err := uuid.NewRandom()
	if err != nil {
		slog.Error("Failed to generate UUID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	call := apimodels.RPCCall{
		ID:             uuid.String(),
		Method:         inboundCall.Method,
		Params:         inboundCall.Params,
		JSONRPCVersion: inboundCall.JSONRPCVersion,
	}

	resp, err := rpcCaller.Call(nats, metrics, dongleID, call)
	if err != nil {
		if errors.Is(err, websocket.ErrNotConnected) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Dongle not connected"})
			return
		}
		slog.Error("Failed to call RPC", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, resp)
}
