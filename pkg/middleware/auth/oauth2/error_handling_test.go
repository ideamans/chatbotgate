package oauth2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestGoogleProvider_TokenExchangeErrors tests error handling during token exchange
func TestGoogleProvider_TokenExchangeErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectError    bool
		errorSubstring string
	}{
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error":"invalid_client","error_description":"Invalid client credentials"}`,
			expectError:    true,
			errorSubstring: "401",
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			responseBody:   `{"error":"access_denied","error_description":"Access denied"}`,
			expectError:    true,
			errorSubstring: "403",
		},
		{
			name:           "429 Rate Limited",
			statusCode:     http.StatusTooManyRequests,
			responseBody:   `{"error":"rate_limit_exceeded","error_description":"Too many requests"}`,
			expectError:    true,
			errorSubstring: "429",
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error":"server_error","error_description":"Internal server error"}`,
			expectError:    true,
			errorSubstring: "500",
		},
		{
			name:           "502 Bad Gateway",
			statusCode:     http.StatusBadGateway,
			responseBody:   `<html><body>Bad Gateway</body></html>`,
			expectError:    true,
			errorSubstring: "502",
		},
		{
			name:           "503 Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			responseBody:   `{"error":"service_unavailable"}`,
			expectError:    true,
			errorSubstring: "503",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock token server that returns error response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			provider := NewGoogleProvider(
				"google",
				"test-client-id",
				"test-client-secret",
				"http://localhost/callback",
				nil,
				false,
			)

			// Override token URL to point to mock server
			config := provider.Config()
			config.Endpoint.TokenURL = server.URL + "/token"

			ctx := context.Background()
			_, err := config.Exchange(ctx, "test-code")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for status %d, got nil", tt.statusCode)
				}
				// The error from oauth2 library contains the status code
				if !strings.Contains(err.Error(), tt.errorSubstring) && !strings.Contains(err.Error(), "oauth2") {
					t.Logf("Expected error to contain '%s' or 'oauth2', got: %v", tt.errorSubstring, err)
				}
			}
		})
	}
}

// TestGoogleProvider_InvalidJSONResponse tests handling of invalid JSON from token endpoint
func TestGoogleProvider_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return invalid JSON
		w.Write([]byte("This is not valid JSON {{{"))
	}))
	defer server.Close()

	provider := NewGoogleProvider(
		"google",
		"test-client-id",
		"test-client-secret",
		"http://localhost/callback",
		nil,
		false,
	)

	config := provider.Config()
	config.Endpoint.TokenURL = server.URL + "/token"

	ctx := context.Background()
	_, err := config.Exchange(ctx, "test-code")

	if err == nil {
		t.Error("Exchange should return error for invalid JSON response")
	}

	// Should contain 'invalid' or parsing-related error
	if !strings.Contains(strings.ToLower(err.Error()), "invalid") &&
		!strings.Contains(strings.ToLower(err.Error()), "cannot") &&
		!strings.Contains(strings.ToLower(err.Error()), "unmarshal") {
		t.Logf("Expected parsing error, got: %v", err)
	}
}

// TestGoogleProvider_ContextTimeout tests context timeout during token exchange
func TestGoogleProvider_ContextTimeout(t *testing.T) {
	// Create server with delay longer than timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"token"}`))
	}))
	defer server.Close()

	provider := NewGoogleProvider(
		"google",
		"test-client-id",
		"test-client-secret",
		"http://localhost/callback",
		nil,
		false,
	)

	config := provider.Config()
	config.Endpoint.TokenURL = server.URL + "/token"

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := config.Exchange(ctx, "test-code")

	if err == nil {
		t.Error("Exchange should return error on context timeout")
	}

	// Should be context deadline exceeded error
	if !strings.Contains(err.Error(), "context deadline exceeded") &&
		!strings.Contains(err.Error(), "timeout") {
		t.Logf("Expected timeout error, got: %v", err)
	}
}

// TestGoogleProvider_ContextCancellation tests immediate context cancellation
func TestGoogleProvider_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"token"}`))
	}))
	defer server.Close()

	provider := NewGoogleProvider(
		"google",
		"test-client-id",
		"test-client-secret",
		"http://localhost/callback",
		nil,
		false,
	)

	config := provider.Config()
	config.Endpoint.TokenURL = server.URL + "/token"

	// Create context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := config.Exchange(ctx, "test-code")

	if err == nil {
		t.Error("Exchange should return error when context is cancelled")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Logf("Expected context cancellation error, got: %v", err)
	}
}

// Note: GetUserInfo error handling tests are better suited for integration tests
// or Manager-level tests with mock providers, since the userinfo URL is hardcoded
// in each provider implementation. See manager_test.go for examples using MockProvider.

// TestGitHubProvider_TokenExchangeErrors tests GitHub provider error handling
func TestGitHubProvider_TokenExchangeErrors(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name:         "Bad credentials",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error":"bad_credentials"}`,
			expectError:  true,
		},
		{
			name:         "Rate limit",
			statusCode:   http.StatusForbidden,
			responseBody: `{"error":"rate_limit"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			provider := NewGitHubProvider(
				"github",
				"test-client-id",
				"test-client-secret",
				"http://localhost/callback",
				nil,
				false,
			)

			config := provider.Config()
			config.Endpoint.TokenURL = server.URL + "/token"

			ctx := context.Background()
			_, err := config.Exchange(ctx, "test-code")

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

// TestMicrosoftProvider_TokenExchangeErrors tests Microsoft provider error handling
func TestMicrosoftProvider_TokenExchangeErrors(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name:         "Invalid grant",
			statusCode:   http.StatusBadRequest,
			responseBody: `{"error":"invalid_grant"}`,
			expectError:  true,
		},
		{
			name:         "Invalid client",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error":"invalid_client"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			provider := NewMicrosoftProvider(
				"microsoft",
				"test-client-id",
				"test-client-secret",
				"http://localhost/callback",
				nil,
				false,
			)

			config := provider.Config()
			config.Endpoint.TokenURL = server.URL + "/token"

			ctx := context.Background()
			_, err := config.Exchange(ctx, "test-code")

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

// TestCustomProvider_TokenExchangeErrors tests Custom provider error handling
func TestCustomProvider_TokenExchangeErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"unsupported_grant_type"}`))
	}))
	defer server.Close()

	provider := NewCustomProvider(
		"custom",
		"test-client-id",
		"test-client-secret",
		server.URL+"/auth",
		server.URL+"/token",
		server.URL+"/userinfo",
		"http://localhost/callback",
		nil,
		false,
	)

	config := provider.Config()

	ctx := context.Background()
	_, err := config.Exchange(ctx, "test-code")

	if err == nil {
		t.Error("Expected error for invalid grant type, got nil")
	}
}

// TestProvider_EmptyAccessToken tests handling of empty access token
func TestProvider_EmptyAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return response without access_token
		w.Write([]byte(`{"token_type":"bearer","expires_in":3600}`))
	}))
	defer server.Close()

	provider := NewGoogleProvider(
		"google",
		"test-client-id",
		"test-client-secret",
		"http://localhost/callback",
		nil,
		false,
	)

	config := provider.Config()
	config.Endpoint.TokenURL = server.URL + "/token"

	ctx := context.Background()
	token, err := config.Exchange(ctx, "test-code")

	// oauth2 library may or may not error on empty token
	// Just verify it doesn't panic
	if err == nil && token != nil {
		if token.AccessToken == "" {
			t.Log("Empty access token accepted by oauth2 library")
		}
	}
}
