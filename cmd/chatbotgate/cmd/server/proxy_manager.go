package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/logging"
	"github.com/ideamans/chatbotgate/pkg/proxy"
	"gopkg.in/yaml.v3"
)

// ProxyManager is an interface for managing proxy lifecycle
type ProxyManager interface {
	// Handler returns the HTTP handler that proxies requests to the upstream
	Handler() http.Handler
}

// ProxyConfig represents the proxy configuration section in the config file
type ProxyConfig struct {
	Proxy ProxyServerConfig `yaml:"proxy" json:"proxy"`
}

// ProxyServerConfig represents proxy server settings
type ProxyServerConfig struct {
	Upstream proxy.UpstreamConfig `yaml:"upstream" json:"upstream"`
}

// SimpleProxyManager is a simple implementation of ProxyManager
type SimpleProxyManager struct {
	handler *proxy.Handler
	logger  logging.Logger
}

// NewProxyManager creates a new SimpleProxyManager from config file
func NewProxyManager(configPath string, logger logging.Logger) (*SimpleProxyManager, error) {
	if logger == nil {
		logger = logging.NewSimpleLogger("proxy-manager", logging.LevelInfo, true)
	}

	// Load proxy configuration from YAML
	upstreamCfg, err := loadProxyConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load proxy config: %w", err)
	}

	logger.Debug("Proxy configuration loaded and validated", "config_path", configPath, "upstream", upstreamCfg.URL)

	// Create proxy handler
	handler, err := proxy.NewHandlerWithConfig(upstreamCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy handler: %w", err)
	}

	logger.Debug("Proxy handler initialized")

	return &SimpleProxyManager{
		handler: handler,
		logger:  logger,
	}, nil
}

// loadProxyConfig loads and validates proxy configuration from a YAML or JSON file
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

	// Validate proxy configuration
	if err := validateProxyConfig(&cfg); err != nil {
		return proxy.UpstreamConfig{}, fmt.Errorf("proxy config validation failed: %w", err)
	}

	return cfg.Proxy.Upstream, nil
}

// validateProxyConfig validates the proxy configuration
// Returns a ValidationError containing all validation errors found
func validateProxyConfig(cfg *ProxyConfig) error {
	verr := config.NewValidationError()

	// Validate required fields
	if cfg.Proxy.Upstream.URL == "" {
		verr.Add(fmt.Errorf("proxy.upstream.url is required"))
	} else {
		// Validate URL format
		if _, err := url.Parse(cfg.Proxy.Upstream.URL); err != nil {
			verr.Add(fmt.Errorf("proxy.upstream.url is not a valid URL: %w", err))
		}
	}

	// Validate secret header configuration (if specified)
	if cfg.Proxy.Upstream.Secret.Header != "" && cfg.Proxy.Upstream.Secret.Value == "" {
		verr.Add(fmt.Errorf("proxy.upstream.secret.value is required when header is specified"))
	}

	return verr.ErrorOrNil()
}

// Handler returns the HTTP handler
func (m *SimpleProxyManager) Handler() http.Handler {
	return m.handler
}
