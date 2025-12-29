package server

import (
	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	proxy "github.com/ideamans/chatbotgate/pkg/proxy/core"
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
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:     "_chatbotgate_session",
				Secret:   "", // Will be generated if empty
				Expire:   "168h",
				Secure:   false,
				HTTPOnly: true,
				SameSite: "lax",
			},
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

// DefaultProxyConfig returns a default proxy configuration
// for running without a config file
func DefaultProxyConfig() proxy.UpstreamConfig {
	return proxy.UpstreamConfig{
		URL: "http://localhost:8080",
	}
}
