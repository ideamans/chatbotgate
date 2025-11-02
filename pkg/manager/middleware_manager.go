package manager

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/ideamans/multi-oauth2-proxy/pkg/auth/email"
	"github.com/ideamans/multi-oauth2-proxy/pkg/auth/oauth2"
	"github.com/ideamans/multi-oauth2-proxy/pkg/authz"
	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
	"github.com/ideamans/multi-oauth2-proxy/pkg/i18n"
	"github.com/ideamans/multi-oauth2-proxy/pkg/logging"
	"github.com/ideamans/multi-oauth2-proxy/pkg/middleware"
	"github.com/ideamans/multi-oauth2-proxy/pkg/proxy"
	"github.com/ideamans/multi-oauth2-proxy/pkg/session"
)

// MiddlewareManager manages middleware instances and supports dynamic reloading.
// It holds the current middleware instance and can atomically swap it with a new one
// when the configuration changes.
type MiddlewareManager struct {
	// Current middleware instance (atomic)
	current atomic.Value // *middleware.Middleware

	// Shared resources that are NOT reloaded
	sessionStore session.Store
	proxyHandler *proxy.Handler

	// Server configuration (host/port)
	host string
	port int

	// Current configuration
	config *config.Config
	mu     sync.RWMutex // Protects config during reload

	logger logging.Logger
}

// ManagerConfig contains the configuration for creating a MiddlewareManager
type ManagerConfig struct {
	Config       *config.Config
	Host         string
	Port         int
	SessionStore session.Store
	ProxyHandler *proxy.Handler
	Logger       logging.Logger
}

// New creates a new MiddlewareManager with the given configuration.
func New(cfg ManagerConfig) (*MiddlewareManager, error) {
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

	manager := &MiddlewareManager{
		sessionStore: cfg.SessionStore,
		proxyHandler: cfg.ProxyHandler,
		host:         cfg.Host,
		port:         cfg.Port,
		config:       cfg.Config,
		logger:       logger,
	}

	// Create initial middleware instance
	mw, err := manager.createMiddleware(cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial middleware: %w", err)
	}

	manager.current.Store(mw)
	manager.logger.Info("MiddlewareManager initialized")

	return manager, nil
}

// ServeHTTP implements http.Handler by delegating to the current middleware instance.
func (m *MiddlewareManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mw := m.current.Load().(*middleware.Middleware)
	mw.ServeHTTP(w, r)
}

// Reload reloads the middleware with a new configuration.
// It creates a new middleware instance and atomically swaps it with the current one.
// If the new configuration is invalid or middleware creation fails, the old instance is kept.
func (m *MiddlewareManager) Reload(newConfig *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Reloading middleware configuration")

	// Validate new configuration
	if errs := middleware.ValidateConfig(newConfig); len(errs) > 0 {
		// Log each validation error as a warning (not fatal for reload)
		m.logger.Warn("Configuration validation failed, keeping current configuration")
		for _, ve := range errs {
			m.logger.Warn("Validation error", "field", ve.Field, "message", ve.Message)
		}
		return errs
	}

	// Create new middleware instance
	newMw, err := m.createMiddleware(newConfig)
	if err != nil {
		m.logger.Error("Failed to create new middleware", "error", err)
		return fmt.Errorf("failed to create new middleware: %w", err)
	}

	// Atomic swap
	m.current.Store(newMw)
	m.config = newConfig

	m.logger.Info("Middleware reloaded successfully")
	return nil
}

// GetConfig returns the current configuration (thread-safe copy).
func (m *MiddlewareManager) GetConfig() *config.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	cfg := *m.config
	return &cfg
}

// createMiddleware creates a new middleware instance with the given configuration.
// This is the central place where all middleware dependencies are initialized.
func (m *MiddlewareManager) createMiddleware(cfg *config.Config) (*middleware.Middleware, error) {
	translator := i18n.NewTranslator()

	// Create OAuth2 manager
	oauthManager := oauth2.NewManager()
	authPrefix := cfg.Server.GetAuthPathPrefix()

	// Setup OAuth2 providers
	for _, providerCfg := range cfg.OAuth2.Providers {
		if !providerCfg.Enabled {
			continue
		}

		var provider oauth2.Provider

		// Calculate redirect URL
		var redirectURL string
		if cfg.Server.CallbackURL != "" {
			redirectURL = cfg.Server.CallbackURL
		} else {
			redirectPath := joinURLPath(authPrefix, "oauth2/callback")
			redirectURL = fmt.Sprintf("http://%s:%d%s", m.host, m.port, redirectPath)
			if m.host == "0.0.0.0" {
				redirectURL = fmt.Sprintf("http://localhost:%d%s", m.port, redirectPath)
			}
		}

		// Determine provider type
		providerType := providerCfg.Type
		if providerType == "" {
			providerType = providerCfg.Name
		}

		switch providerType {
		case "google":
			provider = oauth2.NewGoogleProvider(
				providerCfg.ClientID,
				providerCfg.ClientSecret,
				redirectURL,
			)
		case "github":
			provider = oauth2.NewGitHubProvider(
				providerCfg.ClientID,
				providerCfg.ClientSecret,
				redirectURL,
			)
		case "microsoft":
			provider = oauth2.NewMicrosoftProvider(
				providerCfg.ClientID,
				providerCfg.ClientSecret,
				redirectURL,
			)
		case "custom":
			if providerCfg.AuthURL == "" || providerCfg.TokenURL == "" || providerCfg.UserInfoURL == "" {
				m.logger.Warn("Custom OAuth2 provider missing required URLs", "provider", providerCfg.Name)
				continue
			}
			provider = oauth2.NewCustomProvider(
				providerCfg.Name,
				providerCfg.ClientID,
				providerCfg.ClientSecret,
				redirectURL,
				providerCfg.AuthURL,
				providerCfg.TokenURL,
				providerCfg.UserInfoURL,
				providerCfg.InsecureSkipVerify,
			)
		default:
			m.logger.Warn("Unknown OAuth2 provider type", "provider", providerCfg.Name, "type", providerType)
			continue
		}

		oauthManager.AddProvider(provider)
		m.logger.Debug("OAuth2 provider registered", "provider", providerCfg.Name)
	}

	// Create authorization checker
	authzChecker := authz.NewEmailChecker(cfg.Authorization)
	m.logger.Debug("Authorization checker initialized", "allowed_entries", len(cfg.Authorization.Allowed))

	// Create email authentication handler if enabled
	var emailHandler *email.Handler
	if cfg.EmailAuth.Enabled {
		var emailBaseURL string
		if cfg.Server.BaseURL != "" {
			emailBaseURL = cfg.Server.BaseURL
		} else {
			emailBaseURL = fmt.Sprintf("http://%s:%d", m.host, m.port)
			if m.host == "0.0.0.0" {
				emailBaseURL = fmt.Sprintf("http://localhost:%d", m.port)
			}
		}

		var err error
		emailHandler, err = email.NewHandler(
			cfg.EmailAuth,
			cfg.Service,
			emailBaseURL,
			authPrefix,
			authzChecker,
			translator,
			cfg.Session.CookieSecret,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create email handler: %w", err)
		}
		m.logger.Debug("Email authentication enabled", "sender", cfg.EmailAuth.SenderType)
	}

	// Create middleware
	mw := middleware.New(
		cfg,
		m.sessionStore,
		oauthManager,
		emailHandler,
		authzChecker,
		translator,
		m.logger.WithModule("middleware"),
	)

	// Wrap with proxy handler if available
	if m.proxyHandler != nil {
		mw = mw.Wrap(m.proxyHandler).(*middleware.Middleware)
	}

	return mw, nil
}

// normalizeAuthPrefix normalizes the authentication path prefix
func normalizeAuthPrefix(prefix string) string {
	if prefix == "" {
		prefix = "/_auth"
	}
	if prefix[0] != '/' {
		prefix = "/" + prefix
	}
	if len(prefix) > 1 {
		prefix = trimRight(prefix, "/")
		if prefix == "" {
			prefix = "/"
		}
	}
	return prefix
}

// trimRight removes trailing occurrences of a character from a string
func trimRight(s, cutset string) string {
	for len(s) > 0 && contains(cutset, s[len(s)-1]) {
		s = s[:len(s)-1]
	}
	return s
}

// contains checks if a string contains a byte
func contains(s string, b byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return true
		}
	}
	return false
}

// joinURLPath joins URL path segments
func joinURLPath(prefix, suffix string) string {
	normalized := normalizeAuthPrefix(prefix)
	suffix = trimLeft(suffix, "/")
	if normalized == "/" {
		return "/" + suffix
	}
	return normalized + "/" + suffix
}

// trimLeft removes leading occurrences of a character from a string
func trimLeft(s, cutset string) string {
	for len(s) > 0 && contains(cutset, s[0]) {
		s = s[1:]
	}
	return s
}
