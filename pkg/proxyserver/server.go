package proxyserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/factory"
	"github.com/ideamans/chatbotgate/pkg/logging"
)

// Server represents the proxy server with middleware
type Server struct {
	cfg     *config.Config
	handler http.Handler
	logger  logging.Logger
	host    string
	port    int
}

// New creates a new Server from configuration file
func New(configPath string, host string, port int, logger logging.Logger) (*Server, error) {
	// Load configuration
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewFromConfig(cfg, host, port, logger)
}

// NewFromConfig creates a new Server from a Config object
func NewFromConfig(cfg *config.Config, host string, port int, logger logging.Logger) (*Server, error) {
	if logger == nil {
		logger = logging.NewSimpleLogger("proxyserver", logging.LevelInfo, true)
	}

	// Create factory for building middleware components
	f := factory.NewDefaultFactory(host, port, logger)

	// Create KVS stores
	sessionKVS, tokenKVS, rateLimitKVS, err := f.CreateKVSStores(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create KVS stores: %w", err)
	}

	// Create session store
	sessionStore := f.CreateSessionStore(sessionKVS)

	// Create proxy handler directly (not via factory)
	proxyHandler, err := NewHandlerWithConfig(cfg.Proxy.Upstream, cfg.Proxy.Hosts)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy handler: %w", err)
	}
	logger.Debug("Proxy handler initialized", "upstream", cfg.Proxy.Upstream.URL)

	// Build middleware using factory
	middleware, err := f.CreateMiddleware(cfg, sessionStore, proxyHandler, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create middleware: %w", err)
	}

	// Note: tokenKVS and rateLimitKVS are kept alive for the lifetime of the middleware
	// They will be closed when the middleware is garbage collected
	_ = tokenKVS
	_ = rateLimitKVS

	return &Server{
		cfg:     cfg,
		handler: middleware,
		logger:  logger,
		host:    host,
		port:    port,
	}, nil
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	s.logger.Info("Starting server", "addr", addr, "upstream", s.cfg.Proxy.Upstream.URL)

	server := &http.Server{
		Addr:    addr,
		Handler: s.handler,
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
	return s.handler
}
