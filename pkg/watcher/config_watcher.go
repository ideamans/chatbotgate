package watcher

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
	"github.com/ideamans/multi-oauth2-proxy/pkg/logging"
	"github.com/ideamans/multi-oauth2-proxy/pkg/manager"
)

// ConfigWatcher watches configuration changes and triggers middleware reloads.
type ConfigWatcher struct {
	loader       config.Loader
	manager      *manager.MiddlewareManager
	configPath   string
	lastHash     string
	logger       logging.Logger
	reloadNotify chan struct{} // Optional channel for testing
}

// WatcherConfig contains the configuration for creating a ConfigWatcher
type WatcherConfig struct {
	Loader       config.Loader
	Manager      *manager.MiddlewareManager
	ConfigPath   string // Path to the configuration file to watch
	Logger       logging.Logger
	ReloadNotify chan struct{} // Optional: notified after each reload attempt
}

// New creates a new ConfigWatcher with the given configuration.
func New(cfg WatcherConfig) (*ConfigWatcher, error) {
	if cfg.Loader == nil {
		return nil, fmt.Errorf("loader is required")
	}
	if cfg.Manager == nil {
		return nil, fmt.Errorf("manager is required")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg.ConfigPath == "" {
		return nil, fmt.Errorf("config path is required")
	}

	// Get absolute path
	absPath, err := filepath.Abs(cfg.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Calculate initial hash
	currentConfig := cfg.Manager.GetConfig()
	hash, err := calculateConfigHash(currentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate initial config hash: %w", err)
	}

	watcher := &ConfigWatcher{
		loader:       cfg.Loader,
		manager:      cfg.Manager,
		configPath:   absPath,
		lastHash:     hash,
		logger:       cfg.Logger.WithModule("watcher"),
		reloadNotify: cfg.ReloadNotify,
	}

	watcher.logger.Info("ConfigWatcher initialized", "config_path", absPath)

	return watcher, nil
}

// Watch starts watching for configuration changes using fsnotify.
// Call this method in a goroutine. It blocks until the context is cancelled.
func (w *ConfigWatcher) Watch(ctx context.Context) {
	w.logger.Info("Starting configuration watch")

	// Create fsnotify watcher
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		w.logger.Error("Failed to create fsnotify watcher", "error", err)
		return
	}
	defer fsWatcher.Close()

	// Watch the config file
	err = fsWatcher.Add(w.configPath)
	if err != nil {
		w.logger.Error("Failed to watch config file", "error", err, "path", w.configPath)
		return
	}

	w.logger.Info("Watching configuration file", "path", w.configPath)

	// Debounce timer to handle multiple rapid events
	var debounceTimer *time.Timer
	debounceDuration := 100 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Configuration watch stopped")
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-fsWatcher.Events:
			if !ok {
				w.logger.Warn("fsnotify events channel closed")
				return
			}

			// Only care about write and create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				w.logger.Debug("Config file changed", "event", event.Op.String())

				// Reset debounce timer
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDuration, func() {
					w.checkAndReload()
				})
			}

			// If file was removed and recreated (common with some editors)
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				w.logger.Debug("Config file removed, will re-watch on create")
				// Wait a bit and try to re-add the watch
				time.Sleep(50 * time.Millisecond)
				fsWatcher.Add(w.configPath)
			}

		case err, ok := <-fsWatcher.Errors:
			if !ok {
				w.logger.Warn("fsnotify errors channel closed")
				return
			}
			w.logger.Error("fsnotify error", "error", err)
		}
	}
}

// checkAndReload checks if the configuration has changed and reloads if necessary.
func (w *ConfigWatcher) checkAndReload() {
	// Notify test if channel is set
	if w.reloadNotify != nil {
		defer func() {
			select {
			case w.reloadNotify <- struct{}{}:
			default:
			}
		}()
	}

	// Load new configuration
	newConfig, err := w.loader.Load()
	if err != nil {
		w.logger.Error("Failed to load configuration", "error", err)
		return
	}

	// Calculate new hash
	newHash, err := calculateConfigHash(newConfig)
	if err != nil {
		w.logger.Error("Failed to calculate config hash", "error", err)
		return
	}

	// Check if configuration has changed
	if newHash == w.lastHash {
		w.logger.Debug("Configuration unchanged")
		return
	}

	w.logger.Info("Configuration changed, reloading middleware")

	// Reload middleware
	if err := w.manager.Reload(newConfig); err != nil {
		w.logger.Error("Failed to reload middleware", "error", err)
		return
	}

	// Update last hash
	w.lastHash = newHash
	w.logger.Info("Configuration reloaded successfully")
}

// calculateConfigHash calculates a hash of the configuration for change detection.
// We use JSON marshaling to create a canonical representation, then hash it.
func calculateConfigHash(cfg *config.Config) (string, error) {
	// Marshal config to JSON for canonical representation
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	// Calculate SHA-256 hash
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}
