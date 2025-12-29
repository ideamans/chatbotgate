package server

import (
	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
)

// DefaultMiddlewareConfig returns a default middleware configuration
// for running without a config file
func DefaultMiddlewareConfig() *config.Config {
	return &config.Config{
		Service: config.ServiceConfig{
			Name:        "ChatbotGate",
			Description: "Authentication Proxy",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
			Development:    true, // Enable development mode for default config (relaxes CSP)
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:     "_chatbotgate_session",
				Secret:   "default-secret-for-development-use-only-32chars", // Default secret for development
				Expire:   "168h",
				Secure:   false,
				HTTPOnly: true,
				SameSite: "lax",
			},
		},
		PasswordAuth: config.PasswordAuthConfig{
			Enabled:  true,
			Password: "P@ssW0rd", // Default password for development/testing
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
		KVS: config.KVSConfig{
			Default: kvs.Config{
				Type: "memory",
			},
		},
	}
}
