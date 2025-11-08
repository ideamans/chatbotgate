package cmd

import (
	"context"
	"fmt"

	"github.com/ideamans/chatbotgate/cmd/chatbotgate/cmd/server"
	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
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
	// Load config file to get logging settings
	var appConfig config.Config
	if cfgFile != "" {
		data, err := os.ReadFile(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		if err := yaml.Unmarshal(data, &appConfig); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Setup logger with file output if configured
	level := logging.ParseLevel(appConfig.Logging.Level)
	var fileRotationConfig *logging.FileRotationConfig
	if appConfig.Logging.File != nil && appConfig.Logging.File.Path != "" {
		fileRotationConfig = &logging.FileRotationConfig{
			Path:       appConfig.Logging.File.Path,
			MaxSizeMB:  appConfig.Logging.File.MaxSizeMB,
			MaxBackups: appConfig.Logging.File.MaxBackups,
			MaxAge:     appConfig.Logging.File.MaxAge,
			Compress:   appConfig.Logging.File.Compress,
		}
	}

	logger, err := logging.NewLoggerWithFile("main", level, appConfig.Logging.Color, fileRotationConfig)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

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
