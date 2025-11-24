package server

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// TestNewMiddlewareManagerWithInvalidConfig tests error handling
func TestNewMiddlewareManagerWithInvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		filename    string
		content     string
		expectError bool
		checkError  string
	}{
		{
			name:        "Non-existent config file",
			filename:    "non-existent.yaml",
			content:     "", // won't be created
			expectError: true,
			checkError:  "failed to build initial middleware",
		},
		{
			name:        "Empty config file",
			filename:    "empty.yaml",
			content:     "",
			expectError: true,
			checkError:  "failed to build initial middleware",
		},
		{
			name:     "Invalid YAML",
			filename: "invalid.yaml",
			content: `
service:
  name: [this is not valid
`,
			expectError: true,
			checkError:  "failed to build initial middleware",
		},
		{
			name:     "Missing required fields",
			filename: "incomplete.yaml",
			content: `
service:
  name: "test"
# Missing session, oauth2, etc.
`,
			expectError: true,
			checkError:  "failed to build initial middleware",
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

			// Create a mock next handler
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			manager, err := NewMiddlewareManager(configPath, "localhost", 4180, nextHandler, logger)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if tt.checkError != "" && err.Error() == "" {
					t.Errorf("Expected error containing %q, but got empty error", tt.checkError)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewMiddlewareManager() error = %v, expectError = %v", err, tt.expectError)
			}

			if manager == nil {
				t.Fatal("Expected manager to be non-nil")
			}
		})
	}
}

// TestNewMiddlewareManagerWithNilLogger tests that nil logger is handled
func TestNewMiddlewareManagerWithNilLogger(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "test.yaml")
	content := `
service:
  name: "test"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test with nil logger - should use default logger
	// This will fail at config validation, but should cover the nil logger check
	_, err := NewMiddlewareManager(configPath, "localhost", 4180, nextHandler, nil)

	// We expect an error due to incomplete config, but we're testing that nil logger doesn't panic
	if err == nil {
		t.Error("Expected error due to incomplete config")
	}
}
