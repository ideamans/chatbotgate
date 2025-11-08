package server

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/middleware/core"
	"github.com/ideamans/chatbotgate/pkg/middleware/factory"
	"github.com/ideamans/chatbotgate/pkg/shared/filewatcher"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// MiddlewareManager is an interface for managing middleware lifecycle
type MiddlewareManager interface {
	// Handler returns the HTTP handler that includes the middleware and proxies to the next handler
	Handler() http.Handler
}

// SimpleMiddlewareManager is a simple implementation of MiddlewareManager with hot reload support
type SimpleMiddlewareManager struct {
	middleware atomic.Value // Stores *middleware.Middleware
	configPath string
	host       string
	port       int
	next       http.Handler
	logger     logging.Logger
}

// NewMiddlewareManager creates a new SimpleMiddlewareManager from config file
func NewMiddlewareManager(configPath string, host string, port int, next http.Handler, logger logging.Logger) (*SimpleMiddlewareManager, error) {
	if logger == nil {
		logger = logging.NewSimpleLogger("middleware-manager", logging.LevelInfo, true)
	}

	m := &SimpleMiddlewareManager{
		configPath: configPath,
		host:       host,
		port:       port,
		next:       next,
		logger:     logger,
	}

	// Build initial middleware
	mw, err := m.buildMiddleware(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build initial middleware: %w", err)
	}

	// Store initial middleware atomically
	m.middleware.Store(mw)

	logger.Info("Middleware manager initialized", "config_path", configPath)

	return m, nil
}

// buildMiddleware builds middleware from configuration file
func (m *SimpleMiddlewareManager) buildMiddleware(configPath string) (*middleware.Middleware, error) {
	// Load middleware configuration from YAML
	cfg, err := config.NewFileLoader(configPath).Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load middleware config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("middleware config validation failed: %w", err)
	}

	m.logger.Debug("Middleware configuration loaded and validated", "config_path", configPath)

	// Create factory for building middleware components
	f := factory.NewDefaultFactory(m.host, m.port, m.logger)

	// Create KVS stores
	sessionKVS, tokenKVS, rateLimitKVS, err := f.CreateKVSStores(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create KVS stores: %w", err)
	}

	// Create session store
	sessionStore := f.CreateSessionStore(sessionKVS)

	// Create middleware using factory with KVS stores
	mw, err := f.CreateMiddleware(cfg, sessionStore, tokenKVS, rateLimitKVS, m.next, m.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create middleware: %w", err)
	}

	return mw, nil
}

// OnFileChange implements filewatcher.ChangeListener interface
// This method is called when the configuration file changes
func (m *SimpleMiddlewareManager) OnFileChange(event filewatcher.ChangeEvent) {
	if event.Error != nil {
		m.logger.Error("File change event error", "error", event.Error)
		return
	}

	m.logger.Info("Config content change detected, starting reload", "path", event.Path, "component", "middleware")
	m.reload(event.Path)
}

// reload reloads the middleware configuration and replaces the current middleware atomically
func (m *SimpleMiddlewareManager) reload(configPath string) {
	// Build new middleware
	newMiddleware, err := m.buildMiddleware(configPath)
	if err != nil {
		m.logger.Error("Failed to reload middleware", "error", err, "path", configPath)
		m.logger.Error("Keeping current middleware configuration")
		return
	}

	// Atomically replace the middleware
	m.middleware.Store(newMiddleware)
	m.logger.Info("Configuration reloaded successfully", "component", "middleware")
}

// Handler returns the HTTP handler
// The handler always uses the latest middleware stored atomically
func (m *SimpleMiddlewareManager) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Load the current middleware atomically
		mw := m.middleware.Load().(*middleware.Middleware)
		mw.ServeHTTP(w, r)
	})
}
