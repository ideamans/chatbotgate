package server

import (
	"fmt"
	"net/http"

	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/factory"
	"github.com/ideamans/chatbotgate/pkg/logging"
	"github.com/ideamans/chatbotgate/pkg/middleware"
)

// MiddlewareManager is an interface for managing middleware lifecycle
type MiddlewareManager interface {
	// Handler returns the HTTP handler that includes the middleware and proxies to the next handler
	Handler() http.Handler
}

// SimpleMiddlewareManager is a simple implementation of MiddlewareManager
type SimpleMiddlewareManager struct {
	middleware *middleware.Middleware
	logger     logging.Logger
}

// NewMiddlewareManager creates a new SimpleMiddlewareManager from config file
func NewMiddlewareManager(configPath string, host string, port int, next http.Handler, logger logging.Logger) (*SimpleMiddlewareManager, error) {
	if logger == nil {
		logger = logging.NewSimpleLogger("middleware-manager", logging.LevelInfo, true)
	}

	// Load middleware configuration from YAML
	cfg, err := config.NewFileLoader(configPath).Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load middleware config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("middleware config validation failed: %w", err)
	}

	logger.Debug("Middleware configuration loaded and validated", "config_path", configPath)

	// Create factory for building middleware components
	f := factory.NewDefaultFactory(host, port, logger)

	// Create KVS stores
	sessionKVS, tokenKVS, rateLimitKVS, err := f.CreateKVSStores(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create KVS stores: %w", err)
	}

	// Create session store
	sessionStore := f.CreateSessionStore(sessionKVS)

	// Create middleware using factory
	mw, err := f.CreateMiddleware(cfg, sessionStore, next, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create middleware: %w", err)
	}

	// Keep KVS stores alive
	_ = tokenKVS
	_ = rateLimitKVS

	return &SimpleMiddlewareManager{
		middleware: mw,
		logger:     logger,
	}, nil
}

// Handler returns the HTTP handler
func (m *SimpleMiddlewareManager) Handler() http.Handler {
	return m.middleware
}
