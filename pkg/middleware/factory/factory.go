package factory

import (
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

// Factory is the interface for creating middleware and its components.
// It serves as a simple DI container, allowing customization of specific components.
type Factory interface {
	// CreateMiddleware is the main factory method that creates a complete Middleware instance.
	// It uses other factory methods internally to build all required components.
	// tokenKVS and emailQuotaKVS should be created via CreateKVSStores().
	CreateMiddleware(
		cfg *config.Config,
		sessionStore session.Store,
		tokenKVS kvs.Store,
		emailQuotaKVS kvs.Store,
		proxyHandler http.Handler,
		logger logging.Logger,
	) (*middleware.Middleware, error)

	// Component factory methods (used internally by CreateMiddleware)

	// CreateOAuth2Manager creates an OAuth2 manager with configured providers
	CreateOAuth2Manager(oauth2Cfg config.OAuth2Config, serverCfg config.ServerConfig, host string, port int) *oauth2.Manager

	// CreateEmailHandler creates an email authentication handler if enabled
	CreateEmailHandler(
		emailAuthCfg config.EmailAuthConfig,
		serviceCfg config.ServiceConfig,
		serverCfg config.ServerConfig,
		sessionCfg config.SessionConfig,
		host string,
		port int,
		authzChecker authz.Checker,
		translator *i18n.Translator,
		tokenKVS kvs.Store,
		emailQuotaKVS kvs.Store,
	) (*email.Handler, error)

	// CreateAuthzChecker creates an authorization checker based on config
	CreateAuthzChecker(accessControlCfg config.AccessControlConfig) authz.Checker

	// CreateForwarder creates a forwarder for user info forwarding (may return nil)
	CreateForwarder(forwardingCfg config.ForwardingConfig, providers []config.OAuth2Provider) forwarding.Forwarder

	// CreateRulesEvaluator creates a rules evaluator from configuration
	CreateRulesEvaluator(rulesCfg *rules.Config) (*rules.Evaluator, error)

	// CreateTranslator creates an i18n translator
	CreateTranslator() *i18n.Translator

	// CreateKVSStores creates all required KVS stores from configuration.
	// This is the primary method for creating KVS stores. It should be called once
	// at startup, and the returned stores should be passed to CreateMiddleware().
	// Returns stores in order: sessionKVS, tokenKVS, emailQuotaKVS, error
	CreateKVSStores(cfg *config.Config) (session kvs.Store, token kvs.Store, emailQuota kvs.Store, err error)

	// CreateSessionStore creates a session store using the provided KVS
	CreateSessionStore(kvsStore kvs.Store) session.Store
}
