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
	provider := NewMicrosoftProvider("test-client-id", "test-client-secret", "http://localhost/callback", nil, false)

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
				"mail":               "user@example.com",
				"userPrincipalName":  "user@tenant.onmicrosoft.com",
				"preferredUsername":  "user@example.com",
			},
			statusCode: http.StatusOK,
			wantEmail:  "user@example.com",
			wantErr:    false,
		},
		{
			name: "mail empty - use userPrincipalName",
			response: map[string]interface{}{
				"mail":               "",
				"userPrincipalName":  "user@tenant.onmicrosoft.com",
				"preferredUsername":  "user@example.com",
			},
			statusCode: http.StatusOK,
			wantEmail:  "user@tenant.onmicrosoft.com",
			wantErr:    false,
		},
		{
			name: "mail and userPrincipalName empty - use preferredUsername",
			response: map[string]interface{}{
				"mail":               "",
				"userPrincipalName":  "",
				"preferredUsername":  "user@example.com",
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
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			provider := NewMicrosoftProvider("test-client-id", "test-client-secret", "http://localhost/callback", nil, false)

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
