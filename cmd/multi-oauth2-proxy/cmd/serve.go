package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
	"github.com/ideamans/multi-oauth2-proxy/pkg/kvs"
	"github.com/ideamans/multi-oauth2-proxy/pkg/logging"
	"github.com/ideamans/multi-oauth2-proxy/pkg/manager"
	"github.com/ideamans/multi-oauth2-proxy/pkg/proxy"
	"github.com/ideamans/multi-oauth2-proxy/pkg/session"
	"github.com/ideamans/multi-oauth2-proxy/pkg/watcher"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the authentication proxy server",
	Long: `Start the multi-oauth2-proxy server with the specified configuration.

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

	logger.Info("Starting multi-oauth2-proxy", "version", version, "service", cfg.Service.Name)

	// Initialize KVS stores with namespace isolation
	// Set default namespace names
	cfg.KVS.Namespaces.SetDefaults()
	logger.Debug("Namespace configuration",
		"session", cfg.KVS.Namespaces.Session,
		"token", cfg.KVS.Namespaces.Token,
		"ratelimit", cfg.KVS.Namespaces.RateLimit)

	// Default KVS type fallback
	if cfg.KVS.Default.Type == "" {
		cfg.KVS.Default.Type = "memory"
	}

	// Track KVS stores that need to be closed
	var kvsToClose []kvs.Store
	defer func() {
		for _, store := range kvsToClose {
			store.Close()
		}
	}()

	// Initialize session KVS (override or default with namespace)
	var sessionKVS kvs.Store
	if cfg.KVS.Session != nil {
		// Dedicated session KVS specified
		sessionKVS, err = kvs.New(*cfg.KVS.Session)
		if err != nil {
			logger.Error("Startup failed: could not create session KVS", "error", err)
			logger.Fatal("Server initialization failed")
		}
		logger.Debug("Session KVS initialized (dedicated)", "type", cfg.KVS.Session.Type, "namespace", cfg.KVS.Session.Namespace)
	} else {
		// Use default KVS config with session namespace
		sessionCfg := cfg.KVS.Default
		sessionCfg.Namespace = cfg.KVS.Namespaces.Session
		sessionKVS, err = kvs.New(sessionCfg)
		if err != nil {
			logger.Error("Startup failed: could not create session KVS", "error", err)
			logger.Fatal("Server initialization failed")
		}
		logger.Debug("Session KVS initialized (default)", "type", sessionCfg.Type, "namespace", sessionCfg.Namespace)
	}
	kvsToClose = append(kvsToClose, sessionKVS)

	// Initialize token KVS (override or default with namespace)
	var tokenKVS kvs.Store
	if cfg.KVS.Token != nil {
		// Dedicated token KVS specified
		tokenKVS, err = kvs.New(*cfg.KVS.Token)
		if err != nil {
			logger.Error("Startup failed: could not create token KVS", "error", err)
			logger.Fatal("Server initialization failed")
		}
		logger.Debug("Token KVS initialized (dedicated)", "type", cfg.KVS.Token.Type, "namespace", cfg.KVS.Token.Namespace)
	} else {
		// Use default KVS config with token namespace
		tokenCfg := cfg.KVS.Default
		tokenCfg.Namespace = cfg.KVS.Namespaces.Token
		tokenKVS, err = kvs.New(tokenCfg)
		if err != nil {
			logger.Error("Startup failed: could not create token KVS", "error", err)
			logger.Fatal("Server initialization failed")
		}
		logger.Debug("Token KVS initialized (default)", "type", tokenCfg.Type, "namespace", tokenCfg.Namespace)
	}
	kvsToClose = append(kvsToClose, tokenKVS)

	// Initialize rate limit KVS (override or default with namespace)
	var rateLimitKVS kvs.Store
	if cfg.KVS.RateLimit != nil {
		// Dedicated rate limit KVS specified
		rateLimitKVS, err = kvs.New(*cfg.KVS.RateLimit)
		if err != nil {
			logger.Error("Startup failed: could not create rate limit KVS", "error", err)
			logger.Fatal("Server initialization failed")
		}
		logger.Debug("Rate limit KVS initialized (dedicated)", "type", cfg.KVS.RateLimit.Type, "namespace", cfg.KVS.RateLimit.Namespace)
	} else {
		// Use default KVS config with ratelimit namespace
		rateLimitCfg := cfg.KVS.Default
		rateLimitCfg.Namespace = cfg.KVS.Namespaces.RateLimit
		rateLimitKVS, err = kvs.New(rateLimitCfg)
		if err != nil {
			logger.Error("Startup failed: could not create rate limit KVS", "error", err)
			logger.Fatal("Server initialization failed")
		}
		logger.Debug("Rate limit KVS initialized (default)", "type", rateLimitCfg.Type, "namespace", rateLimitCfg.Namespace)
	}
	kvsToClose = append(kvsToClose, rateLimitKVS)

	// Create session store using KVS
	sessionStore := session.NewKVSStore(sessionKVS)

	// Create proxy handler if upstream is configured
	var proxyHandler *proxy.Handler
	if len(cfg.Proxy.Hosts) > 0 {
		proxyHandler, err = proxy.NewHandlerWithHosts(cfg.Proxy.Upstream, cfg.Proxy.Hosts)
		if err != nil {
			logger.Error("Startup failed: could not create proxy handler", "error", err)
			logger.Fatal("Server initialization failed")
		}
		logger.Debug("Proxy handler initialized with host routing",
			"default_upstream", cfg.Proxy.Upstream,
			"hosts", len(cfg.Proxy.Hosts))
	} else {
		proxyHandler, err = proxy.NewHandler(cfg.Proxy.Upstream)
		if err != nil {
			logger.Error("Startup failed: could not create proxy handler", "error", err)
			logger.Fatal("Server initialization failed")
		}
		logger.Debug("Proxy handler initialized", "upstream", cfg.Proxy.Upstream)
	}

	// Create middleware manager
	middlewareManager, err := manager.New(manager.ManagerConfig{
		Config:       cfg,
		Host:         host,
		Port:         port,
		SessionStore: sessionStore,
		ProxyHandler: proxyHandler,
		TokenKVS:     tokenKVS,
		RateLimitKVS: rateLimitKVS,
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
