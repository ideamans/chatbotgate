package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ideamans/chatbotgate/cmd/chatbotgate/cmd/server"
	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/logging"
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

	// Create proxy manager from config file
	proxyManager, err := server.NewProxyManager(cfgFile, logger)
	if err != nil {
		return formatConfigError("proxy", err)
	}

	logger.Info("Proxy manager initialized successfully")

	// Create middleware manager from config file (with proxy as next handler)
	middlewareManager, err := server.NewMiddlewareManager(cfgFile, host, port, proxyManager.Handler(), logger)
	if err != nil {
		return formatConfigError("middleware", err)
	}

	logger.Info("Middleware manager initialized successfully")

	// Create server with middleware manager
	srv, err := server.New(middlewareManager, host, port, logger)
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
		errChan <- srv.Start(ctx)
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

// formatConfigError formats configuration errors with helpful messages
func formatConfigError(component string, err error) error {
	// Check if it's a ValidationError (multiple errors)
	var validationErr *config.ValidationError
	if errors.As(err, &validationErr) {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Configuration validation failed for %s with %d error(s):\n\n", component, len(validationErr.Errors)))
		for i, e := range validationErr.Errors {
			sb.WriteString(fmt.Sprintf("  %d. %v\n", i+1, e))
		}
		sb.WriteString("\nPlease fix the errors above in your configuration file.")
		return errors.New(sb.String())
	}

	// Check if it's a validation error (wrapped)
	if errors.Is(err, config.ErrServiceNameRequired) ||
		errors.Is(err, config.ErrCookieSecretRequired) ||
		errors.Is(err, config.ErrCookieSecretTooShort) ||
		errors.Is(err, config.ErrNoEnabledProviders) ||
		errors.Is(err, config.ErrEncryptionKeyRequired) ||
		errors.Is(err, config.ErrEncryptionKeyTooShort) ||
		errors.Is(err, config.ErrForwardingFieldsRequired) ||
		errors.Is(err, config.ErrInvalidForwardingField) {
		return fmt.Errorf("Configuration validation error in %s:\n  %v\n\nPlease check your configuration file and fix the issue above.", component, err)
	}

	// Check if it's a config file not found error
	if errors.Is(err, config.ErrConfigFileNotFound) {
		return fmt.Errorf("Configuration file not found:\n  %v\n\nPlease create a configuration file or specify the correct path with --config flag.", err)
	}

	// Check if it contains "validation failed" in the error message
	if strings.Contains(err.Error(), "validation failed") {
		return fmt.Errorf("Configuration validation error in %s:\n  %v\n\nPlease check your configuration file and fix the validation errors above.", component, err)
	}

	// Generic error
	return fmt.Errorf("Failed to initialize %s:\n  %v", component, err)
}
