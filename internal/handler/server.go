package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	apiv1 "github.com/houssemlou/fizz/internal/api/v1"
	"github.com/houssemlou/fizz/internal/middleware"
)

type Server struct {
	httpServer *http.Server
}

func New(addr, env, apiKey string, h *Handler) *Server {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID(env))
	router.Use(middleware.Metrics())
	router.Use(middleware.RequestLogger())

	if env != "dev" && apiKey != "" {
		router.Use(middleware.APIKey(apiKey))
	}

	strict := apiv1.NewStrictHandler(h, nil)
	apiv1.RegisterHandlersWithOptions(router, strict, apiv1.GinServerOptions{
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			c.JSON(statusCode, gin.H{"error": err.Error()})
		},
	})

	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}

func (s *Server) Start() error {
	slog.Info("server starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
