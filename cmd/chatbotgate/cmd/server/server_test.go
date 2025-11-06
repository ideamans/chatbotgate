package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

func TestResolveServerConfig(t *testing.T) {
	// Create a test logger that doesn't output during tests
	logger := logging.NewSimpleLogger("test", logging.LevelError, false)

	// Create temporary directory for test config files
	tmpDir := t.TempDir()

	// Create test config files
	configWithBoth := filepath.Join(tmpDir, "config-both.yaml")
	configContent := `
server:
  host: "127.0.0.1"
  port: 9999
`
	if err := os.WriteFile(configWithBoth, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	configWithHostOnly := filepath.Join(tmpDir, "config-host-only.yaml")
	configHostOnlyContent := `
server:
  host: "192.168.1.1"
`
	if err := os.WriteFile(configWithHostOnly, []byte(configHostOnlyContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	configWithPortOnly := filepath.Join(tmpDir, "config-port-only.yaml")
	configPortOnlyContent := `
server:
  port: 8888
`
	if err := os.WriteFile(configWithPortOnly, []byte(configPortOnlyContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	configEmpty := filepath.Join(tmpDir, "config-empty.yaml")
	if err := os.WriteFile(configEmpty, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	nonExistentConfig := filepath.Join(tmpDir, "non-existent.yaml")

	tests := []struct {
		name         string
		config       Config
		expectedHost string
		expectedPort int
		description  string
	}{
		{
			name: "flags override config file (both)",
			config: Config{
				ConfigPath: configWithBoth,
				Host:       "0.0.0.0",
				Port:       7777,
				HostSet:    true,
				PortSet:    true,
			},
			expectedHost: "0.0.0.0",
			expectedPort: 7777,
			description:  "Command-line flags should override config file values",
		},
		{
			name: "use config file when flags not set",
			config: Config{
				ConfigPath: configWithBoth,
				Host:       "0.0.0.0", // default value
				Port:       4180,      // default value
				HostSet:    false,
				PortSet:    false,
			},
			expectedHost: "127.0.0.1",
			expectedPort: 9999,
			description:  "Config file values should be used when flags not set",
		},
		{
			name: "host flag overrides, port from config",
			config: Config{
				ConfigPath: configWithBoth,
				Host:       "10.0.0.1",
				Port:       4180, // default value
				HostSet:    true,
				PortSet:    false,
			},
			expectedHost: "10.0.0.1",
			expectedPort: 9999,
			description:  "Host from flag, port from config file",
		},
		{
			name: "port flag overrides, host from config",
			config: Config{
				ConfigPath: configWithBoth,
				Host:       "0.0.0.0", // default value
				Port:       6666,
				HostSet:    false,
				PortSet:    true,
			},
			expectedHost: "127.0.0.1",
			expectedPort: 6666,
			description:  "Host from config file, port from flag",
		},
		{
			name: "config has only host",
			config: Config{
				ConfigPath: configWithHostOnly,
				Host:       "0.0.0.0",
				Port:       4180,
				HostSet:    false,
				PortSet:    false,
			},
			expectedHost: "192.168.1.1",
			expectedPort: 4180, // default value
			description:  "Use host from config, default port",
		},
		{
			name: "config has only port",
			config: Config{
				ConfigPath: configWithPortOnly,
				Host:       "0.0.0.0",
				Port:       4180,
				HostSet:    false,
				PortSet:    false,
			},
			expectedHost: "0.0.0.0", // default value
			expectedPort: 8888,
			description:  "Use default host, port from config",
		},
		{
			name: "empty config file",
			config: Config{
				ConfigPath: configEmpty,
				Host:       "0.0.0.0",
				Port:       4180,
				HostSet:    false,
				PortSet:    false,
			},
			expectedHost: "0.0.0.0",
			expectedPort: 4180,
			description:  "Use default values with empty config",
		},
		{
			name: "non-existent config file",
			config: Config{
				ConfigPath: nonExistentConfig,
				Host:       "0.0.0.0",
				Port:       4180,
				HostSet:    false,
				PortSet:    false,
			},
			expectedHost: "0.0.0.0",
			expectedPort: 4180,
			description:  "Use default values when config file doesn't exist",
		},
		{
			name: "flags set to defaults should still override config",
			config: Config{
				ConfigPath: configWithBoth,
				Host:       "0.0.0.0", // default, but explicitly set
				Port:       4180,      // default, but explicitly set
				HostSet:    true,
				PortSet:    true,
			},
			expectedHost: "0.0.0.0",
			expectedPort: 4180,
			description:  "Explicitly set flags (even if default values) should override config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := resolveServerConfig(tt.config, logger)
			if err != nil {
				t.Fatalf("resolveServerConfig() error = %v", err)
			}

			if resolved.Host != tt.expectedHost {
				t.Errorf("Host = %v, want %v (%s)", resolved.Host, tt.expectedHost, tt.description)
			}

			if resolved.Port != tt.expectedPort {
				t.Errorf("Port = %v, want %v (%s)", resolved.Port, tt.expectedPort, tt.description)
			}
		})
	}
}

func TestLoadServerConfig(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		filename     string
		content      string
		expectedHost string
		expectedPort int
		expectError  bool
	}{
		{
			name:     "valid YAML with both values",
			filename: "valid.yaml",
			content: `
server:
  host: "localhost"
  port: 3000
`,
			expectedHost: "localhost",
			expectedPort: 3000,
			expectError:  false,
		},
		{
			name:     "valid JSON with both values",
			filename: "valid.json",
			content: `{
  "server": {
    "host": "127.0.0.1",
    "port": 8080
  }
}`,
			expectedHost: "127.0.0.1",
			expectedPort: 8080,
			expectError:  false,
		},
		{
			name:     "YAML with only host",
			filename: "host-only.yaml",
			content: `
server:
  host: "example.com"
`,
			expectedHost: "example.com",
			expectedPort: 0,
			expectError:  false,
		},
		{
			name:     "YAML with only port",
			filename: "port-only.yaml",
			content: `
server:
  port: 5000
`,
			expectedHost: "",
			expectedPort: 5000,
			expectError:  false,
		},
		{
			name:         "empty YAML",
			filename:     "empty.yaml",
			content:      "",
			expectedHost: "",
			expectedPort: 0,
			expectError:  false,
		},
		{
			name:     "YAML without server section",
			filename: "no-server.yaml",
			content: `
service:
  name: "test"
`,
			expectedHost: "",
			expectedPort: 0,
			expectError:  false,
		},
		{
			name:         "non-existent file",
			filename:     "non-existent.yaml",
			content:      "", // won't be created
			expectedHost: "",
			expectedPort: 0,
			expectError:  false, // loadServerConfig returns empty config for non-existent files
		},
		{
			name:     "invalid YAML",
			filename: "invalid.yaml",
			content: `
server:
  host: [this is not valid
`,
			expectedHost: "",
			expectedPort: 0,
			expectError:  true,
		},
		{
			name:         "unsupported format",
			filename:     "unsupported.txt",
			content:      "host: localhost",
			expectedHost: "",
			expectedPort: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string
			if tt.name != "non-existent file" {
				configPath = filepath.Join(tmpDir, tt.filename)
				if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("Failed to create test config: %v", err)
				}
			} else {
				configPath = filepath.Join(tmpDir, tt.filename)
			}

			cfg, err := loadServerConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("loadServerConfig() error = %v, expectError = %v", err, tt.expectError)
			}

			if cfg.Host != tt.expectedHost {
				t.Errorf("Host = %v, want %v", cfg.Host, tt.expectedHost)
			}

			if cfg.Port != tt.expectedPort {
				t.Errorf("Port = %v, want %v", cfg.Port, tt.expectedPort)
			}
		})
	}
}
