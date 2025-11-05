package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	host    string
	port    int
	version = "dev" // Set by build
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "chatbotgate",
	Short: "ChatbotGate - Unified authentication reverse proxy",
	Long: `ChatbotGate is a reverse proxy that provides unified authentication
across multiple OAuth2 providers and email-based passwordless authentication.

It can be placed in front of upstream applications to provide integrated
authentication capabilities with host-based multi-tenant routing support.`,
	Version: version,
	// Default to serve command when no subcommand is specified
	RunE: func(cmd *cobra.Command, args []string) error {
		// Execute serve command by default
		return serveCmd.RunE(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Persistent flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "proxyserver.yaml", "Path to configuration file")
	rootCmd.PersistentFlags().StringVar(&host, "host", "0.0.0.0", "Server host address")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 4180, "Server port number")
}
