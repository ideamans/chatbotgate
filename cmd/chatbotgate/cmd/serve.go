package cmd

import (
	"context"

	"github.com/ideamans/chatbotgate/cmd/chatbotgate/cmd/server"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the authentication proxy server",
	Long: `Start the chatbotgate server with the specified configuration.

The server will:
- Load the configuration file
- Initialize session storage (memory or Redis)
- Set up OAuth2 and email authentication
- Start the reverse proxy server
- Handle graceful shutdown on SIGTERM/SIGINT`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	// Setup logger
	logger := logging.NewSimpleLogger("main", logging.LevelInfo, true)

	// Create server configuration from command-line flags
	cfg := server.Config{
		ConfigPath: cfgFile,
		Host:       host,
		Port:       port,
		HostSet:    cmd.Flags().Changed("host"),
		PortSet:    cmd.Flags().Changed("port"),
		Logger:     logger,
		Version:    version,
	}

	// Run the server
	return server.Run(context.Background(), cfg)
}
