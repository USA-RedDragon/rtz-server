package server

import (
	"log/slog"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

func applyMiddleware(r *gin.Engine, config *config.HTTP, otelComponent string, db *gorm.DB) {
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.TrustedPlatform = "X-Real-IP"

	err := r.SetTrustedProxies(config.TrustedProxies)
	if err != nil {
		slog.Error("Failed to set trusted proxies", "error", err.Error())
	}

	r.Use(dbMiddleware(db))

	if config.Tracing.Enabled {
		r.Use(otelgin.Middleware(otelComponent))
		r.Use(tracingProvider(config))
	}
}

func tracingProvider(config *config.HTTP) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Tracing.OTLPEndpoint != "" {
			ctx := c.Request.Context()
			span := trace.SpanFromContext(ctx)
			if span.IsRecording() {
				span.SetAttributes(
					attribute.String("http.method", c.Request.Method),
					attribute.String("http.path", c.Request.URL.Path),
				)
			}
		}
		c.Next()
	}
}

func dbMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}
