package oauth2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	oauth2lib "golang.org/x/oauth2"
)

func TestNewGitHubProvider(t *testing.T) {
	provider := NewGitHubProvider("test-client-id", "test-client-secret", "http://localhost/callback")

	if provider == nil {
		t.Fatal("NewGitHubProvider() returned nil")
	}

	if provider.Name() != "github" {
		t.Errorf("Name() = %s, want github", provider.Name())
	}

	config := provider.Config()
	if config.ClientID != "test-client-id" {
		t.Errorf("ClientID = %s, want test-client-id", config.ClientID)
	}

	if config.ClientSecret != "test-client-secret" {
		t.Errorf("ClientSecret = %s, want test-client-secret", config.ClientSecret)
	}

	expectedScopes := []string{"user:email", "read:user"}
	if len(config.Scopes) != len(expectedScopes) {
		t.Errorf("Scopes length = %d, want %d", len(config.Scopes), len(expectedScopes))
	}
	for i, scope := range expectedScopes {
		if i >= len(config.Scopes) || config.Scopes[i] != scope {
			t.Errorf("Scopes = %v, want %v", config.Scopes, expectedScopes)
			break
		}
	}
}

func TestGitHubProvider_GetUserEmail(t *testing.T) {
	tests := []struct {
		name        string
		response    interface{}
		statusCode  int
		wantEmail   string
		wantErr     bool
		wantErrType error
	}{
		{
			name: "primary verified email",
			response: []map[string]interface{}{
				{
					"email":      "user@example.com",
					"primary":    true,
					"verified":   true,
					"visibility": "public",
				},
			},
			statusCode: http.StatusOK,
			wantEmail:  "user@example.com",
			wantErr:    false,
		},
		{
			name: "multiple emails - use primary",
			response: []map[string]interface{}{
				{
					"email":      "secondary@example.com",
					"primary":    false,
					"verified":   true,
					"visibility": "public",
				},
				{
					"email":      "primary@example.com",
					"primary":    true,
					"verified":   true,
					"visibility": "public",
				},
			},
			statusCode: http.StatusOK,
			wantEmail:  "primary@example.com",
			wantErr:    false,
		},
		{
			name: "no primary - use first verified",
			response: []map[string]interface{}{
				{
					"email":      "first@example.com",
					"primary":    false,
					"verified":   true,
					"visibility": "public",
				},
				{
					"email":      "second@example.com",
					"primary":    false,
					"verified":   true,
					"visibility": "public",
				},
			},
			statusCode: http.StatusOK,
			wantEmail:  "first@example.com",
			wantErr:    false,
		},
		{
			name: "no verified email",
			response: []map[string]interface{}{
				{
					"email":      "unverified@example.com",
					"primary":    true,
					"verified":   false,
					"visibility": "public",
				},
			},
			statusCode:  http.StatusOK,
			wantEmail:   "",
			wantErr:     true,
			wantErrType: ErrEmailNotFound,
		},
		{
			name:        "empty response",
			response:    []map[string]interface{}{},
			statusCode:  http.StatusOK,
			wantEmail:   "",
			wantErr:     true,
			wantErrType: ErrEmailNotFound,
		},
		{
			name:       "api error",
			response:   nil,
			statusCode: http.StatusUnauthorized,
			wantEmail:  "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			provider := NewGitHubProvider("test-client-id", "test-client-secret", "http://localhost/callback")

			// Create test token
			token := &oauth2lib.Token{
				AccessToken: "test-token",
			}

			// Create context with custom HTTP client that uses test server
			ctx := context.Background()
			ctx = context.WithValue(ctx, oauth2lib.HTTPClient, &http.Client{
				Transport: &testTransport{
					baseURL: server.URL,
					path:    "/user/emails",
				},
			})

			email, err := provider.GetUserEmail(ctx, token)

			if tt.wantErr {
				if err == nil {
					t.Error("GetUserEmail() expected error, got nil")
				}
				if tt.wantErrType != nil && err != tt.wantErrType {
					t.Errorf("GetUserEmail() error = %v, want %v", err, tt.wantErrType)
				}
				return
			}

			if err != nil {
				t.Errorf("GetUserEmail() unexpected error = %v", err)
			}

			if email != tt.wantEmail {
				t.Errorf("GetUserEmail() = %s, want %s", email, tt.wantEmail)
			}
		})
	}
}
