package controllers

import (
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/connect-server/internal/server/apimodels"
	"github.com/gin-gonic/gin"
)

func HandleRPC(c *gin.Context) {
	dongleID, ok := c.Params.Get("dongle_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dongle_id is required"})
		return
	}

	var call apimodels.RPCCall
	if err := c.BindJSON(&call); err != nil {
		slog.Error("Failed to bind RPC call", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	slog.Info("RPC", "dongle_id", dongleID, "method", call.Method, "params", call.Params)
	c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
}
