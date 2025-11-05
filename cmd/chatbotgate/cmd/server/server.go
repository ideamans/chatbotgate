package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/factory"
	"github.com/ideamans/chatbotgate/pkg/logging"
	"github.com/ideamans/chatbotgate/pkg/proxy"
	"gopkg.in/yaml.v3"
)

// ProxyConfig represents the proxy configuration section in the config file
type ProxyConfig struct {
	Proxy ProxyServerConfig `yaml:"proxy" json:"proxy"`
}

// ProxyServerConfig represents proxy server settings
type ProxyServerConfig struct {
	Upstream proxy.UpstreamConfig `yaml:"upstream" json:"upstream"`
}

// Server represents the HTTP server that integrates proxy and middleware
type Server struct {
	handler http.Handler
	logger  logging.Logger
	host    string
	port    int
	upstream string // for logging
}

// New creates a new Server from configuration file
func New(configPath string, host string, port int, logger logging.Logger) (*Server, error) {
	if logger == nil {
		logger = logging.NewSimpleLogger("server", logging.LevelInfo, true)
	}

	// Load middleware configuration
	middlewareCfg, err := config.NewFileLoader(configPath).Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load middleware config: %w", err)
	}

	// Load proxy configuration
	upstreamCfg, err := loadProxyConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load proxy config: %w", err)
	}

	// Create factory for building middleware components
	f := factory.NewDefaultFactory(host, port, logger)

	// Create KVS stores
	sessionKVS, tokenKVS, rateLimitKVS, err := f.CreateKVSStores(middlewareCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create KVS stores: %w", err)
	}

	// Create session store
	sessionStore := f.CreateSessionStore(sessionKVS)

	// Create proxy handler
	proxyHandler, err := proxy.NewHandlerWithConfig(upstreamCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy handler: %w", err)
	}
	logger.Debug("Proxy handler initialized", "upstream", upstreamCfg.URL)

	// Build middleware using factory
	middleware, err := f.CreateMiddleware(middlewareCfg, sessionStore, proxyHandler, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create middleware: %w", err)
	}

	// Note: tokenKVS and rateLimitKVS are kept alive for the lifetime of the middleware
	// They will be closed when the middleware is garbage collected
	_ = tokenKVS
	_ = rateLimitKVS

	return &Server{
		handler:  middleware,
		logger:   logger,
		host:     host,
		port:     port,
		upstream: upstreamCfg.URL,
	}, nil
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	s.logger.Info("Starting server", "addr", addr, "upstream", s.upstream)

	server := &http.Server{
		Addr:    addr,
		Handler: s.handler,
	}

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down server...")
		if err := server.Shutdown(context.Background()); err != nil {
			s.logger.Error("Server shutdown error", "error", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Handler returns the HTTP handler (useful for testing)
func (s *Server) Handler() http.Handler {
	return s.handler
}

// loadProxyConfig loads proxy configuration from a YAML or JSON file
func loadProxyConfig(path string) (proxy.UpstreamConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return proxy.UpstreamConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg ProxyConfig
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return proxy.UpstreamConfig{}, fmt.Errorf("failed to parse JSON config file: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return proxy.UpstreamConfig{}, fmt.Errorf("failed to parse YAML config file: %w", err)
		}
	default:
		return proxy.UpstreamConfig{}, fmt.Errorf("unsupported config file format: %s (supported: .yaml, .yml, .json)", ext)
	}

	// Validate required fields
	if cfg.Proxy.Upstream.URL == "" {
		return proxy.UpstreamConfig{}, fmt.Errorf("proxy.upstream.url is required")
	}

	return cfg.Proxy.Upstream, nil
}
