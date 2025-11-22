package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sharedconfig "github.com/ideamans/chatbotgate/pkg/shared/config"
	"gopkg.in/yaml.v3"
)

// Loader is an interface for loading configuration
type Loader interface {
	Load() (*Config, error)
}

// FileLoader loads configuration from a YAML or JSON file
type FileLoader struct {
	path string
}

// NewFileLoader creates a new FileLoader
func NewFileLoader(path string) *FileLoader {
	return &FileLoader{path: path}
}

// Load reads and parses the configuration file
// Supports both YAML (.yaml, .yml) and JSON (.json) formats
// Format is automatically detected from file extension
// Environment variables in the format ${VAR} or ${VAR:-default} are expanded
func (l *FileLoader) Load() (*Config, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigFileNotFound, l.path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in config file
	data = sharedconfig.ExpandEnvBytes(data)

	var cfg Config
	ext := strings.ToLower(filepath.Ext(l.path))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config file: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config file: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s (supported: .yaml, .yml, .json)", ext)
	}

	// Apply defaults
	applyDefaults(&cfg)

	// Note: Detailed validation is performed by the middleware manager
	// This allows for better error reporting with multiple validation errors

	return &cfg, nil
}

// applyDefaults sets default values for optional fields
func applyDefaults(cfg *Config) {
	if cfg.Session.Cookie.Name == "" {
		cfg.Session.Cookie.Name = "_oauth2_proxy"
	}

	if cfg.Session.Cookie.Expire == "" {
		cfg.Session.Cookie.Expire = "168h" // 7 days
	}

	if cfg.Session.Cookie.SameSite == "" {
		cfg.Session.Cookie.SameSite = "lax"
	}

	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}

	// Set default cookie_httponly to true
	cfg.Session.Cookie.HTTPOnly = true
}
