package oauth2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	oauth2lib "golang.org/x/oauth2"
)

func TestNewMicrosoftProvider(t *testing.T) {
	provider := NewMicrosoftProvider("microsoft", "test-client-id", "test-client-secret", "http://localhost/callback", nil, false)

	if provider == nil {
		t.Fatal("NewMicrosoftProvider() returned nil")
	}

	if provider.Name() != "microsoft" {
		t.Errorf("Name() = %s, want microsoft", provider.Name())
	}

	config := provider.Config()
	if config.ClientID != "test-client-id" {
		t.Errorf("ClientID = %s, want test-client-id", config.ClientID)
	}

	if config.ClientSecret != "test-client-secret" {
		t.Errorf("ClientSecret = %s, want test-client-secret", config.ClientSecret)
	}

	expectedScopes := []string{"openid", "profile", "email", "User.Read"}
	if len(config.Scopes) != len(expectedScopes) {
		t.Errorf("Scopes count = %d, want %d", len(config.Scopes), len(expectedScopes))
	}

	for i, scope := range expectedScopes {
		if i >= len(config.Scopes) || config.Scopes[i] != scope {
			t.Errorf("Scopes = %v, want %v", config.Scopes, expectedScopes)
			break
		}
	}
}

func TestMicrosoftProvider_CustomScopes(t *testing.T) {
	// Test with custom scopes - should use only custom scopes (no defaults added)
	customScopes := []string{"Calendars.Read", "Mail.Read"}
	provider := NewMicrosoftProvider("microsoft", "test-client-id", "test-client-secret", "http://localhost/callback", customScopes, false)

	config := provider.Config()

	// Should have only custom scopes (defaults not added)
	expectedScopes := []string{
		"Calendars.Read",
		"Mail.Read",
	}

	if len(config.Scopes) != len(expectedScopes) {
		t.Errorf("Scopes length = %d, want %d", len(config.Scopes), len(expectedScopes))
	}

	for i, scope := range expectedScopes {
		if i >= len(config.Scopes) || config.Scopes[i] != scope {
			t.Errorf("Scopes[%d] = %s, want %s", i, config.Scopes[i], scope)
		}
	}
}

func TestMicrosoftProvider_CustomScopesWithResetFlag(t *testing.T) {
	// Test with custom scopes and reset_scopes: true
	// Behavior is same as reset_scopes: false (only custom scopes are used)
	customScopes := []string{"Calendars.Read", "Mail.Read"}
	provider := NewMicrosoftProvider("microsoft", "test-client-id", "test-client-secret", "http://localhost/callback", customScopes, true)

	config := provider.Config()

	// Should have only custom scopes
	expectedScopes := []string{
		"Calendars.Read",
		"Mail.Read",
	}

	if len(config.Scopes) != len(expectedScopes) {
		t.Errorf("Scopes length = %d, want %d", len(config.Scopes), len(expectedScopes))
	}

	for i, scope := range expectedScopes {
		if i >= len(config.Scopes) || config.Scopes[i] != scope {
			t.Errorf("Scopes[%d] = %s, want %s", i, config.Scopes[i], scope)
		}
	}
}

func TestMicrosoftProvider_EmptyScopes(t *testing.T) {
	// Test with empty scopes - should use default scopes
	provider := NewMicrosoftProvider("microsoft", "test-client-id", "test-client-secret", "http://localhost/callback", nil, true)

	config := provider.Config()

	// Should use default scopes when scopes are empty
	expectedScopes := []string{
		"openid",
		"profile",
		"email",
		"User.Read",
	}

	if len(config.Scopes) != len(expectedScopes) {
		t.Errorf("Scopes length = %d, want %d", len(config.Scopes), len(expectedScopes))
	}

	for i, scope := range expectedScopes {
		if i >= len(config.Scopes) || config.Scopes[i] != scope {
			t.Errorf("Scopes[%d] = %s, want %s", i, config.Scopes[i], scope)
		}
	}
}

func TestMicrosoftProvider_GetUserEmail(t *testing.T) {
	tests := []struct {
		name        string
		response    interface{}
		statusCode  int
		wantEmail   string
		wantErr     bool
		wantErrType error
	}{
		{
			name: "mail field present",
			response: map[string]interface{}{
				"mail":              "user@example.com",
				"userPrincipalName": "user@tenant.onmicrosoft.com",
				"preferredUsername": "user@example.com",
			},
			statusCode: http.StatusOK,
			wantEmail:  "user@example.com",
			wantErr:    false,
		},
		{
			name: "mail empty - use userPrincipalName",
			response: map[string]interface{}{
				"mail":              "",
				"userPrincipalName": "user@tenant.onmicrosoft.com",
				"preferredUsername": "user@example.com",
			},
			statusCode: http.StatusOK,
			wantEmail:  "user@tenant.onmicrosoft.com",
			wantErr:    false,
		},
		{
			name: "mail and userPrincipalName empty - use preferredUsername",
			response: map[string]interface{}{
				"mail":              "",
				"userPrincipalName": "",
				"preferredUsername": "user@example.com",
			},
			statusCode: http.StatusOK,
			wantEmail:  "user@example.com",
			wantErr:    false,
		},
		{
			name: "no email fields",
			response: map[string]interface{}{
				"displayName": "Test User",
				"id":          "12345",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					_ = json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			provider := NewMicrosoftProvider("microsoft", "test-client-id", "test-client-secret", "http://localhost/callback", nil, false)

			// Create test token
			token := &oauth2lib.Token{
				AccessToken: "test-token",
			}

			// Create context with custom HTTP client that uses test server
			ctx := context.Background()
			ctx = context.WithValue(ctx, oauth2lib.HTTPClient, &http.Client{
				Transport: &testTransport{
					baseURL: server.URL,
					path:    "/me",
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
