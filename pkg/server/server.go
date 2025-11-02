package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ideamans/chatbotgate/pkg/auth/email"
	"github.com/ideamans/chatbotgate/pkg/auth/oauth2"
	"github.com/ideamans/chatbotgate/pkg/authz"
	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/i18n"
	"github.com/ideamans/chatbotgate/pkg/logging"
	"github.com/ideamans/chatbotgate/pkg/middleware"
	"github.com/ideamans/chatbotgate/pkg/proxy"
	"github.com/ideamans/chatbotgate/pkg/session"
)

// Server represents a simplified HTTP server that wraps the auth middleware
type Server struct {
	config       *config.Config
	host         string
	port         int
	handler      http.Handler
	httpServer   *http.Server
	logger       logging.Logger
}

// New creates a new server instance
// The server can operate in two modes:
// 1. Proxy mode (with proxyHandler): Auth middleware + Reverse proxy
// 2. Server mode (without proxyHandler): Auth middleware only
func New(
	cfg *config.Config,
	host string,
	port int,
	sessionStore session.Store,
	oauthManager *oauth2.Manager,
	emailHandler *email.Handler,
	authzChecker authz.Checker,
	proxyHandler *proxy.Handler,
	logger logging.Logger,
) *Server {
	translator := i18n.NewTranslator()

	// Create the auth middleware
	authMiddleware := middleware.New(
		cfg,
		sessionStore,
		oauthManager,
		emailHandler,
		authzChecker,
		translator,
		logger.WithModule("middleware"),
	)

	var handler http.Handler

	if proxyHandler != nil {
		// Proxy mode: middleware wraps the proxy
		handler = authMiddleware.Wrap(proxyHandler)
		logger.Info("Server configured in proxy mode (auth + reverse proxy)")
	} else {
		// Server mode: middleware only
		handler = authMiddleware.Wrap(nil)
		logger.Info("Server configured in server mode (auth only)")
	}

	return &Server{
		config:  cfg,
		host:    host,
		port:    port,
		handler: handler,
		logger:  logger.WithModule("server"),
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting server", "addr", addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server")
	return s.httpServer.Shutdown(ctx)
}

// Handler returns the HTTP handler (for testing)
func (s *Server) Handler() http.Handler {
	return s.handler
}
