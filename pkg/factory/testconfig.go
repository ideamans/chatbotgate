package factory

import (
	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/kvs"
)

// CreateTestConfig creates a minimal valid configuration for testing
func CreateTestConfig() *config.Config {
	return &config.Config{
		Service: config.ServiceConfig{
			Name:        "Test Service",
			Description: "Test Description",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			CookieName:     "_test_session",
			CookieSecret:   "test-secret-key-with-32-characters",
			CookieExpire:   "1h",
			CookieSecure:   false,
			CookieHTTPOnly: true,
			CookieSameSite: "lax",
		},
		Proxy: config.ProxyConfig{
			Upstream: "http://localhost:8080",
		},
		OAuth2: config.OAuth2Config{
			Providers: []config.OAuth2Provider{},
		},
		EmailAuth: config.EmailAuthConfig{
			Enabled:    false,
			SenderType: "smtp",
		},
		Authorization: config.AuthorizationConfig{
			Allowed: []string{"test@example.com"},
		},
		KVS: config.KVSConfig{
			Default: kvs.Config{
				Type: "memory",
			},
			Namespaces: config.NamespaceConfig{
				Session:   "session",
				Token:     "token",
				RateLimit: "ratelimit",
			},
		},
		Logging: config.LoggingConfig{
			Level: "info",
			Color: false,
		},
		Forwarding:  config.ForwardingConfig{},
		Passthrough: config.PassthroughConfig{},
	}
}

// CreateTestConfigWithOAuth2 creates a test config with a Google OAuth2 provider
func CreateTestConfigWithOAuth2() *config.Config {
	cfg := CreateTestConfig()
	cfg.OAuth2.Providers = []config.OAuth2Provider{
		{
			Name:         "google",
			Type:         "google",
			DisplayName:  "Google",
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
	}
	return cfg
}

// CreateTestConfigWithEmail creates a test config with email authentication enabled
func CreateTestConfigWithEmail() *config.Config {
	cfg := CreateTestConfig()
	cfg.EmailAuth = config.EmailAuthConfig{
		Enabled:    true,
		SenderType: "smtp",
		SMTP: config.SMTPConfig{
			Host:     "localhost",
			Port:     1025,
			Username: "test",
			Password: "test",
		},
		Token: config.EmailTokenConfig{
			Expire: "15m",
		},
	}
	return cfg
}
