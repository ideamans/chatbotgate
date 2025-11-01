package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Loader is an interface for loading configuration
type Loader interface {
	Load() (*Config, error)
}

// FileLoader loads configuration from a YAML file
type FileLoader struct {
	path string
}

// NewFileLoader creates a new FileLoader
func NewFileLoader(path string) *FileLoader {
	return &FileLoader{path: path}
}

// Load reads and parses the configuration file
func (l *FileLoader) Load() (*Config, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigFileNotFound, l.path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	applyDefaults(&cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// applyDefaults sets default values for optional fields
func applyDefaults(cfg *Config) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 4180
	}

	if cfg.Session.CookieName == "" {
		cfg.Session.CookieName = "_oauth2_proxy"
	}

	if cfg.Session.CookieExpire == "" {
		cfg.Session.CookieExpire = "168h" // 7 days
	}

	if cfg.Session.CookieSameSite == "" {
		cfg.Session.CookieSameSite = "lax"
	}

	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}

	if cfg.Logging.ModuleLevel == "" {
		cfg.Logging.ModuleLevel = "debug"
	}

	// Set default cookie_httponly to true
	cfg.Session.CookieHTTPOnly = true
}
