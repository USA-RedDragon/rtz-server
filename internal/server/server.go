package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/events"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	ipv4Server        *http.Server
	ipv6Server        *http.Server
	metricsIPV4Server *http.Server
	metricsIPV6Server *http.Server
	stopped           atomic.Bool
	config            *config.HTTP
}

const defTimeout = 5 * time.Second

func NewServer(config *config.HTTP, eventsChannel chan events.Event) *Server {
	gin.SetMode(gin.ReleaseMode)
	if config.PProf.Enabled {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()

	if config.PProf.Enabled {
		pprof.Register(r)
	}

	writeTimeout := defTimeout
	if config.PProf.Enabled {
		writeTimeout = 60 * time.Second
	}

	applyMiddleware(r, config, "api")
	applyRoutes(r, config, eventsChannel)

	var metricsIPV4Server *http.Server
	var metricsIPV6Server *http.Server

	if config.Metrics.Enabled {
		metricsRouter := gin.New()
		applyMiddleware(metricsRouter, config, "metrics")

		metricsRouter.GET("/metrics", gin.WrapH(promhttp.Handler()))
		metricsIPV4Server = &http.Server{
			Addr:              fmt.Sprintf("%s:%d", config.Metrics.IPV4Host, config.Metrics.Port),
			ReadHeaderTimeout: defTimeout,
			WriteTimeout:      writeTimeout,
			Handler:           metricsRouter,
		}
		metricsIPV6Server = &http.Server{
			Addr:              fmt.Sprintf("[%s]:%d", config.Metrics.IPV6Host, config.Metrics.Port),
			ReadHeaderTimeout: defTimeout,
			WriteTimeout:      defTimeout,
			Handler:           metricsRouter,
		}
	}

	return &Server{
		ipv4Server: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", config.IPV4Host, config.Port),
			ReadHeaderTimeout: defTimeout,
			WriteTimeout:      writeTimeout,
			Handler:           r,
		},
		ipv6Server: &http.Server{
			Addr:              fmt.Sprintf("[%s]:%d", config.IPV6Host, config.Port),
			ReadHeaderTimeout: defTimeout,
			WriteTimeout:      defTimeout,
			Handler:           r,
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
	slog.Info("HTTP server started", "ipv4", s.config.IPV4Host, "ipv6", s.config.IPV6Host, "port", s.config.Port)

	if s.config.Metrics.Enabled {
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
		slog.Info("Metrics server started", "ipv4", s.config.Metrics.IPV4Host, "ipv6", s.config.Metrics.IPV6Host, "port", s.config.Metrics.Port)
	}

	go func() {
		waitGrp.Wait()
	}()
	return nil
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
