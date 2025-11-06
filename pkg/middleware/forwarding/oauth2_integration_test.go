package forwarding

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
)

// TestOAuth2GoogleIntegration tests the complete flow from Google OAuth2 to forwarding
func TestOAuth2GoogleIntegration(t *testing.T) {
	// Simulate UserInfo returned by Google OAuth2 provider
	googleUserInfo := &UserInfo{
		Email:    "user@example.com",
		Username: "John Doe",
		Provider: "google",
		Extra: map[string]any{
			// Standardized fields (populated by Google provider)
			"_email":      "user@example.com",
			"_username":   "John Doe",
			"_avatar_url": "https://lh3.googleusercontent.com/a/default-user",
			// Google-specific fields
			"email":          "user@example.com",
			"name":           "John Doe",
			"picture":        "https://lh3.googleusercontent.com/a/default-user",
			"verified_email": true,
			"given_name":     "John",
			"family_name":    "Doe",
			// OAuth2 tokens
			"secrets": map[string]any{
				"access_token":  "ya29.a0AfH6SMBx...",
				"refresh_token": "1//0gHZjHlQYr...",
			},
		},
	}

	tests := []struct {
		name            string
		config          *config.ForwardingConfig
		testQuery       bool
		testHeaders     bool
		expectedQuery   map[string]string
		expectedHeaders map[string]string
		shouldContain   map[string]bool // For checking if value exists
	}{
		{
			name: "Standardized fields to query",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "_email", Query: "email"},
					{Path: "_username", Query: "username"},
					{Path: "_avatar_url", Query: "avatar"},
				},
			},
			testQuery: true,
			expectedQuery: map[string]string{
				"email":    "user@example.com",
				"username": "John Doe",
				"avatar":   "https://lh3.googleusercontent.com/a/default-user",
			},
		},
		{
			name: "Standardized fields to headers",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "_email", Header: "X-User-Email"},
					{Path: "_username", Header: "X-User-Name"},
					{Path: "_avatar_url", Header: "X-User-Avatar"},
				},
			},
			testHeaders: true,
			expectedHeaders: map[string]string{
				"X-User-Email":  "user@example.com",
				"X-User-Name":   "John Doe",
				"X-User-Avatar": "https://lh3.googleusercontent.com/a/default-user",
			},
		},
		{
			name: "Google-specific fields",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "email", Query: "google_email"},
					{Path: "given_name", Query: "first_name"},
					{Path: "family_name", Query: "last_name"},
					{Path: "verified_email", Query: "verified"},
				},
			},
			testQuery: true,
			expectedQuery: map[string]string{
				"google_email": "user@example.com",
				"first_name":   "John",
				"last_name":    "Doe",
				"verified":     "true",
			},
		},
		{
			name: "Google OAuth2 access token",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "secrets.access_token", Header: "X-OAuth-Token"},
					{Path: "secrets.refresh_token", Header: "X-Refresh-Token"},
				},
			},
			testHeaders: true,
			expectedHeaders: map[string]string{
				"X-OAuth-Token":   "ya29.a0AfH6SMBx...",
				"X-Refresh-Token": "1//0gHZjHlQYr...",
			},
		},
		{
			name: "Mixed query and headers",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "_email", Query: "email", Header: "X-Email"},
					{Path: "provider", Query: "provider", Header: "X-Provider"},
					{Path: "secrets.access_token", Header: "X-Token"},
				},
			},
			testQuery:   true,
			testHeaders: true,
			expectedQuery: map[string]string{
				"email":    "user@example.com",
				"provider": "google",
			},
			expectedHeaders: map[string]string{
				"X-Email":    "user@example.com",
				"X-Provider": "google",
				"X-Token":    "ya29.a0AfH6SMBx...",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forwarder := NewForwarder(tt.config, nil)

			// Test query string
			if tt.testQuery {
				resultURL, err := forwarder.AddToQueryString("http://example.com/path", googleUserInfo)
				if err != nil {
					t.Fatalf("AddToQueryString() error = %v", err)
				}

				u, _ := url.Parse(resultURL)
				for key, expected := range tt.expectedQuery {
					actual := u.Query().Get(key)
					if actual != expected {
						t.Errorf("Query param %s = %v, want %v", key, actual, expected)
					}
				}
			}

			// Test headers
			if tt.testHeaders {
				headers := make(http.Header)
				resultHeaders := forwarder.AddToHeaders(headers, googleUserInfo)

				for key, expected := range tt.expectedHeaders {
					actual := resultHeaders.Get(key)
					if actual != expected {
						t.Errorf("Header %s = %v, want %v", key, actual, expected)
					}
				}
			}
		})
	}
}

// TestOAuth2GitHubIntegration tests the complete flow from GitHub OAuth2 to forwarding
func TestOAuth2GitHubIntegration(t *testing.T) {
	// Simulate UserInfo returned by GitHub OAuth2 provider
	githubUserInfo := &UserInfo{
		Email:    "user@example.com",
		Username: "John Smith", // Display name
		Provider: "github",
		Extra: map[string]any{
			// Standardized fields (populated by GitHub provider)
			"_email":      "user@example.com",
			"_username":   "John Smith", // name field (or login as fallback)
			"_avatar_url": "https://avatars.githubusercontent.com/u/12345?v=4",
			// GitHub-specific fields
			"email":      "user@example.com",
			"name":       "John Smith",
			"login":      "johnsmith",
			"avatar_url": "https://avatars.githubusercontent.com/u/12345?v=4",
			"bio":        "Software Developer",
			"company":    "Example Corp",
			// OAuth2 tokens
			"secrets": map[string]any{
				"access_token": "gho_16C7e42F292c6912E7710c838347Ae178B4a",
			},
		},
	}

	tests := []struct {
		name            string
		config          *config.ForwardingConfig
		testQuery       bool
		testHeaders     bool
		expectedQuery   map[string]string
		expectedHeaders map[string]string
	}{
		{
			name: "Standardized fields to query",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "_email", Query: "email"},
					{Path: "_username", Query: "username"},
					{Path: "_avatar_url", Query: "avatar"},
				},
			},
			testQuery: true,
			expectedQuery: map[string]string{
				"email":    "user@example.com",
				"username": "John Smith",
				"avatar":   "https://avatars.githubusercontent.com/u/12345?v=4",
			},
		},
		{
			name: "Standardized fields to headers",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "_email", Header: "X-User-Email"},
					{Path: "_username", Header: "X-User-Name"},
					{Path: "_avatar_url", Header: "X-User-Avatar"},
				},
			},
			testHeaders: true,
			expectedHeaders: map[string]string{
				"X-User-Email":  "user@example.com",
				"X-User-Name":   "John Smith",
				"X-User-Avatar": "https://avatars.githubusercontent.com/u/12345?v=4",
			},
		},
		{
			name: "GitHub-specific fields",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "login", Query: "gh_login"},
					{Path: "name", Query: "gh_name"},
					{Path: "bio", Query: "bio"},
					{Path: "company", Query: "company"},
				},
			},
			testQuery: true,
			expectedQuery: map[string]string{
				"gh_login": "johnsmith",
				"gh_name":  "John Smith",
				"bio":      "Software Developer",
				"company":  "Example Corp",
			},
		},
		{
			name: "GitHub OAuth2 access token",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "secrets.access_token", Header: "X-GitHub-Token"},
				},
			},
			testHeaders: true,
			expectedHeaders: map[string]string{
				"X-GitHub-Token": "gho_16C7e42F292c6912E7710c838347Ae178B4a",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forwarder := NewForwarder(tt.config, nil)

			// Test query string
			if tt.testQuery {
				resultURL, err := forwarder.AddToQueryString("http://example.com/path", githubUserInfo)
				if err != nil {
					t.Fatalf("AddToQueryString() error = %v", err)
				}

				u, _ := url.Parse(resultURL)
				for key, expected := range tt.expectedQuery {
					actual := u.Query().Get(key)
					if actual != expected {
						t.Errorf("Query param %s = %v, want %v", key, actual, expected)
					}
				}
			}

			// Test headers
			if tt.testHeaders {
				headers := make(http.Header)
				resultHeaders := forwarder.AddToHeaders(headers, githubUserInfo)

				for key, expected := range tt.expectedHeaders {
					actual := resultHeaders.Get(key)
					if actual != expected {
						t.Errorf("Header %s = %v, want %v", key, actual, expected)
					}
				}
			}
		})
	}
}

// TestOAuth2MicrosoftIntegration tests the complete flow from Microsoft OAuth2 to forwarding
func TestOAuth2MicrosoftIntegration(t *testing.T) {
	// Simulate UserInfo returned by Microsoft OAuth2 provider
	microsoftUserInfo := &UserInfo{
		Email:    "user@example.com",
		Username: "John Doe",
		Provider: "microsoft",
		Extra: map[string]any{
			// Standardized fields (populated by Microsoft provider)
			"_email":      "user@example.com",
			"_username":   "John Doe",
			"_avatar_url": "", // Microsoft doesn't provide direct avatar URL
			// Microsoft-specific fields
			"email":             "user@example.com",
			"displayName":       "John Doe",
			"userPrincipalName": "user@tenant.onmicrosoft.com",
			"preferredUsername": "user@example.com",
			"id":                "00000000-0000-0000-0000-000000000000",
			"jobTitle":          "Software Engineer",
			"officeLocation":    "Building 1",
			// OAuth2 tokens
			"secrets": map[string]any{
				"access_token":  "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsIng1dCI6...",
				"refresh_token": "M.R3_BAY...",
			},
		},
	}

	tests := []struct {
		name            string
		config          *config.ForwardingConfig
		testQuery       bool
		testHeaders     bool
		expectedQuery   map[string]string
		expectedHeaders map[string]string
	}{
		{
			name: "Standardized fields to query",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "_email", Query: "email"},
					{Path: "_username", Query: "username"},
					{Path: "_avatar_url", Query: "avatar"},
				},
			},
			testQuery: true,
			expectedQuery: map[string]string{
				"email":    "user@example.com",
				"username": "John Doe",
				"avatar":   "", // Empty for Microsoft
			},
		},
		{
			name: "Standardized fields to headers",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "_email", Header: "X-User-Email"},
					{Path: "_username", Header: "X-User-Name"},
					{Path: "_avatar_url", Header: "X-User-Avatar"},
				},
			},
			testHeaders: true,
			expectedHeaders: map[string]string{
				"X-User-Email":  "user@example.com",
				"X-User-Name":   "John Doe",
				"X-User-Avatar": "", // Empty for Microsoft
			},
		},
		{
			name: "Microsoft-specific fields",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "displayName", Query: "display_name"},
					{Path: "userPrincipalName", Query: "upn"},
					{Path: "jobTitle", Query: "title"},
					{Path: "officeLocation", Query: "office"},
				},
			},
			testQuery: true,
			expectedQuery: map[string]string{
				"display_name": "John Doe",
				"upn":          "user@tenant.onmicrosoft.com",
				"title":        "Software Engineer",
				"office":       "Building 1",
			},
		},
		{
			name: "Microsoft OAuth2 access token",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "secrets.access_token", Header: "X-MS-Token"},
					{Path: "secrets.refresh_token", Header: "X-Refresh-Token"},
				},
			},
			testHeaders: true,
			expectedHeaders: map[string]string{
				"X-MS-Token":      "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsIng1dCI6...",
				"X-Refresh-Token": "M.R3_BAY...",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forwarder := NewForwarder(tt.config, nil)

			// Test query string
			if tt.testQuery {
				resultURL, err := forwarder.AddToQueryString("http://example.com/path", microsoftUserInfo)
				if err != nil {
					t.Fatalf("AddToQueryString() error = %v", err)
				}

				u, _ := url.Parse(resultURL)
				for key, expected := range tt.expectedQuery {
					actual := u.Query().Get(key)
					if actual != expected {
						t.Errorf("Query param %s = %v, want %v", key, actual, expected)
					}
				}
			}

			// Test headers
			if tt.testHeaders {
				headers := make(http.Header)
				resultHeaders := forwarder.AddToHeaders(headers, microsoftUserInfo)

				for key, expected := range tt.expectedHeaders {
					actual := resultHeaders.Get(key)
					if actual != expected {
						t.Errorf("Header %s = %v, want %v", key, actual, expected)
					}
				}
			}
		})
	}
}

// TestOAuth2WithEncryption tests that OAuth2 fields work with encryption filters
func TestOAuth2WithEncryption(t *testing.T) {
	googleUserInfo := &UserInfo{
		Email:    "user@example.com",
		Username: "John Doe",
		Provider: "google",
		Extra: map[string]any{
			"_email":      "user@example.com",
			"_username":   "John Doe",
			"_avatar_url": "https://example.com/avatar.png",
			"secrets": map[string]any{
				"access_token": "ya29.a0AfH6SMBx...",
			},
		},
	}

	tests := []struct {
		name           string
		config         *config.ForwardingConfig
		testQuery      bool
		testHeaders    bool
		shouldEncrypt  map[string]bool
		shouldNotMatch map[string]string // Values that should NOT match (encrypted)
	}{
		{
			name: "Encrypt standardized email",
			config: &config.ForwardingConfig{
				Encryption: &config.EncryptionConfig{
					Key: "this-is-a-32-character-encryption-key-12345",
				},
				Fields: []config.ForwardingField{
					{Path: "_email", Query: "email", Filters: []string{"encrypt"}},
					{Path: "_username", Query: "username"}, // Not encrypted
				},
			},
			testQuery: true,
			shouldEncrypt: map[string]bool{
				"email": true,
			},
			shouldNotMatch: map[string]string{
				"email": "user@example.com", // Should be encrypted
			},
		},
		{
			name: "Encrypt access token in header",
			config: &config.ForwardingConfig{
				Encryption: &config.EncryptionConfig{
					Key: "this-is-a-32-character-encryption-key-12345",
				},
				Fields: []config.ForwardingField{
					{Path: "secrets.access_token", Header: "X-Token", Filters: []string{"encrypt"}},
				},
			},
			testHeaders: true,
			shouldEncrypt: map[string]bool{
				"X-Token": true,
			},
			shouldNotMatch: map[string]string{
				"X-Token": "ya29.a0AfH6SMBx...", // Should be encrypted
			},
		},
		{
			name: "Encrypt with compression",
			config: &config.ForwardingConfig{
				Encryption: &config.EncryptionConfig{
					Key: "this-is-a-32-character-encryption-key-12345",
				},
				Fields: []config.ForwardingField{
					{Path: "_email", Query: "email", Filters: []string{"encrypt", "zip"}},
				},
			},
			testQuery: true,
			shouldEncrypt: map[string]bool{
				"email": true,
			},
			shouldNotMatch: map[string]string{
				"email": "user@example.com", // Should be encrypted and compressed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forwarder := NewForwarder(tt.config, nil)

			// Test query string
			if tt.testQuery {
				resultURL, err := forwarder.AddToQueryString("http://example.com/path", googleUserInfo)
				if err != nil {
					t.Fatalf("AddToQueryString() error = %v", err)
				}

				u, _ := url.Parse(resultURL)
				for key, shouldEnc := range tt.shouldEncrypt {
					actual := u.Query().Get(key)
					if actual == "" {
						t.Errorf("Query param %s not found", key)
						continue
					}
					if shouldEnc && actual == tt.shouldNotMatch[key] {
						t.Errorf("Query param %s should be encrypted but got plain text: %v", key, actual)
					}
					// Check that encrypted value is base64-like
					if shouldEnc && !strings.Contains(actual, "=") && len(actual) < 20 {
						t.Errorf("Query param %s doesn't look encrypted: %v", key, actual)
					}
				}
			}

			// Test headers
			if tt.testHeaders {
				headers := make(http.Header)
				resultHeaders := forwarder.AddToHeaders(headers, googleUserInfo)

				for key, shouldEnc := range tt.shouldEncrypt {
					actual := resultHeaders.Get(key)
					if actual == "" {
						t.Errorf("Header %s not found", key)
						continue
					}
					if shouldEnc && actual == tt.shouldNotMatch[key] {
						t.Errorf("Header %s should be encrypted but got plain text: %v", key, actual)
					}
					// Check that encrypted value is base64-like
					if shouldEnc && !strings.Contains(actual, "=") && len(actual) < 20 {
						t.Errorf("Header %s doesn't look encrypted: %v", key, actual)
					}
				}
			}
		})
	}
}

// TestNonExistentPaths tests that non-existent paths are handled safely
func TestNonExistentPaths(t *testing.T) {
	googleUserInfo := &UserInfo{
		Email:    "user@example.com",
		Username: "John Doe",
		Provider: "google",
		Extra: map[string]any{
			"_email":      "user@example.com",
			"_username":   "John Doe",
			"_avatar_url": "https://example.com/avatar.png",
			"secrets": map[string]any{
				"access_token": "ya29.a0AfH6SMBx...",
			},
		},
	}

	tests := []struct {
		name                 string
		config               *config.ForwardingConfig
		testQuery            bool
		testHeaders          bool
		expectedEmptyQuery   []string
		expectedEmptyHeaders []string
		shouldNotError       bool
	}{
		{
			name: "Non-existent top-level field",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "nonexistent", Query: "ne"},
					{Path: "_email", Query: "email"}, // Valid field for comparison
				},
			},
			testQuery:          true,
			expectedEmptyQuery: []string{"ne"},
			shouldNotError:     true,
		},
		{
			name: "Non-existent nested field in extra",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "does_not_exist", Query: "dne"},
					{Path: "_username", Query: "username"}, // Valid field
				},
			},
			testQuery:          true,
			expectedEmptyQuery: []string{"dne"},
			shouldNotError:     true,
		},
		{
			name: "Deeply nested non-existent path",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "extra.deeply.nested.field.that.does.not.exist", Query: "deep"},
					{Path: "secrets.non_existent_token", Query: "token"},
					{Path: "_email", Query: "email"}, // Valid field
				},
			},
			testQuery:          true,
			expectedEmptyQuery: []string{"deep", "token"},
			shouldNotError:     true,
		},
		{
			name: "Non-existent fields in headers",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "nonexistent", Header: "X-NonExistent"},
					{Path: "extra.missing.field", Header: "X-Missing"},
					{Path: "_email", Header: "X-Email"}, // Valid field
				},
			},
			testHeaders:          true,
			expectedEmptyHeaders: []string{"X-NonExistent", "X-Missing"},
			shouldNotError:       true,
		},
		{
			name: "Non-existent path with filters",
			config: &config.ForwardingConfig{
				Encryption: &config.EncryptionConfig{
					Key: "this-is-a-32-character-encryption-key-12345",
				},
				Fields: []config.ForwardingField{
					{Path: "nonexistent", Query: "ne", Filters: []string{"encrypt"}},
					{Path: "_email", Query: "email"}, // Valid field
				},
			},
			testQuery:          true,
			expectedEmptyQuery: []string{"ne"},
			shouldNotError:     true,
		},
		{
			name: "All non-existent paths",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "field1", Query: "f1"},
					{Path: "field2", Header: "X-F2"},
					{Path: "extra.field3", Query: "f3"},
				},
			},
			testQuery:            true,
			testHeaders:          true,
			expectedEmptyQuery:   []string{"f1", "f3"},
			expectedEmptyHeaders: []string{"X-F2"},
			shouldNotError:       true,
		},
		{
			name: "Invalid path in secrets",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "secrets.invalid_token", Query: "token"},
					{Path: "secrets.another.nested.invalid", Header: "X-Token"},
					{Path: "secrets.access_token", Query: "valid_token"}, // Valid field
				},
			},
			testQuery:            true,
			testHeaders:          true,
			expectedEmptyQuery:   []string{"token"},
			expectedEmptyHeaders: []string{"X-Token"},
			shouldNotError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forwarder := NewForwarder(tt.config, nil)

			// Test query string
			if tt.testQuery {
				resultURL, err := forwarder.AddToQueryString("http://example.com/path", googleUserInfo)

				if tt.shouldNotError && err != nil {
					t.Fatalf("AddToQueryString() should not error but got: %v", err)
				}

				if err == nil {
					u, _ := url.Parse(resultURL)

					// Check that non-existent fields are NOT present (not added at all)
					for _, key := range tt.expectedEmptyQuery {
						if u.Query().Has(key) {
							t.Errorf("Query param %s should not be present for non-existent path, got: %v", key, u.Query().Get(key))
						}
					}

					// Verify valid fields still work
					if u.Query().Has("email") {
						if u.Query().Get("email") != "user@example.com" {
							t.Errorf("Valid field 'email' should still work, got: %v", u.Query().Get("email"))
						}
					}
					if u.Query().Has("username") {
						if u.Query().Get("username") != "John Doe" {
							t.Errorf("Valid field 'username' should still work, got: %v", u.Query().Get("username"))
						}
					}
					if u.Query().Has("valid_token") {
						if u.Query().Get("valid_token") != "ya29.a0AfH6SMBx..." {
							t.Errorf("Valid field 'valid_token' should still work, got: %v", u.Query().Get("valid_token"))
						}
					}
				}
			}

			// Test headers
			if tt.testHeaders {
				headers := make(http.Header)
				resultHeaders := forwarder.AddToHeaders(headers, googleUserInfo)

				// Check that non-existent fields are NOT present (not added at all)
				for _, key := range tt.expectedEmptyHeaders {
					if resultHeaders.Get(key) != "" {
						t.Errorf("Header %s should not be present for non-existent path, got: %v", key, resultHeaders.Get(key))
					}
				}

				// Verify valid fields still work
				if resultHeaders.Get("X-Email") != "" && resultHeaders.Get("X-Email") != "user@example.com" {
					t.Errorf("Valid header 'X-Email' should still work, got: %v", resultHeaders.Get("X-Email"))
				}
			}
		})
	}
}

// TestNonExistentPathsAllProviders tests non-existent paths across all OAuth2 providers
func TestNonExistentPathsAllProviders(t *testing.T) {
	providers := []struct {
		name     string
		userInfo *UserInfo
	}{
		{
			name: "Google",
			userInfo: &UserInfo{
				Email:    "user@example.com",
				Username: "John Doe",
				Provider: "google",
				Extra: map[string]any{
					"_email":      "user@example.com",
					"_username":   "John Doe",
					"_avatar_url": "https://example.com/avatar.png",
				},
			},
		},
		{
			name: "GitHub",
			userInfo: &UserInfo{
				Email:    "user@example.com",
				Username: "John Smith",
				Provider: "github",
				Extra: map[string]any{
					"_email":      "user@example.com",
					"_username":   "John Smith",
					"_avatar_url": "https://avatars.githubusercontent.com/u/12345",
				},
			},
		},
		{
			name: "Microsoft",
			userInfo: &UserInfo{
				Email:    "user@example.com",
				Username: "John Doe",
				Provider: "microsoft",
				Extra: map[string]any{
					"_email":      "user@example.com",
					"_username":   "John Doe",
					"_avatar_url": "",
				},
			},
		},
	}

	// Common config with mix of valid and invalid paths
	cfg := &config.ForwardingConfig{
		Fields: []config.ForwardingField{
			{Path: "_email", Query: "email"},
			{Path: "invalid_field", Query: "invalid"},
			{Path: "extra.nonexistent.deeply.nested", Query: "deep"},
			{Path: "_username", Header: "X-User"},
			{Path: "missing_header_field", Header: "X-Missing"},
		},
	}

	for _, provider := range providers {
		t.Run(provider.name, func(t *testing.T) {
			forwarder := NewForwarder(cfg, nil)

			// Test query string
			resultURL, err := forwarder.AddToQueryString("http://example.com/path", provider.userInfo)
			if err != nil {
				t.Fatalf("AddToQueryString() should not error for %s, got: %v", provider.name, err)
			}

			u, _ := url.Parse(resultURL)

			// Valid field should work
			if u.Query().Get("email") != "user@example.com" {
				t.Errorf("%s: email should be 'user@example.com', got: %v", provider.name, u.Query().Get("email"))
			}

			// Invalid fields should NOT be present (not added at all)
			if u.Query().Has("invalid") {
				t.Errorf("%s: invalid field should not be present, got: %v", provider.name, u.Query().Get("invalid"))
			}
			if u.Query().Has("deep") {
				t.Errorf("%s: deep nested field should not be present, got: %v", provider.name, u.Query().Get("deep"))
			}

			// Test headers
			headers := make(http.Header)
			resultHeaders := forwarder.AddToHeaders(headers, provider.userInfo)

			// Valid header should work
			if resultHeaders.Get("X-User") != provider.userInfo.Username {
				t.Errorf("%s: X-User should be '%s', got: %v", provider.name, provider.userInfo.Username, resultHeaders.Get("X-User"))
			}

			// Invalid header should NOT be present (not added at all)
			if resultHeaders.Get("X-Missing") != "" {
				t.Errorf("%s: X-Missing should not be present, got: %v", provider.name, resultHeaders.Get("X-Missing"))
			}
		})
	}
}

// TestExtremelyDeepPaths tests extremely deep non-existent paths
func TestExtremelyDeepPaths(t *testing.T) {
	userInfo := &UserInfo{
		Email:    "user@example.com",
		Username: "John Doe",
		Provider: "google",
		Extra: map[string]any{
			"_email": "user@example.com",
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": "exists",
				},
			},
		},
	}

	tests := []struct {
		name          string
		path          string
		shouldBeEmpty bool
	}{
		{
			name:          "Valid nested path",
			path:          "level1.level2.level3",
			shouldBeEmpty: false,
		},
		{
			name:          "One level too deep",
			path:          "level1.level2.level3.level4",
			shouldBeEmpty: true,
		},
		{
			name:          "Extremely deep non-existent path",
			path:          "a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z",
			shouldBeEmpty: true,
		},
		{
			name:          "Deep path starting from wrong branch",
			path:          "wrong.level2.level3",
			shouldBeEmpty: true,
		},
		{
			name:          "Path through standardized field (not a map)",
			path:          "_email.nested.field",
			shouldBeEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: tt.path, Query: "test", Header: "X-Test"},
				},
			}

			forwarder := NewForwarder(cfg, nil)

			// Test query
			resultURL, err := forwarder.AddToQueryString("http://example.com/path", userInfo)
			if err != nil {
				t.Fatalf("AddToQueryString() should not error, got: %v", err)
			}

			u, _ := url.Parse(resultURL)

			if tt.shouldBeEmpty {
				// Non-existent paths should NOT add query param at all
				if u.Query().Has("test") {
					t.Errorf("Path '%s' should not be present in query, got: %v", tt.path, u.Query().Get("test"))
				}
			} else {
				// Valid paths should have a value
				actual := u.Query().Get("test")
				if actual == "" {
					t.Errorf("Path '%s' should have a value, got empty", tt.path)
				}
			}

			// Test header
			headers := make(http.Header)
			resultHeaders := forwarder.AddToHeaders(headers, userInfo)

			if tt.shouldBeEmpty {
				// Non-existent paths should NOT add header at all
				if resultHeaders.Get("X-Test") != "" {
					t.Errorf("Path '%s' should not be present in headers, got: %v", tt.path, resultHeaders.Get("X-Test"))
				}
			} else {
				// Valid paths should have a header value
				actualHeader := resultHeaders.Get("X-Test")
				if actualHeader == "" {
					t.Errorf("Path '%s' should have a header value, got empty", tt.path)
				}
			}
		})
	}
}

// TestMultiplePathsSameDestination tests that when multiple paths map to the same
// destination (query param or header), the first existing path wins and non-existent
// paths don't overwrite valid data
func TestMultiplePathsSameDestination(t *testing.T) {
	tests := []struct {
		name         string
		userInfo     *UserInfo
		fields       []config.ForwardingField
		expectQuery  map[string]string // expected query parameters
		expectHeader map[string]string // expected headers
	}{
		{
			name: "First path exists - should use it",
			userInfo: &UserInfo{
				Email:    "user@example.com",
				Username: "John Doe",
				Provider: "google",
				Extra: map[string]any{
					"email":      "user@example.com",
					"user_email": "backup@example.com",
					"info_email": "other@example.com",
				},
			},
			fields: []config.ForwardingField{
				{Path: "email", Query: "email"},
				{Path: "user_email", Query: "email"},
				{Path: "info_email", Query: "email"},
			},
			expectQuery: map[string]string{
				"email": "user@example.com", // First path wins
			},
			expectHeader: map[string]string{},
		},
		{
			name: "First path doesn't exist - use second",
			userInfo: &UserInfo{
				Email:    "user@example.com",
				Username: "John Doe",
				Provider: "google",
				Extra: map[string]any{
					"user": map[string]any{
						"email": "user@example.com",
					},
					"info": map[string]any{
						"email": "backup@example.com",
					},
				},
			},
			fields: []config.ForwardingField{
				{Path: "email", Header: "X-Email"},
				{Path: "user.email", Header: "X-Email"},
				{Path: "info.email", Header: "X-Email"},
			},
			expectQuery: map[string]string{},
			expectHeader: map[string]string{
				"X-Email": "user@example.com", // Second path exists
			},
		},
		{
			name: "Non-existent path doesn't overwrite valid value",
			userInfo: &UserInfo{
				Email:    "user@example.com",
				Username: "John Doe",
				Provider: "google",
				Extra: map[string]any{
					"user": map[string]any{
						"email": "valid@example.com",
					},
				},
			},
			fields: []config.ForwardingField{
				{Path: "missing_field", Query: "test"}, // Doesn't exist - skip
				{Path: "user.email", Query: "test"},    // Exists - should be used
				{Path: "info.email", Query: "test"},    // Doesn't exist - should NOT overwrite
			},
			expectQuery: map[string]string{
				"test": "valid@example.com",
			},
			expectHeader: map[string]string{},
		},
		{
			name: "Multiple destinations with different priorities",
			userInfo: &UserInfo{
				Email:    "user@example.com",
				Username: "John Doe",
				Provider: "google",
				Extra: map[string]any{
					"_email":      "standard@example.com",
					"_username":   "Standard User",
					"custom_name": "Custom Name",
				},
			},
			fields: []config.ForwardingField{
				// Email: first path exists
				{Path: "_email", Query: "email"},
				{Path: "user.email", Query: "email"},
				// Name: first doesn't exist, second exists
				{Path: "display_name", Header: "X-Name"},
				{Path: "_username", Header: "X-Name"},
				{Path: "custom_name", Header: "X-Name"},
			},
			expectQuery: map[string]string{
				"email": "standard@example.com",
			},
			expectHeader: map[string]string{
				"X-Name": "Standard User", // Second path is first existing
			},
		},
		{
			name: "All paths non-existent - nothing should be added",
			userInfo: &UserInfo{
				Email:    "user@example.com",
				Username: "John Doe",
				Provider: "google",
				Extra: map[string]any{
					"_email": "user@example.com",
				},
			},
			fields: []config.ForwardingField{
				{Path: "nonexistent1", Query: "missing"},
				{Path: "nonexistent2", Query: "missing"},
				{Path: "nonexistent3", Header: "X-Missing"},
				{Path: "nonexistent4", Header: "X-Missing"},
			},
			expectQuery: map[string]string{
				// Nothing should be added
			},
			expectHeader: map[string]string{
				// Nothing should be added
			},
		},
		{
			name: "Deep nested paths with priority",
			userInfo: &UserInfo{
				Email:    "",
				Username: "John Doe",
				Provider: "google",
				Extra: map[string]any{
					"level1": map[string]any{
						"level2": map[string]any{
							"email": "deep@example.com",
						},
					},
					"shallow": map[string]any{
						"email": "shallow@example.com",
					},
				},
			},
			fields: []config.ForwardingField{
				{Path: "email", Query: "email"},                      // Top-level email is empty - skip
				{Path: "level1.level2.level3.email", Query: "email"}, // Too deep - doesn't exist
				{Path: "shallow.email", Query: "email"},              // Exists - should win
				{Path: "level1.level2.email", Query: "email"},        // Exists but comes later
			},
			expectQuery: map[string]string{
				"email": "shallow@example.com", // First existing path
			},
			expectHeader: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ForwardingConfig{
				Fields: tt.fields,
			}

			forwarder := NewForwarder(cfg, nil)

			// Test query string
			resultURL, err := forwarder.AddToQueryString("http://example.com/path", tt.userInfo)
			if err != nil {
				t.Fatalf("AddToQueryString() error = %v", err)
			}

			u, _ := url.Parse(resultURL)

			// Verify expected query parameters
			for key, expectedValue := range tt.expectQuery {
				actualValue := u.Query().Get(key)
				if actualValue != expectedValue {
					t.Errorf("Query param %s = %v, want %v", key, actualValue, expectedValue)
				}
			}

			// Verify no unexpected query parameters were added
			// (we only check keys that were configured)
			configuredQueryKeys := make(map[string]bool)
			for _, field := range tt.fields {
				if field.Query != "" {
					configuredQueryKeys[field.Query] = true
				}
			}
			for queryKey := range configuredQueryKeys {
				if _, expected := tt.expectQuery[queryKey]; !expected {
					if u.Query().Has(queryKey) {
						t.Errorf("Query param %s should not be present, got: %v", queryKey, u.Query().Get(queryKey))
					}
				}
			}

			// Test headers
			headers := make(http.Header)
			resultHeaders := forwarder.AddToHeaders(headers, tt.userInfo)

			// Verify expected headers
			for key, expectedValue := range tt.expectHeader {
				actualValue := resultHeaders.Get(key)
				if actualValue != expectedValue {
					t.Errorf("Header %s = %v, want %v", key, actualValue, expectedValue)
				}
			}

			// Verify no unexpected headers were added
			configuredHeaderKeys := make(map[string]bool)
			for _, field := range tt.fields {
				if field.Header != "" {
					configuredHeaderKeys[field.Header] = true
				}
			}
			for headerKey := range configuredHeaderKeys {
				if _, expected := tt.expectHeader[headerKey]; !expected {
					if resultHeaders.Get(headerKey) != "" {
						t.Errorf("Header %s should not be present, got: %v", headerKey, resultHeaders.Get(headerKey))
					}
				}
			}
		})
	}
}

// TestGitHubWithLoginFallback tests GitHub's name -> login fallback behavior
func TestGitHubWithLoginFallback(t *testing.T) {
	// GitHub user without display name (falls back to login)
	githubUserNoName := &UserInfo{
		Email:    "user@example.com",
		Username: "johnsmith", // login name used as fallback
		Provider: "github",
		Extra: map[string]any{
			"_email":      "user@example.com",
			"_username":   "johnsmith", // login used as username
			"_avatar_url": "https://avatars.githubusercontent.com/u/12345?v=4",
			"email":       "user@example.com",
			"name":        "", // Empty name
			"login":       "johnsmith",
			"avatar_url":  "https://avatars.githubusercontent.com/u/12345?v=4",
		},
	}

	cfg := &config.ForwardingConfig{
		Fields: []config.ForwardingField{
			{Path: "_username", Query: "username", Header: "X-User-Name"},
			{Path: "login", Query: "login"},
		},
	}

	forwarder := NewForwarder(cfg, nil)

	// Test query
	resultURL, err := forwarder.AddToQueryString("http://example.com/path", githubUserNoName)
	if err != nil {
		t.Fatalf("AddToQueryString() error = %v", err)
	}

	u, _ := url.Parse(resultURL)
	if u.Query().Get("username") != "johnsmith" {
		t.Errorf("_username should fallback to login, got %v", u.Query().Get("username"))
	}
	if u.Query().Get("login") != "johnsmith" {
		t.Errorf("login = %v, want johnsmith", u.Query().Get("login"))
	}

	// Test headers
	headers := make(http.Header)
	resultHeaders := forwarder.AddToHeaders(headers, githubUserNoName)
	if resultHeaders.Get("X-User-Name") != "johnsmith" {
		t.Errorf("X-User-Name should fallback to login, got %v", resultHeaders.Get("X-User-Name"))
	}
}
