package cmd

import (
	"fmt"

	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/proxyserver"
	"github.com/spf13/cobra"
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

func runTestConfig(cmd *cobra.Command, args []string) error {
	fmt.Printf("Testing configuration file: %s\n", cfgFile)

	// Load middleware configuration
	middlewareCfg, err := config.NewFileLoader(cfgFile).Load()
	if err != nil {
		return fmt.Errorf("failed to load middleware configuration: %w", err)
	}

	// Load proxy configuration
	proxyCfg, err := proxyserver.LoadProxyConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load proxy configuration: %w", err)
	}

	fmt.Println("✓ Configuration file loaded successfully")
	fmt.Println("✓ Configuration validation passed")

	// Print summary
	fmt.Println("\nConfiguration Summary:")
	fmt.Printf("  Service Name: %s\n", middlewareCfg.Service.Name)
	fmt.Printf("  Upstream: %s\n", proxyCfg.Proxy.Upstream.URL)

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
