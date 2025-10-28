package httpserver

import (
	"context"
	"net/http"
	"strings"
	"time"

	"backoffice/backend/internal/config"
	authusecase "backoffice/backend/internal/usecase/auth"
	productusecase "backoffice/backend/internal/usecase/product"
)

// Server wraps the HTTP server lifecycle.
type Server struct {
	httpServer     *http.Server
	router         *http.ServeMux
	authService    *authusecase.Service
	productService *productusecase.Service
	allowedOrigins []string
	addr           string
}

// NewServer constructs a new Server with configured dependencies.
func NewServer(cfg config.Config, authService *authusecase.Service, productService *productusecase.Service) *Server {
	mux := http.NewServeMux()
	addr := cfg.HTTPPort
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}

	handler := withLogging(withCORS(mux, cfg.AllowedOrigins))

	srv := &Server{
		httpServer: &http.Server{
			Handler:      handler,
			ReadTimeout:  time.Duration(cfg.ReadTimeoutSec) * time.Second,
			WriteTimeout: time.Duration(cfg.WriteTimeoutSec) * time.Second,
			IdleTimeout:  time.Duration(cfg.IdleTimeoutSec) * time.Second,
		},
		router:         mux,
		authService:    authService,
		productService: productService,
		allowedOrigins: cfg.AllowedOrigins,
		addr:           addr,
	}
	srv.httpServer.Addr = addr
	srv.registerRoutes()
	return srv
}

// Start bootstraps the HTTP server on the provided address.
func (s *Server) Start() error {
	s.httpServer.Addr = s.addr
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Router exposes the underlying ServeMux so routes can be registered.
func (s *Server) Router() *http.ServeMux {
	return s.router
}

// Addr returns the configured network address for the HTTP server.
func (s *Server) Addr() string {
	return s.addr
}
