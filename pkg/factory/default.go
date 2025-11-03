package factory

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ideamans/chatbotgate/pkg/auth/email"
	"github.com/ideamans/chatbotgate/pkg/auth/oauth2"
	"github.com/ideamans/chatbotgate/pkg/authz"
	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/forwarding"
	"github.com/ideamans/chatbotgate/pkg/i18n"
	"github.com/ideamans/chatbotgate/pkg/kvs"
	"github.com/ideamans/chatbotgate/pkg/logging"
	"github.com/ideamans/chatbotgate/pkg/middleware"
	"github.com/ideamans/chatbotgate/pkg/passthrough"
	"github.com/ideamans/chatbotgate/pkg/proxy"
	"github.com/ideamans/chatbotgate/pkg/session"
)

// DefaultFactory is the default implementation of Factory.
// It can be embedded in custom factories to override specific methods.
type DefaultFactory struct {
	host   string
	port   int
	logger logging.Logger
}

// NewDefaultFactory creates a new DefaultFactory
func NewDefaultFactory(host string, port int, logger logging.Logger) *DefaultFactory {
	return &DefaultFactory{
		host:   host,
		port:   port,
		logger: logger,
	}
}

// CreateMiddleware creates a complete Middleware instance with all components
func (f *DefaultFactory) CreateMiddleware(
	cfg *config.Config,
	sessionStore session.Store,
	proxyHandler http.Handler,
	logger logging.Logger,
) (*middleware.Middleware, error) {
	// Create all components using factory methods
	translator := f.CreateTranslator()
	authzChecker := f.CreateAuthzChecker(cfg)
	forwarder := f.CreateForwarder(cfg)

	// Create KVS stores for email auth (if needed)
	var tokenKVS, rateLimitKVS kvs.Store
	if cfg.EmailAuth.Enabled {
		tokenKVS = f.CreateTokenKVS()
		rateLimitKVS = f.CreateRateLimitKVS()
	}

	// Create OAuth2 manager with factory's host/port
	oauthManager := f.CreateOAuth2Manager(cfg, f.host, f.port)

	// Create email handler if enabled
	var emailHandler *email.Handler
	if cfg.EmailAuth.Enabled {
		var err error
		emailHandler, err = f.CreateEmailHandler(
			cfg,
			f.host,
			f.port,
			authzChecker,
			translator,
			tokenKVS,
			rateLimitKVS,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create email handler: %w", err)
		}
	}

	// Create middleware
	mw := middleware.New(
		cfg,
		sessionStore,
		oauthManager,
		emailHandler,
		authzChecker,
		forwarder,
		translator,
		logger,
	)

	// Wrap with proxy handler if available
	if proxyHandler != nil {
		mw = mw.Wrap(proxyHandler).(*middleware.Middleware)
	}

	return mw, nil
}

// CreateOAuth2Manager creates an OAuth2 manager with configured providers
func (f *DefaultFactory) CreateOAuth2Manager(cfg *config.Config, host string, port int) *oauth2.Manager {
	manager := oauth2.NewManager()

	// Setup OAuth2 providers
	for _, providerCfg := range cfg.OAuth2.Providers {
		if !providerCfg.Enabled {
			continue
		}

		var provider oauth2.Provider

		// Get callback URL (auto-generated from base_url and auth_path_prefix)
		normalizedHost := host
		if host == "0.0.0.0" {
			normalizedHost = "localhost"
		}
		redirectURL := cfg.Server.GetCallbackURL(normalizedHost, port)

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
				f.logger.Warn("Skipping custom OAuth2 provider: missing required URLs", "provider", providerCfg.Name)
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
				providerCfg.Scopes,
				providerCfg.InsecureSkipVerify,
			)
		default:
			f.logger.Warn("Skipping OAuth2 provider: unknown provider type", "provider", providerCfg.Name, "type", providerType)
			continue
		}

		manager.AddProvider(provider)
		f.logger.Debug("OAuth2 provider registered", "provider", providerCfg.Name, "type", providerType)
	}

	return manager
}

// CreateEmailHandler creates an email authentication handler if enabled
func (f *DefaultFactory) CreateEmailHandler(
	cfg *config.Config,
	host string,
	port int,
	authzChecker authz.Checker,
	translator *i18n.Translator,
	tokenKVS kvs.Store,
	rateLimitKVS kvs.Store,
) (*email.Handler, error) {
	authPrefix := cfg.Server.GetAuthPathPrefix()

	var emailBaseURL string
	if cfg.Server.BaseURL != "" {
		emailBaseURL = cfg.Server.BaseURL
	} else {
		emailBaseURL = fmt.Sprintf("http://%s:%d", host, port)
		if host == "0.0.0.0" {
			emailBaseURL = fmt.Sprintf("http://localhost:%d", port)
		}
	}

	handler, err := email.NewHandler(
		cfg.EmailAuth,
		cfg.Service,
		emailBaseURL,
		authPrefix,
		authzChecker,
		translator,
		cfg.Session.CookieSecret,
		tokenKVS,
		rateLimitKVS,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create email handler: %w", err)
	}

	f.logger.Debug("Email authentication handler initialized", "sender", cfg.EmailAuth.SenderType)
	return handler, nil
}

// CreateAuthzChecker creates an authorization checker based on config
func (f *DefaultFactory) CreateAuthzChecker(cfg *config.Config) authz.Checker {
	checker := authz.NewEmailChecker(cfg.Authorization)
	if len(cfg.Authorization.Allowed) > 0 {
		f.logger.Debug("Authorization checker initialized", "allowed_entries", len(cfg.Authorization.Allowed))
	} else {
		f.logger.Debug("Authorization checker initialized with no restrictions")
	}
	return checker
}

// CreateForwarder creates a forwarder for user info forwarding (may return nil)
func (f *DefaultFactory) CreateForwarder(cfg *config.Config) forwarding.Forwarder {
	if !cfg.Forwarding.QueryString.Enabled && !cfg.Forwarding.Header.Enabled {
		return nil
	}

	forwarder := forwarding.NewForwarder(&cfg.Forwarding, cfg.OAuth2.Providers)
	f.logger.Debug("User info forwarder initialized",
		"querystring_enabled", cfg.Forwarding.QueryString.Enabled,
		"header_enabled", cfg.Forwarding.Header.Enabled,
		"fields", len(cfg.Forwarding.Fields))

	return forwarder
}

// CreatePassthroughMatcher creates a matcher for authentication bypass (may return nil)
func (f *DefaultFactory) CreatePassthroughMatcher(cfg *config.Config) passthrough.Matcher {
	matcher := passthrough.NewMatcher(&cfg.Passthrough)
	if matcher.HasErrors() {
		f.logger.Warn("Passthrough configuration has errors", "errors", matcher.Errors())
	}
	return matcher
}

// CreateTranslator creates an i18n translator
func (f *DefaultFactory) CreateTranslator() *i18n.Translator {
	return i18n.NewTranslator()
}

// CreateTokenKVS creates a KVS store for email authentication tokens
func (f *DefaultFactory) CreateTokenKVS() kvs.Store {
	// Tokens expire after configured duration (default 15 minutes)
	// Cleanup every 5 minutes
	store, _ := kvs.NewMemoryStore("tokens", kvs.MemoryConfig{
		CleanupInterval: 5 * time.Minute,
	})
	return store
}

// CreateRateLimitKVS creates a KVS store for rate limiting
func (f *DefaultFactory) CreateRateLimitKVS() kvs.Store {
	// Rate limit entries expire after 1 hour
	// Cleanup every 15 minutes
	store, _ := kvs.NewMemoryStore("ratelimit", kvs.MemoryConfig{
		CleanupInterval: 15 * time.Minute,
	})
	return store
}

// CreateKVSStores creates all required KVS stores from configuration.
// This consolidates the KVS creation logic that was previously in serve.go.
func (f *DefaultFactory) CreateKVSStores(cfg *config.Config) (session kvs.Store, token kvs.Store, rateLimit kvs.Store, err error) {
	// Set default namespace names
	cfg.KVS.Namespaces.SetDefaults()

	// Default KVS type fallback
	if cfg.KVS.Default.Type == "" {
		cfg.KVS.Default.Type = "memory"
	}

	// Initialize session KVS (override or default with namespace)
	if cfg.KVS.Session != nil {
		session, err = kvs.New(*cfg.KVS.Session)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create session KVS: %w", err)
		}
		f.logger.Debug("Session KVS initialized (dedicated)", "type", cfg.KVS.Session.Type, "namespace", cfg.KVS.Session.Namespace)
	} else {
		sessionCfg := cfg.KVS.Default
		sessionCfg.Namespace = cfg.KVS.Namespaces.Session
		session, err = kvs.New(sessionCfg)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create session KVS: %w", err)
		}
		f.logger.Debug("Session KVS initialized (default)", "type", sessionCfg.Type, "namespace", sessionCfg.Namespace)
	}

	// Initialize token KVS (override or default with namespace)
	if cfg.KVS.Token != nil {
		token, err = kvs.New(*cfg.KVS.Token)
		if err != nil {
			session.Close() // Cleanup
			return nil, nil, nil, fmt.Errorf("failed to create token KVS: %w", err)
		}
		f.logger.Debug("Token KVS initialized (dedicated)", "type", cfg.KVS.Token.Type, "namespace", cfg.KVS.Token.Namespace)
	} else {
		tokenCfg := cfg.KVS.Default
		tokenCfg.Namespace = cfg.KVS.Namespaces.Token
		token, err = kvs.New(tokenCfg)
		if err != nil {
			session.Close() // Cleanup
			return nil, nil, nil, fmt.Errorf("failed to create token KVS: %w", err)
		}
		f.logger.Debug("Token KVS initialized (default)", "type", tokenCfg.Type, "namespace", tokenCfg.Namespace)
	}

	// Initialize rate limit KVS (override or default with namespace)
	if cfg.KVS.RateLimit != nil {
		rateLimit, err = kvs.New(*cfg.KVS.RateLimit)
		if err != nil {
			session.Close() // Cleanup
			token.Close()
			return nil, nil, nil, fmt.Errorf("failed to create rate limit KVS: %w", err)
		}
		f.logger.Debug("Rate limit KVS initialized (dedicated)", "type", cfg.KVS.RateLimit.Type, "namespace", cfg.KVS.RateLimit.Namespace)
	} else {
		rateLimitCfg := cfg.KVS.Default
		rateLimitCfg.Namespace = cfg.KVS.Namespaces.RateLimit
		rateLimit, err = kvs.New(rateLimitCfg)
		if err != nil {
			session.Close() // Cleanup
			token.Close()
			return nil, nil, nil, fmt.Errorf("failed to create rate limit KVS: %w", err)
		}
		f.logger.Debug("Rate limit KVS initialized (default)", "type", rateLimitCfg.Type, "namespace", rateLimitCfg.Namespace)
	}

	return session, token, rateLimit, nil
}

// CreateSessionStore creates a session store using the provided KVS
func (f *DefaultFactory) CreateSessionStore(kvsStore kvs.Store) session.Store {
	return session.NewKVSStore(kvsStore)
}

// CreateProxyHandler creates a proxy handler from configuration
func (f *DefaultFactory) CreateProxyHandler(cfg *config.Config) (*proxy.Handler, error) {
	if len(cfg.Proxy.Hosts) > 0 {
		handler, err := proxy.NewHandlerWithHosts(cfg.Proxy.Upstream, cfg.Proxy.Hosts)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy handler with hosts: %w", err)
		}
		f.logger.Debug("Proxy handler initialized with host routing",
			"default_upstream", cfg.Proxy.Upstream,
			"hosts", len(cfg.Proxy.Hosts))
		return handler, nil
	}

	handler, err := proxy.NewHandler(cfg.Proxy.Upstream)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy handler: %w", err)
	}
	f.logger.Debug("Proxy handler initialized", "upstream", cfg.Proxy.Upstream)
	return handler, nil
}
