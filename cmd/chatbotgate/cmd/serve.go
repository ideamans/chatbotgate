package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/factory"
	"github.com/ideamans/chatbotgate/pkg/logging"
	"github.com/ideamans/chatbotgate/pkg/manager"
	"github.com/ideamans/chatbotgate/pkg/watcher"
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
- Watch for configuration changes
- Handle graceful shutdown on SIGTERM/SIGINT`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	// Load configuration
	loader := config.NewFileLoader(cfgFile)
	cfg, err := loader.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Setup logger
	logLevel := logging.ParseLevel(cfg.Logging.Level)
	logger := logging.NewSimpleLogger("main", logLevel, cfg.Logging.Color)

	logger.Info("Starting chatbotgate", "version", version, "service", cfg.Service.Name)

	// Create factory for all components
	mwFactory := factory.NewDefaultFactory(host, port, logger)

	// Create KVS stores
	sessionKVS, tokenKVS, rateLimitKVS, err := mwFactory.CreateKVSStores(cfg)
	if err != nil {
		logger.Error("Startup failed: could not create KVS stores", "error", err)
		logger.Fatal("Server initialization failed")
	}
	defer sessionKVS.Close()
	defer tokenKVS.Close()
	defer rateLimitKVS.Close()

	// Create session store
	sessionStore := mwFactory.CreateSessionStore(sessionKVS)

	// Create proxy handler
	proxyHandler, err := mwFactory.CreateProxyHandler(cfg)
	if err != nil {
		logger.Error("Startup failed: could not create proxy handler", "error", err)
		logger.Fatal("Server initialization failed")
	}

	// Create middleware manager
	middlewareManager, err := manager.New(manager.ManagerConfig{
		Config:       cfg,
		Factory:      mwFactory,
		SessionStore: sessionStore,
		ProxyHandler: proxyHandler,
		Logger:       logger,
	})
	if err != nil {
		logger.Error("Startup failed: could not create middleware manager", "error", err)
		logger.Fatal("Server initialization failed")
	}

	// Create config watcher
	configWatcher, err := watcher.New(watcher.WatcherConfig{
		Loader:     loader,
		Manager:    middlewareManager,
		ConfigPath: cfgFile,
		Logger:     logger,
	})
	if err != nil {
		logger.Error("Startup failed: could not create config watcher", "error", err)
		logger.Fatal("Server initialization failed")
	}

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start config watcher in background
	go configWatcher.Watch(ctx)
	logger.Debug("Configuration watcher started")

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", host, port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      middlewareManager,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in goroutine
	go func() {
		logger.Debug("Starting HTTP server", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server stopped with error", "error", err)
			os.Exit(1)
		}
	}()

	logger.Info("Server started successfully", "addr", addr)

	// Wait for shutdown signal
	<-stop
	logger.Info("Shutdown signal received, starting graceful shutdown")

	// Cancel config watcher
	cancel()

	// Graceful shutdown of HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", "error", err)
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	logger.Info("Server stopped successfully")
	return nil
}
