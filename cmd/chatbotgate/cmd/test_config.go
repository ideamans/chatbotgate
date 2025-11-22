package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/proxy/core"
	sharedconfig "github.com/ideamans/chatbotgate/pkg/shared/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// testConfigCmd represents the test-config command
var testConfigCmd = &cobra.Command{
	Use:   "test-config",
	Short: "Validate the configuration file",
	Long: `Test and validate the configuration file without starting the server.

This command will:
- Load the configuration file from the specified path
- Parse the YAML/JSON content
- Validate all required fields
- Check for common configuration errors
- Report any issues found

If the configuration is valid, the command exits with status 0.
If there are validation errors, the command exits with status 1.`,
	RunE: runTestConfig,
}

func init() {
	rootCmd.AddCommand(testConfigCmd)
}

// ProxyConfig represents the proxy configuration section in the config file
type ProxyConfig struct {
	Proxy ProxyServerConfig `yaml:"proxy" json:"proxy"`
}

// ProxyServerConfig represents proxy server settings
type ProxyServerConfig struct {
	Upstream proxy.UpstreamConfig `yaml:"upstream" json:"upstream"`
}

func runTestConfig(cmd *cobra.Command, args []string) error {
	fmt.Printf("Testing configuration file: %s\n", cfgFile)

	// Load middleware configuration
	middlewareCfg, err := config.NewFileLoader(cfgFile).Load()
	if err != nil {
		return fmt.Errorf("failed to load middleware configuration: %w", err)
	}

	// Load proxy configuration
	upstreamCfg, err := loadProxyConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load proxy configuration: %w", err)
	}

	fmt.Println("✓ Configuration file loaded successfully")
	fmt.Println("✓ Configuration validation passed")

	// Print summary
	fmt.Println("\nConfiguration Summary:")
	fmt.Printf("  Service Name: %s\n", middlewareCfg.Service.Name)
	fmt.Printf("  Upstream: %s\n", upstreamCfg.URL)

	// Count available OAuth2 providers (not disabled)
	availableProviders := 0
	for _, p := range middlewareCfg.OAuth2.Providers {
		if !p.Disabled {
			availableProviders++
		}
	}
	fmt.Printf("  OAuth2 Providers: %d available\n", availableProviders)

	if middlewareCfg.EmailAuth.Enabled {
		fmt.Printf("  Email Auth: enabled (%s)\n", middlewareCfg.EmailAuth.SenderType)
	} else {
		fmt.Println("  Email Auth: disabled")
	}

	// KVS configuration
	fmt.Printf("  Default KVS: %s\n", middlewareCfg.KVS.Default.Type)
	if middlewareCfg.KVS.Session != nil {
		fmt.Printf("  Session KVS: %s (dedicated)\n", middlewareCfg.KVS.Session.Type)
	} else {
		fmt.Printf("  Session KVS: %s (shared with namespace: %s)\n", middlewareCfg.KVS.Default.Type, middlewareCfg.KVS.Namespaces.Session)
	}

	// Authorization
	allowedCount := len(middlewareCfg.Authorization.Allowed)
	if allowedCount > 0 {
		fmt.Printf("  Allowed Users/Domains: %d entries\n", allowedCount)
	} else {
		fmt.Println("  Allowed Users/Domains: none (all authenticated users allowed)")
	}

	fmt.Println("\n✓ Configuration is valid and ready to use")
	return nil
}

// loadProxyConfig loads proxy configuration from a YAML or JSON file
func loadProxyConfig(path string) (proxy.UpstreamConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return proxy.UpstreamConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in config file
	data = sharedconfig.ExpandEnvBytes(data)

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
