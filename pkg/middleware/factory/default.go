package factory

import (
	"fmt"
	"net/http"

	"github.com/ideamans/chatbotgate/pkg/middleware/auth/email"
	"github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
	"github.com/ideamans/chatbotgate/pkg/middleware/authz"
	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/middleware/core"
	"github.com/ideamans/chatbotgate/pkg/middleware/forwarding"
	"github.com/ideamans/chatbotgate/pkg/middleware/rules"
	"github.com/ideamans/chatbotgate/pkg/middleware/session"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
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
// tokenKVS and rateLimitKVS should be created via CreateKVSStores() and passed in.
func (f *DefaultFactory) CreateMiddleware(
	cfg *config.Config,
	sessionStore session.Store,
	tokenKVS kvs.Store,
	rateLimitKVS kvs.Store,
	proxyHandler http.Handler,
	logger logging.Logger,
) (*middleware.Middleware, error) {
	// Create all components using factory methods
	translator := f.CreateTranslator()
	authzChecker := f.CreateAuthzChecker(cfg.Authorization)
	forwarder := f.CreateForwarder(cfg.Forwarding, cfg.OAuth2.Providers)
	rulesEvaluator, err := f.CreateRulesEvaluator(&cfg.Rules)
	if err != nil {
		return nil, fmt.Errorf("failed to create rules evaluator: %w", err)
	}

	// Create OAuth2 manager with factory's host/port
	oauthManager := f.CreateOAuth2Manager(cfg.OAuth2, cfg.Server, f.host, f.port)

	// Create email handler if enabled
	var emailHandler *email.Handler
	if cfg.EmailAuth.Enabled {
		var err error
		emailHandler, err = f.CreateEmailHandler(
			cfg.EmailAuth,
			cfg.Service,
			cfg.Server,
			cfg.Session,
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
		rulesEvaluator,
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
func (f *DefaultFactory) CreateOAuth2Manager(oauth2Cfg config.OAuth2Config, serverCfg config.ServerConfig, host string, port int) *oauth2.Manager {
	manager := oauth2.NewManager()

	// Setup OAuth2 providers
	for _, providerCfg := range oauth2Cfg.Providers {
		if providerCfg.Disabled {
			continue
		}

		var provider oauth2.Provider

		// Get callback URL (auto-generated from base_url and auth_path_prefix)
		normalizedHost := host
		if host == "0.0.0.0" {
			normalizedHost = "localhost"
		}
		redirectURL := serverCfg.GetCallbackURL(normalizedHost, port)

		// Use provider type directly
		switch providerCfg.Type {
		case "google":
			provider = oauth2.NewGoogleProvider(
				providerCfg.ClientID,
				providerCfg.ClientSecret,
				redirectURL,
				providerCfg.Scopes,
				providerCfg.ResetScopes,
			)
		case "github":
			provider = oauth2.NewGitHubProvider(
				providerCfg.ClientID,
				providerCfg.ClientSecret,
				redirectURL,
				providerCfg.Scopes,
				providerCfg.ResetScopes,
			)
		case "microsoft":
			provider = oauth2.NewMicrosoftProvider(
				providerCfg.ClientID,
				providerCfg.ClientSecret,
				redirectURL,
				providerCfg.Scopes,
				providerCfg.ResetScopes,
			)
		case "custom":
			if providerCfg.AuthURL == "" || providerCfg.TokenURL == "" || providerCfg.UserInfoURL == "" {
				f.logger.Warn("Skipping custom OAuth2 provider: missing required URLs", "type", providerCfg.Type)
				continue
			}
			provider = oauth2.NewCustomProvider(
				providerCfg.Type,
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
			f.logger.Warn("Skipping OAuth2 provider: unknown provider type", "type", providerCfg.Type)
			continue
		}

		manager.AddProvider(provider)
		f.logger.Debug("OAuth2 provider registered", "type", providerCfg.Type)
	}

	return manager
}

// CreateEmailHandler creates an email authentication handler if enabled
func (f *DefaultFactory) CreateEmailHandler(
	emailAuthCfg config.EmailAuthConfig,
	serviceCfg config.ServiceConfig,
	serverCfg config.ServerConfig,
	sessionCfg config.SessionConfig,
	host string,
	port int,
	authzChecker authz.Checker,
	translator *i18n.Translator,
	tokenKVS kvs.Store,
	rateLimitKVS kvs.Store,
) (*email.Handler, error) {
	authPrefix := serverCfg.GetAuthPathPrefix()

	var emailBaseURL string
	if serverCfg.BaseURL != "" {
		emailBaseURL = serverCfg.BaseURL
	} else {
		// Use HTTPS by default for security, except for localhost/127.0.0.1 (development)
		scheme := "https"
		if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" {
			scheme = "http"
		}
		emailBaseURL = fmt.Sprintf("%s://%s:%d", scheme, host, port)
		if host == "0.0.0.0" {
			emailBaseURL = fmt.Sprintf("%s://localhost:%d", scheme, port)
		}
	}

	handler, err := email.NewHandler(
		emailAuthCfg,
		serviceCfg,
		emailBaseURL,
		authPrefix,
		authzChecker,
		translator,
		sessionCfg.Cookie.Secret,
		tokenKVS,
		rateLimitKVS,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create email handler: %w", err)
	}

	f.logger.Debug("Email authentication handler initialized", "sender", emailAuthCfg.SenderType)
	return handler, nil
}

// CreateAuthzChecker creates an authorization checker based on config
func (f *DefaultFactory) CreateAuthzChecker(authzCfg config.AuthorizationConfig) authz.Checker {
	checker := authz.NewEmailChecker(authzCfg)
	if len(authzCfg.Allowed) > 0 {
		f.logger.Debug("Authorization checker initialized", "allowed_entries", len(authzCfg.Allowed))
	} else {
		f.logger.Debug("Authorization checker initialized with no restrictions")
	}
	return checker
}

// CreateForwarder creates a forwarder for user info forwarding (may return nil)
func (f *DefaultFactory) CreateForwarder(forwardingCfg config.ForwardingConfig, providers []config.OAuth2Provider) forwarding.Forwarder {
	if len(forwardingCfg.Fields) == 0 {
		return nil
	}

	forwarder := forwarding.NewForwarder(&forwardingCfg, providers)
	f.logger.Debug("User info forwarder initialized",
		"fields", len(forwardingCfg.Fields),
		"encryption_enabled", forwardingCfg.Encryption != nil)

	return forwarder
}

// CreateRulesEvaluator creates a rules evaluator from configuration
func (f *DefaultFactory) CreateRulesEvaluator(rulesCfg *rules.Config) (*rules.Evaluator, error) {
	evaluator, err := rules.NewEvaluator(rulesCfg)
	if err != nil {
		return nil, fmt.Errorf("invalid rules configuration: %w", err)
	}
	f.logger.Debug("Rules evaluator initialized", "rule_count", len(rulesCfg.Rules))
	return evaluator, nil
}

// CreateTranslator creates an i18n translator
func (f *DefaultFactory) CreateTranslator() *i18n.Translator {
	return i18n.NewTranslator()
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
			_ = session.Close() // Cleanup
			return nil, nil, nil, fmt.Errorf("failed to create token KVS: %w", err)
		}
		f.logger.Debug("Token KVS initialized (dedicated)", "type", cfg.KVS.Token.Type, "namespace", cfg.KVS.Token.Namespace)
	} else {
		tokenCfg := cfg.KVS.Default
		tokenCfg.Namespace = cfg.KVS.Namespaces.Token
		token, err = kvs.New(tokenCfg)
		if err != nil {
			_ = session.Close() // Cleanup
			return nil, nil, nil, fmt.Errorf("failed to create token KVS: %w", err)
		}
		f.logger.Debug("Token KVS initialized (default)", "type", tokenCfg.Type, "namespace", tokenCfg.Namespace)
	}

	// Initialize rate limit KVS (override or default with namespace)
	if cfg.KVS.RateLimit != nil {
		rateLimit, err = kvs.New(*cfg.KVS.RateLimit)
		if err != nil {
			_ = session.Close() // Cleanup
			_ = token.Close()
			return nil, nil, nil, fmt.Errorf("failed to create rate limit KVS: %w", err)
		}
		f.logger.Debug("Rate limit KVS initialized (dedicated)", "type", cfg.KVS.RateLimit.Type, "namespace", cfg.KVS.RateLimit.Namespace)
	} else {
		rateLimitCfg := cfg.KVS.Default
		rateLimitCfg.Namespace = cfg.KVS.Namespaces.RateLimit
		rateLimit, err = kvs.New(rateLimitCfg)
		if err != nil {
			_ = session.Close() // Cleanup
			_ = token.Close()
			return nil, nil, nil, fmt.Errorf("failed to create rate limit KVS: %w", err)
		}
		f.logger.Debug("Rate limit KVS initialized (default)", "type", rateLimitCfg.Type, "namespace", rateLimitCfg.Namespace)
	}

	return session, token, rateLimit, nil
}

// CreateSessionStore creates a session store using the provided KVS
// Since session.Store is now an alias for kvs.Store, this just returns the input
func (f *DefaultFactory) CreateSessionStore(kvsStore kvs.Store) session.Store {
	return kvsStore
}
