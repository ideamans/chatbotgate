package watcher

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/logging"
	"github.com/ideamans/chatbotgate/pkg/manager"
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

	watcher.logger.Debug("Config watcher initialized", "path", absPath)

	return watcher, nil
}

// Watch starts watching for configuration changes using fsnotify.
// Call this method in a goroutine. It blocks until the context is cancelled.
func (w *ConfigWatcher) Watch(ctx context.Context) {
	w.logger.Debug("Starting configuration file watch")

	// Create fsnotify watcher
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		w.logger.Debug("fsnotify watcher creation failed", "error", err)
		w.logger.Error("Config watcher startup failed: could not create file watcher")
		return
	}
	defer fsWatcher.Close()

	// Watch the config file
	err = fsWatcher.Add(w.configPath)
	if err != nil {
		w.logger.Debug("File watch add failed", "error", err, "path", w.configPath)
		w.logger.Error("Config watcher startup failed: could not watch config file", "path", w.configPath)
		return
	}

	w.logger.Debug("Config file watch started", "path", w.configPath)

	// Debounce timer to handle multiple rapid events
	var debounceTimer *time.Timer
	debounceDuration := 100 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			w.logger.Debug("Config file watch stopped")
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-fsWatcher.Events:
			if !ok {
				w.logger.Warn("File watch stopped: events channel closed unexpectedly")
				return
			}

			// Only care about write and create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				w.logger.Debug("Config file change detected", "event", event.Op.String())

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
				w.logger.Warn("File watch stopped: errors channel closed unexpectedly")
				return
			}
			w.logger.Error("File watch error", "error", err)
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
		w.logger.Debug("Config load failed", "error", err)
		w.logger.Error("Config reload failed: could not load configuration file")
		return
	}

	// Calculate new hash
	newHash, err := calculateConfigHash(newConfig)
	if err != nil {
		w.logger.Debug("Hash calculation failed", "error", err)
		w.logger.Error("Config reload failed: could not calculate config hash")
		return
	}

	// Check if configuration has changed
	if newHash == w.lastHash {
		w.logger.Debug("Config file changed but content unchanged")
		return
	}

	w.logger.Debug("Config content change detected, starting reload")

	// Reload middleware
	if err := w.manager.Reload(newConfig); err != nil {
		// Error already logged by manager
		return
	}

	// Update last hash
	w.lastHash = newHash
	// Success already logged by manager
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
