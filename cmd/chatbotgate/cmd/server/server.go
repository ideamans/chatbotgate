package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	proxy "github.com/ideamans/chatbotgate/pkg/proxy/core"
	"github.com/ideamans/chatbotgate/pkg/shared/filewatcher"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
	"gopkg.in/yaml.v3"
)

// Config represents the configuration for running the server
type Config struct {
	ConfigPath string
	Host       string // From command-line flag
	Port       int    // From command-line flag
	HostSet    bool   // Whether host was explicitly set via flag
	PortSet    bool   // Whether port was explicitly set via flag
	Logger     logging.Logger
	Version    string
}

// ServerConfigWrapper represents the server configuration section in the config file
type ServerConfigWrapper struct {
	Server ServerConfig `yaml:"server" json:"server"`
}

// ServerConfig represents server settings from config file
type ServerConfig struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

// ResolvedConfig represents the final resolved configuration
type ResolvedConfig struct {
	Host string
	Port int
}

// Run starts the server with the given configuration
// This is the main entry point for starting the server
func Run(ctx context.Context, cfg Config) error {
	logger := cfg.Logger
	if logger == nil {
		logger = logging.NewSimpleLogger("main", logging.LevelInfo, true)
	}

	logger.Info("Starting chatbotgate", "version", cfg.Version)

	// Check if config file exists and determine if we should use defaults
	useDefaultConfig := false
	configPath := cfg.ConfigPath
	if configPath != "" {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			logger.Warn("Config file not found, using default configuration", "path", configPath)
			logDefaultConfigWarnings(logger)
			useDefaultConfig = true
			configPath = "" // Clear config path to signal managers to use defaults
		}
	} else {
		logger.Warn("No config file specified, using default configuration")
		logDefaultConfigWarnings(logger)
		useDefaultConfig = true
	}

	// Resolve final host and port
	resolved, err := resolveServerConfig(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to resolve server config: %w", err)
	}

	// Get default configs if needed
	var defaultMiddlewareConfig *config.Config
	var defaultProxyConfig *proxy.UpstreamConfig
	var dummyUpstream *DummyUpstream
	if useDefaultConfig {
		defaultMiddlewareConfig = DefaultMiddlewareConfig()

		// Start dummy upstream server when no config is provided
		dummyUpstream = NewDummyUpstream(logger)
		if dummyUpstream != nil {
			// Use dummy upstream URL
			defaultProxyConfig = &proxy.UpstreamConfig{
				URL: dummyUpstream.URL(),
			}
			logger.Warn("Using DUMMY upstream server (for development only)", "url", dummyUpstream.URL())
		} else {
			return fmt.Errorf("failed to start dummy upstream server")
		}
	}

	// Create proxy manager from config file (with default config fallback)
	proxyManager, err := NewProxyManagerWithDefault(configPath, defaultProxyConfig, logger)
	if err != nil {
		return formatConfigError("proxy", err)
	}

	logger.Info("Proxy manager initialized successfully")

	// Create middleware manager from config file (with proxy as next handler and default config fallback)
	middlewareManager, err := NewMiddlewareManagerWithDefault(configPath, defaultMiddlewareConfig, resolved.Host, resolved.Port, proxyManager.Handler(), logger)
	if err != nil {
		return formatConfigError("middleware", err)
	}

	logger.Info("Middleware manager initialized successfully")

	// Create file watcher for hot reload (100ms debounce) only if config file exists
	var watcher *filewatcher.Watcher
	if !useDefaultConfig && cfg.ConfigPath != "" {
		watcher, err = filewatcher.NewWatcher(cfg.ConfigPath, 100*time.Millisecond)
		if err != nil {
			logger.Error("Failed to create file watcher", "error", err)
			return fmt.Errorf("failed to create file watcher: %w", err)
		}
		defer func() { _ = watcher.Close() }()

		// Register managers as listeners for config file changes
		watcher.AddListener(middlewareManager)
		watcher.AddListener(proxyManager)

		logger.Info("File watcher initialized for hot reload", "config_file", cfg.ConfigPath)
	}

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", resolved.Host, resolved.Port)
	logger.Info("Server initialized successfully")

	// Setup signal handling
	sigCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start file watcher in background (only if watcher was created)
	if watcher != nil {
		go func() {
			if err := watcher.Start(sigCtx); err != nil && err != context.Canceled {
				logger.Error("File watcher error", "error", err)
			}
		}()
	}

	// Create and start HTTP server
	server := &http.Server{
		Addr:    addr,
		Handler: middlewareManager.Handler(),
	}

	logger.Info("Starting server", "addr", addr)

	// Run server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-stop:
		logger.Info("Shutdown signal received, stopping server...")
		cancel()

		// Mark middleware as draining (will return 503 for health checks)
		middlewareManager.SetDraining()

		// Graceful shutdown
		if err := server.Shutdown(context.Background()); err != nil {
			logger.Error("Server shutdown error", "error", err)
		}
		// Wait for server to finish
		if err := <-errChan; err != nil {
			logger.Error("Server stopped with error", "error", err)
			// Stop dummy upstream before returning
			if dummyUpstream != nil {
				dummyUpstream.Stop()
			}
			return err
		}
	case err := <-errChan:
		if err != nil {
			logger.Error("Server stopped with error", "error", err)
			// Stop dummy upstream before returning
			if dummyUpstream != nil {
				dummyUpstream.Stop()
			}
			return err
		}
	}

	// Stop dummy upstream server if it was started
	if dummyUpstream != nil {
		dummyUpstream.Stop()
	}

	logger.Info("Server stopped successfully")
	return nil
}

// resolveServerConfig resolves the final host and port configuration
// Priority: Command-line flags > Config file > Default values
func resolveServerConfig(cfg Config, logger logging.Logger) (ResolvedConfig, error) {
	// Load server configuration from config file
	serverCfg, err := loadServerConfig(cfg.ConfigPath)
	if err != nil {
		logger.Warn("Failed to load server config from file, using defaults", "error", err)
	}

	resolved := ResolvedConfig{
		Host: cfg.Host,
		Port: cfg.Port,
	}

	// If host flag was not explicitly set, try config file value
	if !cfg.HostSet && serverCfg.Host != "" {
		resolved.Host = serverCfg.Host
		logger.Info("Using host from config file", "host", resolved.Host)
	} else if cfg.HostSet {
		logger.Info("Using host from command-line flag", "host", resolved.Host)
	}

	// If port flag was not explicitly set, try config file value
	if !cfg.PortSet && serverCfg.Port != 0 {
		resolved.Port = serverCfg.Port
		logger.Info("Using port from config file", "port", resolved.Port)
	} else if cfg.PortSet {
		logger.Info("Using port from command-line flag", "port", resolved.Port)
	}

	return resolved, nil
}

// loadServerConfig loads server configuration from a YAML or JSON file
// Returns default values (empty string for host, 0 for port) if not found in config
func loadServerConfig(path string) (ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// If config file doesn't exist, return empty config (will use defaults)
		if os.IsNotExist(err) {
			return ServerConfig{}, nil
		}
		return ServerConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg ServerConfigWrapper
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return ServerConfig{}, fmt.Errorf("failed to parse JSON config file: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return ServerConfig{}, fmt.Errorf("failed to parse YAML config file: %w", err)
		}
	default:
		return ServerConfig{}, fmt.Errorf("unsupported config file format: %s (supported: .yaml, .yml, .json)", ext)
	}

	return cfg.Server, nil
}

// formatConfigError formats configuration errors with helpful messages
func formatConfigError(component string, err error) error {
	// Check if it's a ValidationError (multiple errors)
	var validationErr *config.ValidationError
	if errors.As(err, &validationErr) {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Configuration validation failed for %s with %d error(s):\n\n", component, len(validationErr.Errors)))
		for i, e := range validationErr.Errors {
			sb.WriteString(fmt.Sprintf("  %d. %v\n", i+1, e))
		}
		sb.WriteString("\nPlease fix the errors above in your configuration file.")
		return errors.New(sb.String())
	}

	// Check if it's a validation error (wrapped)
	if errors.Is(err, config.ErrServiceNameRequired) ||
		errors.Is(err, config.ErrCookieSecretRequired) ||
		errors.Is(err, config.ErrCookieSecretTooShort) ||
		errors.Is(err, config.ErrNoEnabledProviders) ||
		errors.Is(err, config.ErrEncryptionKeyRequired) ||
		errors.Is(err, config.ErrEncryptionKeyTooShort) ||
		errors.Is(err, config.ErrEncryptionConfigRequired) {
		return fmt.Errorf("configuration validation error in %s: %v - please check your configuration file and fix the issue above", component, err)
	}

	// Check if it's a config file not found error
	if errors.Is(err, config.ErrConfigFileNotFound) {
		return fmt.Errorf("configuration file not found: %v - please create a configuration file or specify the correct path with --config flag", err)
	}

	// Check if it contains "validation failed" in the error message
	if strings.Contains(err.Error(), "validation failed") {
		return fmt.Errorf("configuration validation error in %s: %v - please check your configuration file and fix the validation errors above", component, err)
	}

	// Generic error
	return fmt.Errorf("failed to initialize %s: %v", component, err)
}

// logDefaultConfigWarnings logs warnings about default configuration values
// These warnings are important to remind users that default values are for development only
func logDefaultConfigWarnings(logger logging.Logger) {
	logger.Warn("========================================")
	logger.Warn("WARNING: Using default configuration")
	logger.Warn("DO NOT USE IN PRODUCTION")
	logger.Warn("========================================")
	logger.Warn("Default values in use:")
	logger.Warn("  - Cookie Secret: fixed dummy value")
	logger.Warn("  - Password Auth: enabled with password 'P@ssW0rd'")
	logger.Warn("  - Upstream: dummy server (auto-started)")
	logger.Warn("========================================")
}
