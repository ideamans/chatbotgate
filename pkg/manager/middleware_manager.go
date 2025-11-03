package manager

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/factory"
	"github.com/ideamans/chatbotgate/pkg/logging"
	"github.com/ideamans/chatbotgate/pkg/middleware"
	"github.com/ideamans/chatbotgate/pkg/proxy"
	"github.com/ideamans/chatbotgate/pkg/session"
)

// SingleDomainManager manages a single middleware instance with dynamic reloading support.
// It holds the current middleware instance and can atomically swap it with a new one
// when the configuration changes. This is the basic manager for single-domain setups.
type SingleDomainManager struct {
	// Current middleware instance (atomic)
	current atomic.Value // *middleware.Middleware

	// Factory for creating middleware instances
	factory factory.Factory

	// Shared resources that are NOT reloaded
	sessionStore session.Store
	proxyHandler *proxy.Handler

	// Current configuration
	config *config.Config
	mu     sync.RWMutex // Protects config during reload

	logger logging.Logger
}

// ManagerConfig contains the configuration for creating a MiddlewareManager
type ManagerConfig struct {
	Config       *config.Config
	Factory      factory.Factory  // Factory for creating middleware (optional - will use default if nil)
	SessionStore session.Store
	ProxyHandler *proxy.Handler
	Logger       logging.Logger
}

// New creates a new SingleDomainManager with the given configuration.
func New(cfg ManagerConfig) (*SingleDomainManager, error) {
	if cfg.Config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.SessionStore == nil {
		return nil, fmt.Errorf("session store is required")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	logger := cfg.Logger.WithModule("manager")

	// Validate configuration before creating middleware
	if errs := middleware.ValidateConfig(cfg.Config); len(errs) > 0 {
		// Log each validation error
		for _, ve := range errs {
			logger.Error("Configuration validation error", "field", ve.Field, "message", ve.Message)
		}
		return nil, errs
	}

	// Use provided factory or create default factory
	// Note: We can't create DefaultFactory here without host/port
	// So factory must be provided by caller
	if cfg.Factory == nil {
		return nil, fmt.Errorf("factory is required")
	}

	manager := &SingleDomainManager{
		factory:      cfg.Factory,
		sessionStore: cfg.SessionStore,
		proxyHandler: cfg.ProxyHandler,
		config:       cfg.Config,
		logger:       logger,
	}

	// Create initial middleware instance using factory
	mw, err := cfg.Factory.CreateMiddleware(
		cfg.Config,
		cfg.SessionStore,
		cfg.ProxyHandler,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial middleware: %w", err)
	}

	manager.current.Store(mw)
	manager.logger.Info("Middleware manager initialized successfully")

	return manager, nil
}

// ServeHTTP implements http.Handler by delegating to the current middleware instance.
func (m *SingleDomainManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mw := m.current.Load().(*middleware.Middleware)
	mw.ServeHTTP(w, r)
}

// Reload reloads the middleware with a new configuration.
// It creates a new middleware instance and atomically swaps it with the current one.
// If the new configuration is invalid or middleware creation fails, the old instance is kept.
// This is a convenience method - not part of any interface.
func (m *SingleDomainManager) Reload(newConfig *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debug("Starting middleware configuration reload")

	// Validate new configuration
	if errs := middleware.ValidateConfig(newConfig); len(errs) > 0 {
		// Log each validation error
		for _, ve := range errs {
			m.logger.Debug("Configuration validation error", "field", ve.Field, "message", ve.Message)
		}
		m.logger.Warn("Configuration reload failed: validation errors detected, keeping current configuration")
		return errs
	}

	// Create new middleware instance using factory
	newMw, err := m.factory.CreateMiddleware(
		newConfig,
		m.sessionStore,
		m.proxyHandler,
		m.logger,
	)
	if err != nil {
		m.logger.Debug("Middleware creation failed", "error", err)
		m.logger.Error("Configuration reload failed: could not create new middleware")
		return fmt.Errorf("failed to create new middleware: %w", err)
	}

	// Atomic swap
	m.current.Store(newMw)
	m.config = newConfig

	m.logger.Info("Configuration reloaded successfully")
	return nil
}
