package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/proxy/core"
	"github.com/ideamans/chatbotgate/pkg/shared/filewatcher"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
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

// SimpleProxyManager is a simple implementation of ProxyManager with hot reload support
type SimpleProxyManager struct {
	handler       atomic.Value // Stores *proxy.Handler
	configPath    string
	defaultConfig *proxy.UpstreamConfig // Default config to use when file not found
	logger        logging.Logger
}

// NewProxyManager creates a new SimpleProxyManager from config file
func NewProxyManager(configPath string, logger logging.Logger) (*SimpleProxyManager, error) {
	return NewProxyManagerWithDefault(configPath, nil, logger)
}

// NewProxyManagerWithDefault creates a new SimpleProxyManager from config file
// with fallback to default config when the file is not found
func NewProxyManagerWithDefault(configPath string, defaultConfig *proxy.UpstreamConfig, logger logging.Logger) (*SimpleProxyManager, error) {
	if logger == nil {
		logger = logging.NewSimpleLogger("proxy-manager", logging.LevelInfo, true)
	}

	m := &SimpleProxyManager{
		configPath:    configPath,
		defaultConfig: defaultConfig,
		logger:        logger,
	}

	// Build initial proxy handler
	handler, err := m.buildProxyHandler(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build initial proxy handler: %w", err)
	}

	// Store initial handler atomically
	m.handler.Store(handler)

	if defaultConfig != nil && configPath == "" {
		logger.Info("Proxy manager initialized with default config")
	} else {
		logger.Info("Proxy manager initialized", "config_path", configPath)
	}

	return m, nil
}

// buildProxyHandler builds proxy handler from configuration file
func (m *SimpleProxyManager) buildProxyHandler(configPath string) (*proxy.Handler, error) {
	// Load proxy configuration from YAML
	upstreamCfg, err := loadProxyConfig(configPath)
	if err != nil {
		// If config file not found and we have default config, use it
		if errors.Is(err, os.ErrNotExist) && m.defaultConfig != nil {
			upstreamCfg = *m.defaultConfig
			m.logger.Debug("Using default proxy configuration", "upstream", upstreamCfg.URL)
		} else {
			return nil, fmt.Errorf("failed to load proxy config: %w", err)
		}
	} else {
		m.logger.Debug("Proxy configuration loaded and validated", "config_path", configPath, "upstream", upstreamCfg.URL)
	}

	// Create proxy handler
	handler, err := proxy.NewHandlerWithConfig(upstreamCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy handler: %w", err)
	}

	m.logger.Debug("Proxy handler initialized")

	return handler, nil
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

// OnFileChange implements filewatcher.ChangeListener interface
// This method is called when the configuration file changes
func (m *SimpleProxyManager) OnFileChange(event filewatcher.ChangeEvent) {
	if event.Error != nil {
		m.logger.Error("File change event error", "error", event.Error)
		return
	}

	m.logger.Info("Config content change detected, starting reload", "path", event.Path, "component", "proxy")
	m.reload(event.Path)
}

// reload reloads the proxy configuration and replaces the current handler atomically
func (m *SimpleProxyManager) reload(configPath string) {
	// Build new proxy handler
	newHandler, err := m.buildProxyHandler(configPath)
	if err != nil {
		m.logger.Error("Failed to reload proxy", "error", err, "path", configPath)
		m.logger.Error("Keeping current proxy configuration")
		return
	}

	// Atomically replace the handler
	m.handler.Store(newHandler)
	m.logger.Info("Configuration reloaded successfully", "component", "proxy")
}

// Handler returns the HTTP handler
// The handler always uses the latest proxy handler stored atomically
func (m *SimpleProxyManager) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Load the current handler atomically
		h := m.handler.Load().(*proxy.Handler)
		h.ServeHTTP(w, r)
	})
}
