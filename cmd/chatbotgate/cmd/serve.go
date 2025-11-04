package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ideamans/chatbotgate/pkg/logging"
	"github.com/ideamans/chatbotgate/pkg/proxyserver"
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
	// Setup logger for initialization
	logger := logging.NewSimpleLogger("main", logging.LevelInfo, true)

	logger.Info("Starting chatbotgate", "version", version)

	// Create server from config file
	server, err := proxyserver.New(cfgFile, host, port, logger)
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		return fmt.Errorf("failed to create server: %w", err)
	}

	logger.Info("Server initialized successfully")

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start(ctx)
	}()

	// Wait for shutdown signal or error
	select {
	case <-stop:
		logger.Info("Shutdown signal received, stopping server...")
		cancel()
		// Wait for server to finish
		if err := <-errChan; err != nil {
			logger.Error("Server stopped with error", "error", err)
			return err
		}
	case err := <-errChan:
		if err != nil {
			logger.Error("Server stopped with error", "error", err)
			return err
		}
	}

	logger.Info("Server stopped successfully")
	return nil
}
