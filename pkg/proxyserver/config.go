package proxyserver

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/ideamans/chatbotgate/pkg/config"
)

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set namespace defaults
	cfg.KVS.Namespaces.SetDefaults()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}
