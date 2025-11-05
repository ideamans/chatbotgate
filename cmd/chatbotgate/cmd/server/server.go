package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ideamans/chatbotgate/pkg/logging"
)

// Server represents the HTTP server that integrates proxy and middleware managers
type Server struct {
	middlewareManager MiddlewareManager
	logger            logging.Logger
	host              string
	port              int
}

// New creates a new Server from MiddlewareManager
func New(middlewareManager MiddlewareManager, host string, port int, logger logging.Logger) (*Server, error) {
	if logger == nil {
		logger = logging.NewSimpleLogger("server", logging.LevelInfo, true)
	}

	return &Server{
		middlewareManager: middlewareManager,
		logger:            logger,
		host:              host,
		port:              port,
	}, nil
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	s.logger.Info("Starting server", "addr", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: s.middlewareManager.Handler(),
	}

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down server...")
		if err := server.Shutdown(context.Background()); err != nil {
			s.logger.Error("Server shutdown error", "error", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Handler returns the HTTP handler (useful for testing)
func (s *Server) Handler() http.Handler {
	return s.middlewareManager.Handler()
}
