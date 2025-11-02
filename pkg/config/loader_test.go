package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileLoader_Load(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantErr  bool
		validate func(*testing.T, *Config)
	}{
		{
			name: "valid config",
			content: `
service:
  name: "Test Service"
  description: "Test Description"

server:
  auth_path_prefix: "/_auth"

proxy:
  upstream: "http://localhost:9090"

session:
  cookie_name: "_test_cookie"
  cookie_secret: "this-is-a-very-long-secret-key-for-testing-purposes"
  cookie_expire: "24h"
  cookie_secure: true
  cookie_httponly: true
  cookie_samesite: "strict"

oauth2:
  providers:
    - name: "google"
      display_name: "Google"
      client_id: "test-client-id"
      client_secret: "test-client-secret"
      enabled: true

authorization:
  allowed_emails:
    - "user@example.com"
  allowed_domains:
    - "@example.org"

logging:
  level: "debug"
  module_level: "info"
  color: true
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Service.Name != "Test Service" {
					t.Errorf("Service.Name = %s, want Test Service", cfg.Service.Name)
				}
				if cfg.Proxy.Upstream != "http://localhost:9090" {
					t.Errorf("Proxy.Upstream = %s, want http://localhost:9090", cfg.Proxy.Upstream)
				}
				if cfg.Session.CookieName != "_test_cookie" {
					t.Errorf("Session.CookieName = %s, want _test_cookie", cfg.Session.CookieName)
				}
				if len(cfg.OAuth2.Providers) != 1 {
					t.Errorf("len(OAuth2.Providers) = %d, want 1", len(cfg.OAuth2.Providers))
				}
			},
		},
		{
			name: "apply defaults",
			content: `
service:
  name: "Test Service"

server:
  auth_path_prefix: "/_auth"

proxy:
  upstream: "http://localhost:8080"

session:
  cookie_secret: "this-is-a-very-long-secret-key-for-testing-purposes"

oauth2:
  providers:
    - name: "google"
      enabled: true

authorization:
  allowed_emails:
    - "user@example.com"
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Session.CookieName != "_oauth2_proxy" {
					t.Errorf("Session.CookieName = %s, want _oauth2_proxy (default)", cfg.Session.CookieName)
				}
				if cfg.Session.CookieExpire != "168h" {
					t.Errorf("Session.CookieExpire = %s, want 168h (default)", cfg.Session.CookieExpire)
				}
				if cfg.Logging.Level != "info" {
					t.Errorf("Logging.Level = %s, want info (default)", cfg.Logging.Level)
				}
			},
		},
		{
			name: "invalid YAML",
			content: `
this is not valid yaml: [
`,
			wantErr: true,
		},
		{
			name: "incomplete configuration - loads successfully (validation is done by manager)",
			content: `
server:
  auth_path_prefix: "/_auth"

proxy:
  upstream: "http://localhost:8080"

session:
  cookie_secret: "this-is-a-very-long-secret-key-for-testing-purposes"

oauth2:
  providers:
    - name: "google"
      enabled: true
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				// Service name is empty, but loader doesn't validate
				// Validation is performed by the middleware manager
				if cfg.Service.Name != "" {
					t.Errorf("Service.Name = %s, want empty", cfg.Service.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			loader := NewFileLoader(configPath)
			cfg, err := loader.Load()

			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestFileLoader_Load_FileNotFound(t *testing.T) {
	loader := NewFileLoader("/nonexistent/path/config.yaml")
	_, err := loader.Load()

	if err == nil {
		t.Error("Load() should return error for non-existent file")
	}
}

func TestFileLoader_Load_JSON(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantErr  bool
		validate func(*testing.T, *Config)
	}{
		{
			name: "valid JSON config",
			content: `{
  "service": {
    "name": "Test Service",
    "description": "Test Description"
  },
  "server": {
    "auth_path_prefix": "/_auth"
  },
  "proxy": {
    "upstream": "http://localhost:9090"
  },
  "session": {
    "cookie_name": "_test_cookie",
    "cookie_secret": "this-is-a-very-long-secret-key-for-testing-purposes",
    "cookie_expire": "24h",
    "cookie_secure": true,
    "cookie_httponly": true,
    "cookie_samesite": "strict"
  },
  "oauth2": {
    "providers": [
      {
        "name": "google",
        "display_name": "Google",
        "client_id": "test-client-id",
        "client_secret": "test-client-secret",
        "enabled": true
      }
    ]
  },
  "authorization": {
    "allowed_emails": ["user@example.com"],
    "allowed_domains": ["@example.org"]
  },
  "logging": {
    "level": "debug",
    "module_level": "info",
    "color": true
  }
}`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Service.Name != "Test Service" {
					t.Errorf("Service.Name = %s, want Test Service", cfg.Service.Name)
				}
				if cfg.Proxy.Upstream != "http://localhost:9090" {
					t.Errorf("Proxy.Upstream = %s, want http://localhost:9090", cfg.Proxy.Upstream)
				}
				if cfg.Session.CookieName != "_test_cookie" {
					t.Errorf("Session.CookieName = %s, want _test_cookie", cfg.Session.CookieName)
				}
				if len(cfg.OAuth2.Providers) != 1 {
					t.Errorf("len(OAuth2.Providers) = %d, want 1", len(cfg.OAuth2.Providers))
				}
			},
		},
		{
			name: "JSON with defaults",
			content: `{
  "service": {"name": "Test Service"},
  "server": {"auth_path_prefix": "/_auth"},
  "proxy": {"upstream": "http://localhost:8080"},
  "session": {
    "cookie_secret": "this-is-a-very-long-secret-key-for-testing-purposes"
  },
  "oauth2": {
    "providers": [{"name": "google", "enabled": true}]
  },
  "authorization": {
    "allowed_emails": ["user@example.com"]
  }
}`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Session.CookieName != "_oauth2_proxy" {
					t.Errorf("Session.CookieName = %s, want _oauth2_proxy (default)", cfg.Session.CookieName)
				}
			},
		},
		{
			name:    "invalid JSON",
			content: `{"invalid": json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")

			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			loader := NewFileLoader(configPath)
			cfg, err := loader.Load()

			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestFileLoader_Load_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.txt")

	content := "this is a text file"
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewFileLoader(configPath)
	_, err := loader.Load()

	if err == nil {
		t.Error("Load() should return error for unsupported file format")
	}
}
