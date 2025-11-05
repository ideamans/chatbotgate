package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/logging"
)

func TestNew(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
service:
  name: "Test Service"
  description: "Test"

server:
  auth_path_prefix: "/_auth"

proxy:
  upstream:
    url: "http://localhost:8080"

session:
  cookie_name: "_test"
  cookie_secret: "test-secret-key-with-32-characters-long"
  cookie_expire: "1h"

kvs:
  default:
    type: "memory"

oauth2:
  providers:
    - name: "google"
      type: "google"
      client_id: "test-client-id"
      client_secret: "test-client-secret"
      enabled: true

email_auth:
  enabled: false
  sender_type: "smtp"

authorization:
  allowed:
    - "test@example.com"

logging:
  level: "info"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	t.Run("valid config", func(t *testing.T) {
		// Create managers
		proxyManager, err := NewProxyManager(configPath, logger)
		if err != nil {
			t.Fatalf("NewProxyManager() error = %v", err)
		}

		middlewareManager, err := NewMiddlewareManager(configPath, "localhost", 4180, proxyManager.Handler(), logger)
		if err != nil {
			t.Fatalf("NewMiddlewareManager() error = %v", err)
		}

		server, err := New(middlewareManager, "localhost", 4180, logger)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if server == nil {
			t.Fatal("New() returned nil server")
		}
		if server.host != "localhost" {
			t.Errorf("server.host = %v, want localhost", server.host)
		}
		if server.port != 4180 {
			t.Errorf("server.port = %v, want 4180", server.port)
		}
	})

	t.Run("invalid config path", func(t *testing.T) {
		_, err := NewProxyManager("/nonexistent/config.yaml", logger)
		if err == nil {
			t.Error("NewProxyManager() expected error for nonexistent config, got nil")
		}
	})
}

func TestServer_Handler(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	// Create a test upstream
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("upstream response"))
	}))
	defer upstream.Close()

	// Create config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
service:
  name: "Test Service"
  description: "Test"

server:
  auth_path_prefix: "/_auth"

proxy:
  upstream:
    url: "` + upstream.URL + `"

session:
  cookie_name: "_test"
  cookie_secret: "test-secret-key-with-32-characters-long"
  cookie_expire: "1h"

kvs:
  default:
    type: "memory"

oauth2:
  providers:
    - name: "google"
      type: "google"
      client_id: "test-client-id"
      client_secret: "test-client-secret"
      enabled: true

email_auth:
  enabled: false
  sender_type: "smtp"

authorization:
  allowed:
    - "test@example.com"

logging:
  level: "info"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create managers
	proxyManager, err := NewProxyManager(configPath, logger)
	if err != nil {
		t.Fatalf("NewProxyManager() error = %v", err)
	}

	middlewareManager, err := NewMiddlewareManager(configPath, "localhost", 4180, proxyManager.Handler(), logger)
	if err != nil {
		t.Fatalf("NewMiddlewareManager() error = %v", err)
	}

	server, err := New(middlewareManager, "localhost", 4180, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	handler := server.Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil")
	}
}

func TestServer_Start(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	// Create a test upstream
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	// Create config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
service:
  name: "Test Service"
  description: "Test"

server:
  auth_path_prefix: "/_auth"

proxy:
  upstream:
    url: "` + upstream.URL + `"

session:
  cookie_name: "_test"
  cookie_secret: "test-secret-key-with-32-characters-long"
  cookie_expire: "1h"

kvs:
  default:
    type: "memory"

oauth2:
  providers:
    - name: "google"
      type: "google"
      client_id: "test-client-id"
      client_secret: "test-client-secret"
      enabled: true

email_auth:
  enabled: false
  sender_type: "smtp"

authorization:
  allowed:
    - "test@example.com"

logging:
  level: "info"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create managers
	proxyManager, err := NewProxyManager(configPath, logger)
	if err != nil {
		t.Fatalf("NewProxyManager() error = %v", err)
	}

	middlewareManager, err := NewMiddlewareManager(configPath, "localhost", 0, proxyManager.Handler(), logger)
	if err != nil {
		t.Fatalf("NewMiddlewareManager() error = %v", err)
	}

	// Use a random available port
	server, err := New(middlewareManager, "localhost", 0, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Start server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start(ctx)
	}()

	// Wait for context timeout
	<-ctx.Done()

	// The server should shut down gracefully
	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Start() error = %v, want nil or ErrServerClosed", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Server did not shut down within timeout")
	}
}

func TestLoadProxyConfig(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() (string, func())
		wantErr bool
	}{
		{
			name: "valid config file",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				configContent := `
proxy:
  upstream:
    url: "http://localhost:8080"
`
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					t.Fatal(err)
				}
				return configPath, func() {}
			},
			wantErr: false,
		},
		{
			name: "file not found",
			setup: func() (string, func()) {
				return "/nonexistent/config.yaml", func() {}
			},
			wantErr: true,
		},
		{
			name: "invalid yaml",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				invalidContent := "invalid: yaml: content: [unclosed"
				if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
					t.Fatal(err)
				}
				return configPath, func() {}
			},
			wantErr: true,
		},
		{
			name: "missing upstream url",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")
				configContent := `
proxy:
  upstream:
    url: ""
`
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					t.Fatal(err)
				}
				return configPath, func() {}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup := tt.setup()
			defer cleanup()

			cfg, err := loadProxyConfig(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadProxyConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cfg.URL == "" {
				t.Error("loadProxyConfig() returned empty upstream URL without error")
			}
		})
	}
}
