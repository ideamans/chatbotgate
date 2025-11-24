package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	proxy "github.com/ideamans/chatbotgate/pkg/proxy/core"
	"github.com/ideamans/chatbotgate/pkg/shared/filewatcher"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// TestValidateProxyConfig tests the validateProxyConfig function
func TestValidateProxyConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *ProxyConfig
		expectError bool
		checkError  string
	}{
		{
			name: "Valid configuration",
			cfg: &ProxyConfig{
				Proxy: ProxyServerConfig{
					Upstream: proxy.UpstreamConfig{
						URL: "http://localhost:8080",
					},
				},
			},
			expectError: false,
		},
		{
			name: "Valid configuration with HTTPS",
			cfg: &ProxyConfig{
				Proxy: ProxyServerConfig{
					Upstream: proxy.UpstreamConfig{
						URL: "https://example.com:443",
					},
				},
			},
			expectError: false,
		},
		{
			name: "Missing upstream URL",
			cfg: &ProxyConfig{
				Proxy: ProxyServerConfig{
					Upstream: proxy.UpstreamConfig{
						URL: "",
					},
				},
			},
			expectError: true,
			checkError:  "proxy.upstream.url is required",
		},
		{
			name: "Invalid URL format (missing scheme)",
			cfg: &ProxyConfig{
				Proxy: ProxyServerConfig{
					Upstream: proxy.UpstreamConfig{
						URL: ":invalid",
					},
				},
			},
			expectError: true,
			checkError:  "proxy.upstream.url is not a valid URL",
		},
		{
			name: "Secret header without value",
			cfg: &ProxyConfig{
				Proxy: ProxyServerConfig{
					Upstream: proxy.UpstreamConfig{
						URL: "http://localhost:8080",
						Secret: proxy.SecretConfig{
							Header: "X-Secret-Token",
							Value:  "",
						},
					},
				},
			},
			expectError: true,
			checkError:  "proxy.upstream.secret.value is required when header is specified",
		},
		{
			name: "Valid configuration with secret",
			cfg: &ProxyConfig{
				Proxy: ProxyServerConfig{
					Upstream: proxy.UpstreamConfig{
						URL: "http://localhost:8080",
						Secret: proxy.SecretConfig{
							Header: "X-Secret-Token",
							Value:  "secret-value-123",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Secret value without header (valid)",
			cfg: &ProxyConfig{
				Proxy: ProxyServerConfig{
					Upstream: proxy.UpstreamConfig{
						URL: "http://localhost:8080",
						Secret: proxy.SecretConfig{
							Header: "",
							Value:  "secret-value-123",
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProxyConfig(tt.cfg)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if tt.checkError != "" && !strings.Contains(err.Error(), tt.checkError) {
					t.Errorf("Expected error to contain %q, but got: %s", tt.checkError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestLoadProxyConfig tests the loadProxyConfig function
func TestLoadProxyConfig(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		filename    string
		content     string
		expectedURL string
		expectError bool
		checkError  string
	}{
		{
			name:     "Valid YAML configuration",
			filename: "valid.yaml",
			content: `
proxy:
  upstream:
    url: "http://localhost:8080"
`,
			expectedURL: "http://localhost:8080",
			expectError: false,
		},
		{
			name:     "Valid JSON configuration",
			filename: "valid.json",
			content: `{
  "proxy": {
    "upstream": {
      "url": "http://localhost:9000"
    }
  }
}`,
			expectedURL: "http://localhost:9000",
			expectError: false,
		},
		{
			name:     "Valid YAML with secret",
			filename: "with-secret.yaml",
			content: `
proxy:
  upstream:
    url: "https://api.example.com"
    secret:
      header: "X-API-Key"
      value: "secret-key-123"
`,
			expectedURL: "https://api.example.com",
			expectError: false,
		},
		{
			name:     "Missing upstream URL",
			filename: "missing-url.yaml",
			content: `
proxy:
  upstream:
    url: ""
`,
			expectError: true,
			checkError:  "proxy.upstream.url is required",
		},
		{
			name:     "Empty proxy section",
			filename: "empty-proxy.yaml",
			content: `
proxy:
  upstream:
`,
			expectError: true,
			checkError:  "proxy.upstream.url is required",
		},
		{
			name:        "Non-existent file",
			filename:    "non-existent.yaml",
			content:     "", // won't be created
			expectError: true,
			checkError:  "failed to read config file",
		},
		{
			name:     "Invalid YAML",
			filename: "invalid.yaml",
			content: `
proxy:
  upstream: [this is not valid
`,
			expectError: true,
			checkError:  "failed to parse YAML config file",
		},
		{
			name:     "Invalid JSON",
			filename: "invalid.json",
			content: `{
  "proxy": {
    "upstream": [this is not valid JSON
  }
}`,
			expectError: true,
			checkError:  "failed to parse JSON config file",
		},
		{
			name:        "Unsupported file format",
			filename:    "unsupported.txt",
			content:     "proxy:\n  upstream:\n    url: http://localhost:8080",
			expectError: true,
			checkError:  "unsupported config file format",
		},
		{
			name:     "Secret header without value",
			filename: "secret-no-value.yaml",
			content: `
proxy:
  upstream:
    url: "http://localhost:8080"
    secret:
      header: "X-Secret"
      value: ""
`,
			expectError: true,
			checkError:  "proxy.upstream.secret.value is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string
			if tt.name != "Non-existent file" {
				configPath = filepath.Join(tmpDir, tt.filename)
				if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("Failed to create test config: %v", err)
				}
			} else {
				configPath = filepath.Join(tmpDir, tt.filename)
			}

			cfg, err := loadProxyConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if tt.checkError != "" && !strings.Contains(err.Error(), tt.checkError) {
					t.Errorf("Expected error to contain %q, but got: %s", tt.checkError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("loadProxyConfig() error = %v, expectError = %v", err, tt.expectError)
			}

			if cfg.URL != tt.expectedURL {
				t.Errorf("URL = %v, want %v", cfg.URL, tt.expectedURL)
			}
		})
	}
}

// TestNewProxyManager tests the NewProxyManager function
func TestNewProxyManager(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		filename    string
		content     string
		expectError bool
		checkError  string
	}{
		{
			name:     "Valid configuration",
			filename: "valid.yaml",
			content: `
proxy:
  upstream:
    url: "http://localhost:8080"
`,
			expectError: false,
		},
		{
			name:     "Valid configuration with secret",
			filename: "with-secret.yaml",
			content: `
proxy:
  upstream:
    url: "http://localhost:8080"
    secret:
      header: "X-Secret"
      value: "test-secret"
`,
			expectError: false,
		},
		{
			name:     "Missing upstream URL",
			filename: "missing-url.yaml",
			content: `
proxy:
  upstream:
    url: ""
`,
			expectError: true,
			checkError:  "failed to build initial proxy handler",
		},
		{
			name:     "Invalid URL",
			filename: "invalid-url.yaml",
			content: `
proxy:
  upstream:
    url: "://invalid"
`,
			expectError: true,
			checkError:  "failed to build initial proxy handler",
		},
		{
			name:        "Non-existent config file",
			filename:    "non-existent.yaml",
			content:     "", // won't be created
			expectError: true,
			checkError:  "failed to build initial proxy handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string
			if tt.name != "Non-existent config file" {
				configPath = filepath.Join(tmpDir, tt.filename)
				if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("Failed to create test config: %v", err)
				}
			} else {
				configPath = filepath.Join(tmpDir, tt.filename)
			}

			logger := logging.NewSimpleLogger("test", logging.LevelError, false)

			manager, err := NewProxyManager(configPath, logger)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if tt.checkError != "" && !strings.Contains(err.Error(), tt.checkError) {
					t.Errorf("Expected error to contain %q, but got: %s", tt.checkError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("NewProxyManager() error = %v, expectError = %v", err, tt.expectError)
			}

			if manager == nil {
				t.Fatal("Expected manager to be non-nil")
			}

			// Test that Handler() returns a non-nil handler
			handler := manager.Handler()
			if handler == nil {
				t.Error("Expected Handler() to return non-nil")
			}
		})
	}
}

// TestProxyManagerHandler tests the Handler method
func TestProxyManagerHandler(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test upstream server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer testServer.Close()

	configPath := filepath.Join(tmpDir, "test.yaml")
	content := `
proxy:
  upstream:
    url: "` + testServer.URL + `"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	logger := logging.NewSimpleLogger("test", logging.LevelError, false)
	manager, err := NewProxyManager(configPath, logger)
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}

	// Get handler
	handler := manager.Handler()
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}

	// Test that handler can process a request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should get response from upstream
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if body != "test response" {
		t.Errorf("Expected body 'test response', got %q", body)
	}
}

// TestProxyManagerOnFileChange tests the OnFileChange method
func TestProxyManagerOnFileChange(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial config
	configPath := filepath.Join(tmpDir, "test.yaml")
	content := `
proxy:
  upstream:
    url: "http://localhost:8080"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	logger := logging.NewSimpleLogger("test", logging.LevelError, false)
	manager, err := NewProxyManager(configPath, logger)
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}

	// Test successful reload with valid config
	newContent := `
proxy:
  upstream:
    url: "http://localhost:9000"
`
	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to write new config: %v", err)
	}

	event := filewatcher.ChangeEvent{
		Path:  configPath,
		Error: nil,
	}

	// OnFileChange should reload successfully
	manager.OnFileChange(event)

	// Handler should still work after reload
	handler := manager.Handler()
	if handler == nil {
		t.Error("Expected non-nil handler after reload")
	}

	// Test reload with invalid config (should keep old config)
	invalidContent := `
proxy:
  upstream:
    url: ""
`
	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	manager.OnFileChange(event)

	// Handler should still work (using old config)
	handler = manager.Handler()
	if handler == nil {
		t.Error("Expected non-nil handler after failed reload")
	}

	// Test OnFileChange with error event
	errorEvent := filewatcher.ChangeEvent{
		Path:  configPath,
		Error: os.ErrNotExist,
	}

	// Should not panic
	manager.OnFileChange(errorEvent)
}

// TestNewProxyManagerWithNilLogger tests that NewProxyManager works with nil logger
func TestNewProxyManagerWithNilLogger(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "test.yaml")
	content := `
proxy:
  upstream:
    url: "http://localhost:8080"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test with nil logger - should use default logger
	manager, err := NewProxyManager(configPath, nil)
	if err != nil {
		t.Fatalf("NewProxyManager() with nil logger error = %v", err)
	}

	if manager == nil {
		t.Fatal("Expected manager to be non-nil")
	}

	// Handler should still work
	handler := manager.Handler()
	if handler == nil {
		t.Error("Expected Handler() to return non-nil")
	}
}
