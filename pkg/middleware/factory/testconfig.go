package factory

import (
	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/middleware/rules"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
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
			Cookie: config.CookieConfig{
				Name:     "_test_session",
				Secret:   "test-secret-key-with-32-characters",
				Expire:   "1h",
				Secure:   false,
				HTTPOnly: true,
				SameSite: "lax",
			},
		},
		OAuth2: config.OAuth2Config{
			Providers: []config.OAuth2Provider{},
		},
		EmailAuth: config.EmailAuthConfig{
			Enabled:    false,
			SenderType: "smtp",
		},
		AccessControl: config.AccessControlConfig{
			Emails: []string{"test@example.com"},
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
		Forwarding: config.ForwardingConfig{},
		Rules:      rules.Config{}, // Empty rules = default behavior (require auth for all)
	}
}

// CreateTestConfigWithOAuth2 creates a test config with a Google OAuth2 provider
func CreateTestConfigWithOAuth2() *config.Config {
	cfg := CreateTestConfig()
	cfg.OAuth2.Providers = []config.OAuth2Provider{
		{
			ID:           "google",
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
