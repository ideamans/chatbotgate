package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
	"github.com/ideamans/multi-oauth2-proxy/pkg/logging"
	"github.com/ideamans/multi-oauth2-proxy/pkg/manager"
	"github.com/ideamans/multi-oauth2-proxy/pkg/session"
)

// createTestConfigFile creates a temporary config file for testing
func createTestConfigFile(t *testing.T, serviceName string, allowedEmails []string) string {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	allowedStr := ""
	for _, email := range allowedEmails {
		allowedStr += fmt.Sprintf("\n    - \"%s\"", email)
	}

	content := fmt.Sprintf(`
service:
  name: "%s"
  description: "Test Service"

server:
  auth_path_prefix: "/_auth"

proxy:
  upstream: "http://localhost:8080"

session:
  cookie_name: "_test_session"
  cookie_secret: "test-secret-key-with-32-characters"
  cookie_expire: "1h"
  cookie_secure: false
  cookie_httponly: true
  cookie_samesite: "lax"

oauth2:
  providers:
    - name: "test"
      type: "google"
      display_name: "Test Provider"
      client_id: "test-client-id"
      client_secret: "test-client-secret"
      enabled: true

authorization:
  allowed:%s

logging:
  level: "info"
`, serviceName, allowedStr)

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	return configPath
}

// updateConfigFile updates an existing config file
func updateConfigFile(t *testing.T, configPath, serviceName string, allowedEmails []string) {
	t.Helper()

	allowedStr := ""
	for _, email := range allowedEmails {
		allowedStr += fmt.Sprintf("\n    - \"%s\"", email)
	}

	content := fmt.Sprintf(`
service:
  name: "%s"
  description: "Test Service"

server:
  auth_path_prefix: "/_auth"

proxy:
  upstream: "http://localhost:8080"

session:
  cookie_name: "_test_session"
  cookie_secret: "test-secret-key-with-32-characters"
  cookie_expire: "1h"
  cookie_secure: false
  cookie_httponly: true
  cookie_samesite: "lax"

oauth2:
  providers:
    - name: "test"
      type: "google"
      display_name: "Test Provider"
      client_id: "test-client-id"
      client_secret: "test-client-secret"
      enabled: true

authorization:
  allowed:%s

logging:
  level: "info"
`, serviceName, allowedStr)

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to update test config: %v", err)
	}

	// Small delay to ensure file modification time changes
	time.Sleep(10 * time.Millisecond)
}

func TestConfigWatcher_New(t *testing.T) {
	configPath := createTestConfigFile(t, "Test Service", []string{"user@example.com"})
	loader := config.NewFileLoader(configPath)

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	mgr, err := manager.New(manager.ManagerConfig{
		Config:       cfg,
		Host:         "localhost",
		Port:         4180,
		SessionStore: sessionStore,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	watcher, err := New(WatcherConfig{
		Loader:     loader,
		Manager:    mgr,
		ConfigPath: configPath,
		Logger:     logger,
	})

	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if watcher == nil {
		t.Fatal("Watcher is nil")
	}
}

func TestConfigWatcher_New_Validation(t *testing.T) {
	configPath := createTestConfigFile(t, "Test Service", nil)
	loader := config.NewFileLoader(configPath)

	cfg, _ := loader.Load()
	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	mgr, _ := manager.New(manager.ManagerConfig{
		Config:       cfg,
		Host:         "localhost",
		Port:         4180,
		SessionStore: sessionStore,
		Logger:       logger,
	})

	tests := []struct {
		name      string
		config    WatcherConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: WatcherConfig{
				Loader:     loader,
				Manager:    mgr,
				ConfigPath: configPath,
				Logger:     logger,
			},
			wantError: false,
		},
		{
			name: "missing loader",
			config: WatcherConfig{
				Loader:     nil,
				Manager:    mgr,
				ConfigPath: configPath,
				Logger:     logger,
			},
			wantError: true,
		},
		{
			name: "missing manager",
			config: WatcherConfig{
				Loader:     loader,
				Manager:    nil,
				ConfigPath: configPath,
				Logger:     logger,
			},
			wantError: true,
		},
		{
			name: "missing logger",
			config: WatcherConfig{
				Loader:     loader,
				Manager:    mgr,
				ConfigPath: configPath,
				Logger:     nil,
			},
			wantError: true,
		},
		{
			name: "missing config path",
			config: WatcherConfig{
				Loader:     loader,
				Manager:    mgr,
				ConfigPath: "",
				Logger:     logger,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("New() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestConfigWatcher_Watch_DetectsChanges(t *testing.T) {
	configPath := createTestConfigFile(t, "Service V1", []string{"user1@example.com"})
	loader := config.NewFileLoader(configPath)

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	mgr, err := manager.New(manager.ManagerConfig{
		Config:       cfg,
		Host:         "localhost",
		Port:         4180,
		SessionStore: sessionStore,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create channel to receive reload notifications
	reloadNotify := make(chan struct{}, 10)

	watcher, err := New(WatcherConfig{
		Loader:       loader,
		Manager:      mgr,
		ConfigPath:   configPath,
		Logger:       logger,
		ReloadNotify: reloadNotify,
	})
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Start watching in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go watcher.Watch(ctx)

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Verify initial config
	currentCfg := mgr.GetConfig()
	if currentCfg.Service.Name != "Service V1" {
		t.Errorf("Initial service name = %s, want Service V1", currentCfg.Service.Name)
	}

	// Update config file
	updateConfigFile(t, configPath, "Service V2", []string{"user2@example.com"})

	// Wait for watcher to detect change and reload (fsnotify should be fast)
	select {
	case <-reloadNotify:
		// OK, reload notification received
	case <-time.After(1 * time.Second):
		t.Fatal("Watcher did not detect file change")
	}

	// Verify config was reloaded
	newCfg := mgr.GetConfig()
	if newCfg.Service.Name != "Service V2" {
		t.Errorf("Reloaded service name = %s, want Service V2", newCfg.Service.Name)
	}

	if len(newCfg.Authorization.Allowed) != 1 || newCfg.Authorization.Allowed[0] != "user2@example.com" {
		t.Errorf("Reloaded allowed list = %v, want [user2@example.com]", newCfg.Authorization.Allowed)
	}
}

func TestConfigWatcher_Watch_HandlesInvalidConfig(t *testing.T) {
	configPath := createTestConfigFile(t, "Service V1", []string{"user1@example.com"})
	loader := config.NewFileLoader(configPath)

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	mgr, err := manager.New(manager.ManagerConfig{
		Config:       cfg,
		Host:         "localhost",
		Port:         4180,
		SessionStore: sessionStore,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	reloadNotify := make(chan struct{}, 10)

	watcher, err := New(WatcherConfig{
		Loader:       loader,
		Manager:      mgr,
		ConfigPath:   configPath,
		Logger:       logger,
		ReloadNotify: reloadNotify,
	})
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go watcher.Watch(ctx)

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Write invalid config (missing service name)
	updateConfigFile(t, configPath, "", []string{"user1@example.com"})

	// Wait for watcher to detect change
	select {
	case <-reloadNotify:
		// OK, change detected
	case <-time.After(1 * time.Second):
		t.Fatal("Watcher did not detect file change")
	}

	// Verify old config is still active (reload should have failed)
	time.Sleep(50 * time.Millisecond)
	currentCfg := mgr.GetConfig()
	if currentCfg.Service.Name != "Service V1" {
		t.Errorf("After invalid config, service name = %s, want Service V1 (old config)", currentCfg.Service.Name)
	}
}

func TestConfigWatcher_Watch_ContextCancellation(t *testing.T) {
	configPath := createTestConfigFile(t, "Test Service", nil)
	loader := config.NewFileLoader(configPath)

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	mgr, err := manager.New(manager.ManagerConfig{
		Config:       cfg,
		Host:         "localhost",
		Port:         4180,
		SessionStore: sessionStore,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	watcher, err := New(WatcherConfig{
		Loader:     loader,
		Manager:    mgr,
		ConfigPath: configPath,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		watcher.Watch(ctx)
		close(done)
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for watcher to stop
	select {
	case <-done:
		// OK, watcher stopped
	case <-time.After(500 * time.Millisecond):
		t.Error("Watcher did not stop after context cancellation")
	}
}

func TestCalculateConfigHash(t *testing.T) {
	cfg1 := &config.Config{
		Service: config.ServiceConfig{
			Name: "Service 1",
		},
		Authorization: config.AuthorizationConfig{
			Allowed: []string{"user1@example.com"},
		},
	}

	cfg2 := &config.Config{
		Service: config.ServiceConfig{
			Name: "Service 2",
		},
		Authorization: config.AuthorizationConfig{
			Allowed: []string{"user1@example.com"},
		},
	}

	cfg3 := &config.Config{
		Service: config.ServiceConfig{
			Name: "Service 1",
		},
		Authorization: config.AuthorizationConfig{
			Allowed: []string{"user2@example.com"},
		},
	}

	hash1, err := calculateConfigHash(cfg1)
	if err != nil {
		t.Fatalf("Failed to calculate hash for cfg1: %v", err)
	}

	hash2, err := calculateConfigHash(cfg2)
	if err != nil {
		t.Fatalf("Failed to calculate hash for cfg2: %v", err)
	}

	hash3, err := calculateConfigHash(cfg3)
	if err != nil {
		t.Fatalf("Failed to calculate hash for cfg3: %v", err)
	}

	// Different configs should have different hashes
	if hash1 == hash2 {
		t.Error("cfg1 and cfg2 should have different hashes")
	}

	if hash1 == hash3 {
		t.Error("cfg1 and cfg3 should have different hashes")
	}

	// Same config should have same hash
	hash1Again, _ := calculateConfigHash(cfg1)
	if hash1 != hash1Again {
		t.Error("Same config should produce same hash")
	}
}
