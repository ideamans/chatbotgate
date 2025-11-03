package factory

import (
	"net/http"

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

// Factory is the interface for creating middleware and its components.
// It serves as a simple DI container, allowing customization of specific components.
type Factory interface {
	// CreateMiddleware is the main factory method that creates a complete Middleware instance.
	// It uses other factory methods internally to build all required components.
	CreateMiddleware(
		cfg *config.Config,
		sessionStore session.Store,
		proxyHandler http.Handler,
		logger logging.Logger,
	) (*middleware.Middleware, error)

	// Component factory methods (used internally by CreateMiddleware)

	// CreateOAuth2Manager creates an OAuth2 manager with configured providers
	CreateOAuth2Manager(cfg *config.Config, host string, port int) *oauth2.Manager

	// CreateEmailHandler creates an email authentication handler if enabled
	CreateEmailHandler(
		cfg *config.Config,
		host string,
		port int,
		authzChecker authz.Checker,
		translator *i18n.Translator,
		tokenKVS kvs.Store,
		rateLimitKVS kvs.Store,
	) (*email.Handler, error)

	// CreateAuthzChecker creates an authorization checker based on config
	CreateAuthzChecker(cfg *config.Config) authz.Checker

	// CreateForwarder creates a forwarder for user info forwarding (may return nil)
	CreateForwarder(cfg *config.Config) forwarding.Forwarder

	// CreatePassthroughMatcher creates a matcher for authentication bypass (may return nil)
	CreatePassthroughMatcher(cfg *config.Config) passthrough.Matcher

	// CreateTranslator creates an i18n translator
	CreateTranslator() *i18n.Translator

	// CreateTokenKVS creates a KVS store for email authentication tokens
	CreateTokenKVS() kvs.Store

	// CreateRateLimitKVS creates a KVS store for rate limiting
	CreateRateLimitKVS() kvs.Store

	// CreateKVSStores creates all required KVS stores from configuration.
	// Returns stores in order: sessionKVS, tokenKVS, rateLimitKVS, error
	CreateKVSStores(cfg *config.Config) (session kvs.Store, token kvs.Store, rateLimit kvs.Store, err error)

	// CreateSessionStore creates a session store using the provided KVS
	CreateSessionStore(kvsStore kvs.Store) session.Store

	// CreateProxyHandler creates a proxy handler from configuration
	CreateProxyHandler(cfg *config.Config) (*proxy.Handler, error)
}
