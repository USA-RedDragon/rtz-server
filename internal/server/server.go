package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	websocketControllers "github.com/USA-RedDragon/rtz-server/internal/server/websocket"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type Server struct {
	ipv4Server        *http.Server
	ipv6Server        *http.Server
	metricsIPV4Server *http.Server
	metricsIPV6Server *http.Server
	stopped           atomic.Bool
	config            *config.Config
}

const defTimeout = 120 * time.Second

type Router struct {
	*gin.Engine
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if strings.HasSuffix(req.URL.Path, "/") {
		req.URL.Path = filepath.Clean(req.URL.Path)
	}
	r.Engine.ServeHTTP(w, req)
}

func NewServer(config *config.Config, db *gorm.DB, redis *redis.Client) *Server {
	gin.SetMode(gin.ReleaseMode)
	if config.HTTP.PProf.Enabled {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	skipContextPathRouter := Router{
		Engine: r,
	}

	if config.HTTP.PProf.Enabled {
		pprof.Register(r)
	}

	writeTimeout := defTimeout

	rpcWebsocket := websocketControllers.CreateRPCWebsocket()
	applyMiddleware(r, config, "api", db, rpcWebsocket, redis)
	applyRoutes(r, config, rpcWebsocket)

	var metricsIPV4Server *http.Server
	var metricsIPV6Server *http.Server

	if config.HTTP.Metrics.Enabled {
		metricsRouter := gin.New()
		applyMiddleware(metricsRouter, config, "metrics", db, rpcWebsocket, redis)

		metricsRouter.GET("/metrics", gin.WrapH(promhttp.Handler()))
		metricsIPV4Server = &http.Server{
			Addr:              fmt.Sprintf("%s:%d", config.HTTP.Metrics.IPV4Host, config.HTTP.Metrics.Port),
			ReadHeaderTimeout: defTimeout,
			WriteTimeout:      writeTimeout,
			Handler:           metricsRouter,
		}
		metricsIPV6Server = &http.Server{
			Addr:              fmt.Sprintf("[%s]:%d", config.HTTP.Metrics.IPV6Host, config.HTTP.Metrics.Port),
			ReadHeaderTimeout: defTimeout,
			WriteTimeout:      defTimeout,
			Handler:           metricsRouter,
		}
	}

	return &Server{
		ipv4Server: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", config.HTTP.IPV4Host, config.HTTP.Port),
			ReadHeaderTimeout: defTimeout,
			WriteTimeout:      writeTimeout,
			Handler:           &skipContextPathRouter,
		},
		ipv6Server: &http.Server{
			Addr:              fmt.Sprintf("[%s]:%d", config.HTTP.IPV6Host, config.HTTP.Port),
			ReadHeaderTimeout: defTimeout,
			WriteTimeout:      defTimeout,
			Handler:           &skipContextPathRouter,
		},
		metricsIPV4Server: metricsIPV4Server,
		metricsIPV6Server: metricsIPV6Server,
		config:            config,
	}
}

func (s *Server) Start() error {
	waitGrp := sync.WaitGroup{}
	if s.ipv4Server != nil {
		ipv4Listener, err := net.Listen("tcp4", s.ipv4Server.Addr)
		if err != nil {
			return err
		}
		waitGrp.Add(1)
		go func() {
			defer waitGrp.Done()
			if err := s.ipv4Server.Serve(ipv4Listener); err != nil && !s.stopped.Load() {
				slog.Error("HTTP IPv4 server error", "error", err.Error())
			}
		}()
	}

	if s.ipv6Server != nil {
		ipv6Listener, err := net.Listen("tcp6", s.ipv6Server.Addr)
		if err != nil {
			return err
		}
		waitGrp.Add(1)
		go func() {
			defer waitGrp.Done()
			if err := s.ipv6Server.Serve(ipv6Listener); err != nil && !s.stopped.Load() {
				slog.Error("HTTP IPv6 server error", "error", err.Error())
			}
		}()
	}
	slog.Info("HTTP server started", "ipv4", s.config.HTTP.IPV4Host, "ipv6", s.config.HTTP.IPV6Host, "port", s.config.HTTP.Port)

	if s.config.HTTP.Metrics.Enabled {
		if s.metricsIPV4Server != nil {
			metricsIPV4Listener, err := net.Listen("tcp4", s.metricsIPV4Server.Addr)
			if err != nil {
				return err
			}
			waitGrp.Add(1)
			go func() {
				defer waitGrp.Done()
				if err := s.metricsIPV4Server.Serve(metricsIPV4Listener); err != nil && !s.stopped.Load() {
					slog.Error("Metrics IPv4 server error", "error", err.Error())
				}
			}()
		}

		if s.metricsIPV6Server != nil {
			metricsIPV6Listener, err := net.Listen("tcp6", s.metricsIPV6Server.Addr)
			if err != nil {
				return err
			}
			waitGrp.Add(1)
			go func() {
				defer waitGrp.Done()
				if err := s.metricsIPV6Server.Serve(metricsIPV6Listener); err != nil && !s.stopped.Load() {
					slog.Error("Metrics IPv6 server error", "error", err.Error())
				}
			}()
		}
		slog.Info("Metrics server started", "ipv4", s.config.HTTP.Metrics.IPV4Host, "ipv6", s.config.HTTP.Metrics.IPV6Host, "port", s.config.HTTP.Metrics.Port)
	}

	go func() {
		waitGrp.Wait()
	}()
	return nil
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	s.stopped.Store(true)

	errGrp := errgroup.Group{}
	if s.ipv4Server != nil {
		errGrp.Go(func() error {
			return s.ipv4Server.Shutdown(ctx)
		})
	}
	if s.ipv6Server != nil {
		errGrp.Go(func() error {
			return s.ipv6Server.Shutdown(ctx)
		})
	}
	if s.metricsIPV4Server != nil {
		errGrp.Go(func() error {
			return s.metricsIPV4Server.Shutdown(ctx)
		})
	}
	if s.metricsIPV6Server != nil {
		errGrp.Go(func() error {
			return s.metricsIPV6Server.Shutdown(ctx)
		})
	}

	return errGrp.Wait()
}
