package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "config.yaml")

	// Create initial config file
	initialConfig := `
service:
  name: "Test Service"
server:
  host: "localhost"
  port: 4180
proxy:
  upstream: "http://localhost:8080"
session:
  cookie_name: "_test"
  cookie_secret: "test-secret-key-with-32-characters"
  cookie_expire: "1h"
oauth2:
  providers:
    - name: "google"
      client_id: "test-client-id"
      client_secret: "test-client-secret"
      enabled: true
authorization:
  allowed_emails:
    - "test@example.com"
logging:
  level: "info"
`
	if err := os.WriteFile(tempFile, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	watcher, err := NewWatcher(tempFile, nil)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer watcher.Stop()

	if watcher == nil {
		t.Error("NewWatcher() returned nil")
	}

	if watcher.path != tempFile {
		t.Errorf("path = %s, want %s", watcher.path, tempFile)
	}
}

func TestWatcher_ConfigReload(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "config.yaml")

	// Create initial config file
	initialConfig := `
service:
  name: "Initial Service"
server:
  host: "localhost"
  port: 4180
proxy:
  upstream: "http://localhost:8080"
session:
  cookie_name: "_test"
  cookie_secret: "test-secret-key-with-32-characters"
  cookie_expire: "1h"
oauth2:
  providers:
    - name: "google"
      client_id: "test-client-id"
      client_secret: "test-client-secret"
      enabled: true
authorization:
  allowed_emails:
    - "test@example.com"
logging:
  level: "info"
`
	if err := os.WriteFile(tempFile, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Track callback invocations
	callbackCalled := make(chan *Config, 1)
	callback := func(cfg *Config) error {
		callbackCalled <- cfg
		return nil
	}

	watcher, err := NewWatcher(tempFile, callback)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer watcher.Stop()

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait a bit for watcher to be ready
	time.Sleep(100 * time.Millisecond)

	// Modify config file
	updatedConfig := `
service:
  name: "Updated Service"
server:
  host: "localhost"
  port: 4180
proxy:
  upstream: "http://localhost:9090"
session:
  cookie_name: "_test"
  cookie_secret: "test-secret-key-with-32-characters"
  cookie_expire: "1h"
oauth2:
  providers:
    - name: "google"
      client_id: "test-client-id"
      client_secret: "test-client-secret"
      enabled: true
authorization:
  allowed_emails:
    - "test@example.com"
logging:
  level: "debug"
`
	if err := os.WriteFile(tempFile, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("Failed to update test config file: %v", err)
	}

	// Wait for callback
	select {
	case cfg := <-callbackCalled:
		if cfg.Service.Name != "Updated Service" {
			t.Errorf("Service name = %s, want Updated Service", cfg.Service.Name)
		}
		if cfg.Proxy.Upstream != "http://localhost:9090" {
			t.Errorf("Proxy upstream = %s, want http://localhost:9090", cfg.Proxy.Upstream)
		}
		if cfg.Logging.Level != "debug" {
			t.Errorf("Logging level = %s, want debug", cfg.Logging.Level)
		}
	case <-time.After(2 * time.Second):
		t.Error("Callback was not called after config change")
	}
}

func TestWatcher_Stop(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "config.yaml")

	// Create initial config file
	initialConfig := `
service:
  name: "Test Service"
server:
  host: "localhost"
  port: 4180
proxy:
  upstream: "http://localhost:8080"
session:
  cookie_name: "_test"
  cookie_secret: "test-secret-key-with-32-characters"
  cookie_expire: "1h"
oauth2:
  providers:
    - name: "google"
      client_id: "test-client-id"
      client_secret: "test-client-secret"
      enabled: true
authorization:
  allowed_emails:
    - "test@example.com"
logging:
  level: "info"
`
	if err := os.WriteFile(tempFile, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	watcher, err := NewWatcher(tempFile, nil)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Stop watcher
	if err := watcher.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Second stop should be safe
	if err := watcher.Stop(); err != nil {
		t.Errorf("Second Stop() error = %v", err)
	}
}
