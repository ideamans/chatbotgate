package email

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
)

// TestSendGridSender_HTTPErrors tests SendGrid API error responses
func TestSendGridSender_HTTPErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectError    bool
		errorSubstring string
	}{
		{
			name:           "400 Bad Request",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"errors":[{"message":"Invalid email address"}]}`,
			expectError:    true,
			errorSubstring: "400",
		},
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"errors":[{"message":"Invalid API key"}]}`,
			expectError:    true,
			errorSubstring: "401",
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			responseBody:   `{"errors":[{"message":"Access denied"}]}`,
			expectError:    true,
			errorSubstring: "403",
		},
		{
			name:           "429 Rate Limited",
			statusCode:     http.StatusTooManyRequests,
			responseBody:   `{"errors":[{"message":"Rate limit exceeded"}]}`,
			expectError:    true,
			errorSubstring: "429",
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"errors":[{"message":"Internal server error"}]}`,
			expectError:    true,
			errorSubstring: "500",
		},
		{
			name:           "503 Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			responseBody:   `{"errors":[{"message":"Service unavailable"}]}`,
			expectError:    true,
			errorSubstring: "503",
		},
		{
			name:         "202 Accepted (Success)",
			statusCode:   http.StatusAccepted,
			responseBody: `{}`,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock SendGrid API server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request headers
				if r.Header.Get("Authorization") == "" {
					t.Error("Authorization header not set")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create SendGrid sender with mock endpoint
			cfg := config.SendGridConfig{
				APIKey:      "test-api-key",
				EndpointURL: server.URL,
				From:        "test@example.com",
				FromName:    "Test Sender",
			}

			sender := NewSendGridSender(cfg, "", "")

			// Send email
			err := sender.Send("recipient@example.com", "Test Subject", "Test Body")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for status %d, got nil", tt.statusCode)
				}
				if tt.errorSubstring != "" && !contains(err.Error(), tt.errorSubstring) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorSubstring, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for status %d: %v", tt.statusCode, err)
				}
			}
		})
	}
}

// TestSendGridSender_SendHTMLErrors tests SendHTML error responses
func TestSendGridSender_SendHTMLErrors(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name:         "400 Bad Request",
			statusCode:   http.StatusBadRequest,
			responseBody: `{"errors":[{"message":"Invalid HTML content"}]}`,
			expectError:  true,
		},
		{
			name:         "500 Internal Server Error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"errors":[{"message":"Server error"}]}`,
			expectError:  true,
		},
		{
			name:         "200 OK",
			statusCode:   http.StatusOK,
			responseBody: `{}`,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			cfg := config.SendGridConfig{
				APIKey:      "test-api-key",
				EndpointURL: server.URL,
				From:        "test@example.com",
				FromName:    "Test Sender",
			}

			sender := NewSendGridSender(cfg, "", "")

			err := sender.SendHTML(
				"recipient@example.com",
				"Test Subject",
				"<html><body>HTML Body</body></html>",
				"Text Body",
			)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for status %d, got nil", tt.statusCode)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for status %d: %v", tt.statusCode, err)
			}
		})
	}
}

// TestSendGridSender_NetworkErrors tests network-level errors
func TestSendGridSender_NetworkErrors(t *testing.T) {
	// Create sender with invalid endpoint
	cfg := config.SendGridConfig{
		APIKey:      "test-api-key",
		EndpointURL: "http://invalid-host-that-does-not-exist.example.com:9999",
		From:        "test@example.com",
		FromName:    "Test Sender",
	}

	sender := NewSendGridSender(cfg, "", "")

	// Try to send email - should fail with network error
	err := sender.Send("recipient@example.com", "Test Subject", "Test Body")

	if err == nil {
		t.Error("Expected network error, got nil")
	}
}

// TestSendGridSender_CustomEndpointCalled tests that custom endpoint is actually called
func TestSendGridSender_CustomEndpointCalled(t *testing.T) {
	customEndpointCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customEndpointCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	cfg := config.SendGridConfig{
		APIKey:      "test-api-key",
		EndpointURL: server.URL,
		From:        "test@example.com",
		FromName:    "Test Sender",
	}

	sender := NewSendGridSender(cfg, "", "")
	err := sender.Send("recipient@example.com", "Test", "Body")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !customEndpointCalled {
		t.Error("Custom endpoint was not called")
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
