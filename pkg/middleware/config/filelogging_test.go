package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfig_FileLoggingYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		check   func(*testing.T, *Config)
	}{
		{
			name: "file logging disabled",
			yaml: `
service:
  name: "Test"
session:
  cookie_secret: "test-secret-at-least-32-chars-long"
oauth2:
  providers:
    - name: "google"
      client_id: "test"
      client_secret: "test"
logging:
  level: "info"
  color: true
proxy:
  upstream:
    url: "http://localhost:8080"
`,
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Logging.File != nil {
					t.Error("File logging should be nil when not configured")
				}
			},
		},
		{
			name: "file logging with path only",
			yaml: `
service:
  name: "Test"
session:
  cookie_secret: "test-secret-at-least-32-chars-long"
oauth2:
  providers:
    - name: "google"
      client_id: "test"
      client_secret: "test"
logging:
  level: "debug"
  color: false
  file:
    path: "/var/log/test.log"
proxy:
  upstream:
    url: "http://localhost:8080"
`,
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Logging.File == nil {
					t.Fatal("File logging config should not be nil")
				}
				if cfg.Logging.File.Path != "/var/log/test.log" {
					t.Errorf("Path = %s, want /var/log/test.log", cfg.Logging.File.Path)
				}
				if cfg.Logging.File.MaxSizeMB != 0 {
					t.Errorf("MaxSizeMB should be 0 (default), got %d", cfg.Logging.File.MaxSizeMB)
				}
			},
		},
		{
			name: "file logging with all options",
			yaml: `
service:
  name: "Test"
session:
  cookie_secret: "test-secret-at-least-32-chars-long"
oauth2:
  providers:
    - name: "google"
      client_id: "test"
      client_secret: "test"
logging:
  level: "info"
  color: true
  file:
    path: "/var/log/chatbotgate/app.log"
    max_size_mb: 50
    max_backups: 10
    max_age: 14
    compress: true
proxy:
  upstream:
    url: "http://localhost:8080"
`,
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Logging.File == nil {
					t.Fatal("File logging config should not be nil")
				}
				if cfg.Logging.File.Path != "/var/log/chatbotgate/app.log" {
					t.Errorf("Path = %s, want /var/log/chatbotgate/app.log", cfg.Logging.File.Path)
				}
				if cfg.Logging.File.MaxSizeMB != 50 {
					t.Errorf("MaxSizeMB = %d, want 50", cfg.Logging.File.MaxSizeMB)
				}
				if cfg.Logging.File.MaxBackups != 10 {
					t.Errorf("MaxBackups = %d, want 10", cfg.Logging.File.MaxBackups)
				}
				if cfg.Logging.File.MaxAge != 14 {
					t.Errorf("MaxAge = %d, want 14", cfg.Logging.File.MaxAge)
				}
				if !cfg.Logging.File.Compress {
					t.Error("Compress should be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			err := yaml.Unmarshal([]byte(tt.yaml), &cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("yaml.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, &cfg)
			}
		})
	}
}

func TestConfig_FileLoggingFromFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configYAML := `
service:
  name: "ChatbotGate Test"
session:
  cookie_secret: "test-secret-key-at-least-32-characters-long"
oauth2:
  providers:
    - name: "google"
      client_id: "test-client-id"
      client_secret: "test-client-secret"
logging:
  level: "debug"
  color: true
  file:
    path: "/tmp/test-chatbotgate.log"
    max_size_mb: 1
    max_backups: 2
    max_age: 7
    compress: false
proxy:
  upstream:
    url: "http://localhost:8080"
`

	// Write config file
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Read and parse config
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify logging config
	if cfg.Logging.Level != "debug" {
		t.Errorf("Level = %s, want debug", cfg.Logging.Level)
	}
	if !cfg.Logging.Color {
		t.Error("Color should be true")
	}

	// Verify file logging config
	if cfg.Logging.File == nil {
		t.Fatal("File config should not be nil")
	}
	if cfg.Logging.File.Path != "/tmp/test-chatbotgate.log" {
		t.Errorf("Path = %s, want /tmp/test-chatbotgate.log", cfg.Logging.File.Path)
	}
	if cfg.Logging.File.MaxSizeMB != 1 {
		t.Errorf("MaxSizeMB = %d, want 1", cfg.Logging.File.MaxSizeMB)
	}
	if cfg.Logging.File.MaxBackups != 2 {
		t.Errorf("MaxBackups = %d, want 2", cfg.Logging.File.MaxBackups)
	}
	if cfg.Logging.File.MaxAge != 7 {
		t.Errorf("MaxAge = %d, want 7", cfg.Logging.File.MaxAge)
	}
	if cfg.Logging.File.Compress {
		t.Error("Compress should be false")
	}
}
