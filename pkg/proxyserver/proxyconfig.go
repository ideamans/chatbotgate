package proxyserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProxyConfig represents the complete proxy server configuration
type ProxyConfig struct {
	Proxy ProxyServerConfig `yaml:"proxy" json:"proxy"` // Proxy configuration
}

// ProxyServerConfig represents proxy server settings
type ProxyServerConfig struct {
	Upstream UpstreamConfig `yaml:"upstream" json:"upstream"` // Upstream configuration (required)
}

// UpstreamConfig represents upstream server configuration with optional secret header
type UpstreamConfig struct {
	URL    string       `yaml:"url" json:"url"`       // Upstream URL (required)
	Secret SecretConfig `yaml:"secret" json:"secret"` // Secret header configuration (optional)
}

// SecretConfig represents secret header configuration for upstream authentication
type SecretConfig struct {
	Header string `yaml:"header" json:"header"` // HTTP header name (e.g., "X-Chatbotgate-Secret")
	Value  string `yaml:"value" json:"value"`   // Secret value to send
}

// LoadProxyConfig loads proxy configuration from a YAML or JSON file
func LoadProxyConfig(path string) (*ProxyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg ProxyConfig
	ext := strings.ToLower(filepath.Ext(path))

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

	// Validate required fields
	if cfg.Proxy.Upstream.URL == "" {
		return nil, fmt.Errorf("proxy.upstream.url is required")
	}

	return &cfg, nil
}
