package oauth2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	oauth2lib "golang.org/x/oauth2"
)

func TestNewCustomProvider(t *testing.T) {
	provider := NewCustomProvider(
		"custom-provider",
		"test-client-id",
		"test-client-secret",
		"http://localhost/callback",
		"https://auth.example.com/oauth/authorize",
		"https://auth.example.com/oauth/token",
		"https://auth.example.com/oauth/userinfo",
		nil, // Use default scopes
		false,
	)

	if provider == nil {
		t.Fatal("NewCustomProvider() returned nil")
	}

	if provider.Name() != "custom-provider" {
		t.Errorf("Name() = %s, want custom-provider", provider.Name())
	}

	config := provider.Config()
	if config.ClientID != "test-client-id" {
		t.Errorf("ClientID = %s, want test-client-id", config.ClientID)
	}

	if config.ClientSecret != "test-client-secret" {
		t.Errorf("ClientSecret = %s, want test-client-secret", config.ClientSecret)
	}

	if config.Endpoint.AuthURL != "https://auth.example.com/oauth/authorize" {
		t.Errorf("AuthURL = %s, want https://auth.example.com/oauth/authorize", config.Endpoint.AuthURL)
	}

	if config.Endpoint.TokenURL != "https://auth.example.com/oauth/token" {
		t.Errorf("TokenURL = %s, want https://auth.example.com/oauth/token", config.Endpoint.TokenURL)
	}

	expectedScopes := []string{"openid", "email", "profile"}
	if len(config.Scopes) != len(expectedScopes) {
		t.Errorf("Scopes length = %d, want %d", len(config.Scopes), len(expectedScopes))
	}
	for i, scope := range expectedScopes {
		if config.Scopes[i] != scope {
			t.Errorf("Scopes[%d] = %s, want %s", i, config.Scopes[i], scope)
		}
	}
}

func TestCustomProvider_GetUserEmail(t *testing.T) {
	tests := []struct {
		name        string
		response    interface{}
		statusCode  int
		wantEmail   string
		wantErr     bool
		wantErrType error
	}{
		{
			name: "valid email",
			response: map[string]interface{}{
				"email":          "user@example.com",
				"email_verified": true,
			},
			statusCode: http.StatusOK,
			wantEmail:  "user@example.com",
			wantErr:    false,
		},
		{
			name: "email without verification field",
			response: map[string]interface{}{
				"email": "user@example.com",
			},
			statusCode: http.StatusOK,
			wantEmail:  "user@example.com",
			wantErr:    false,
		},
		{
			name: "missing email",
			response: map[string]interface{}{
				"email_verified": true,
			},
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
		{
			name: "empty email",
			response: map[string]interface{}{
				"email":          "",
				"email_verified": true,
			},
			statusCode:  http.StatusOK,
			wantEmail:   "",
			wantErr:     true,
			wantErrType: ErrEmailNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					if err := json.NewEncoder(w).Encode(tt.response); err != nil {
						t.Fatalf("Failed to encode response: %v", err)
					}
				}
			}))
			defer server.Close()

			provider := NewCustomProvider(
				"custom-provider",
				"test-client-id",
				"test-client-secret",
				"http://localhost/callback",
				"https://auth.example.com/oauth/authorize",
				"https://auth.example.com/oauth/token",
				server.URL+"/userinfo",
				nil, // Use default scopes
				false,
			)

			// Create test token
			token := &oauth2lib.Token{
				AccessToken: "test-token",
			}

			ctx := context.Background()
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

func TestCustomProvider_GetUserEmail_InsecureSkipVerify(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"email":          "user@example.com",
			"email_verified": true,
		})
	}))
	defer server.Close()

	// Test with insecure skip verify enabled
	provider := NewCustomProvider(
		"custom-provider",
		"test-client-id",
		"test-client-secret",
		"http://localhost/callback",
		"http://auth.example.com/oauth/authorize",
		"http://auth.example.com/oauth/token",
		server.URL+"/userinfo",
		nil,  // Use default scopes
		true, // insecureSkipVerify enabled
	)

	token := &oauth2lib.Token{
		AccessToken: "test-token",
	}

	ctx := context.Background()
	email, err := provider.GetUserEmail(ctx, token)

	if err != nil {
		t.Errorf("GetUserEmail() with insecureSkipVerify unexpected error = %v", err)
	}

	if email != "user@example.com" {
		t.Errorf("GetUserEmail() = %s, want user@example.com", email)
	}
}
