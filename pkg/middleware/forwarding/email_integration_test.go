package forwarding

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
)

// TestEmailAuthIntegration tests the complete flow from email authentication to forwarding
func TestEmailAuthIntegration(t *testing.T) {
	// Simulate UserInfo created by email authentication
	emailUserInfo := &UserInfo{
		Email:    "john.doe@example.com",
		Username: "john.doe",
		Provider: "email",
		Extra: map[string]any{
			// Standardized fields (populated by email auth handler)
			"_email":      "john.doe@example.com",
			"_username":   "john.doe",
			"_avatar_url": "",
			// Email-specific fields
			"userpart": "john.doe",
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
				"email":    "john.doe@example.com",
				"username": "john.doe",
				"avatar":   "",
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
				"X-User-Email":  "john.doe@example.com",
				"X-User-Name":   "john.doe",
				"X-User-Avatar": "",
			},
		},
		{
			name: "Email-specific fields (userpart)",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "userpart", Query: "user"},
					{Path: "email", Query: "email"},
					{Path: "provider", Query: "provider"},
				},
			},
			testQuery: true,
			expectedQuery: map[string]string{
				"user":     "john.doe",
				"email":    "john.doe@example.com",
				"provider": "email",
			},
		},
		{
			name: "Mixed query and headers",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "_email", Query: "email"},
					{Path: "_username", Header: "X-User-Name"},
					{Path: "userpart", Query: "user"},
				},
			},
			testQuery:   true,
			testHeaders: true,
			expectedQuery: map[string]string{
				"email": "john.doe@example.com",
				"user":  "john.doe",
			},
			expectedHeaders: map[string]string{
				"X-User-Name": "john.doe",
			},
		},
		{
			name: "All standardized and email-specific fields",
			config: &config.ForwardingConfig{
				Fields: []config.ForwardingField{
					{Path: "_email", Query: "std_email"},
					{Path: "_username", Query: "std_username"},
					{Path: "_avatar_url", Query: "std_avatar"},
					{Path: "userpart", Query: "userpart"},
					{Path: "email", Query: "email"},
					{Path: "username", Query: "username"},
					{Path: "provider", Query: "provider"},
				},
			},
			testQuery: true,
			expectedQuery: map[string]string{
				"std_email":    "john.doe@example.com",
				"std_username": "john.doe",
				"std_avatar":   "",
				"userpart":     "john.doe",
				"email":        "john.doe@example.com",
				"username":     "john.doe",
				"provider":     "email",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create forwarder
			forwarder := NewForwarder(tt.config, []config.OAuth2Provider{})

			if tt.testQuery {
				// Test query string forwarding
				baseURL := "http://example.com/path"
				resultURL, err := forwarder.AddToQueryString(baseURL, emailUserInfo)
				if err != nil {
					t.Fatalf("AddToQueryString failed: %v", err)
				}

				// Parse the result URL
				u, err := url.Parse(resultURL)
				if err != nil {
					t.Fatalf("Failed to parse result URL: %v", err)
				}

				// Check expected query parameters
				for key, expectedValue := range tt.expectedQuery {
					actualValue := u.Query().Get(key)
					if actualValue != expectedValue {
						t.Errorf("Query param %q = %q, want %q", key, actualValue, expectedValue)
					}
				}
			}

			if tt.testHeaders {
				// Test header forwarding
				headers := http.Header{}
				result := forwarder.AddToHeaders(headers, emailUserInfo)

				// Check expected headers
				for key, expectedValue := range tt.expectedHeaders {
					actualValue := result.Get(key)
					if actualValue != expectedValue {
						t.Errorf("Header %q = %q, want %q", key, actualValue, expectedValue)
					}
				}
			}
		})
	}
}

// TestEmailAuthWithEncryption tests email auth with encryption
func TestEmailAuthWithEncryption(t *testing.T) {
	emailUserInfo := &UserInfo{
		Email:    "john.doe@example.com",
		Username: "john.doe",
		Provider: "email",
		Extra: map[string]any{
			"_email":      "john.doe@example.com",
			"_username":   "john.doe",
			"_avatar_url": "",
			"userpart":    "john.doe",
		},
	}

	encryptionKey := "test-encryption-key-32-chars!!"

	tests := []struct {
		name           string
		config         *config.ForwardingConfig
		expectedFields map[string]bool // Fields that should be present
		checkEncrypted bool            // Whether to check if value is encrypted
	}{
		{
			name: "Encrypt standardized email",
			config: &config.ForwardingConfig{
				Encryption: &config.EncryptionConfig{
					Key: encryptionKey,
				},
				Fields: []config.ForwardingField{
					{Path: "_email", Query: "email", Filters: config.FilterList{"encrypt"}},
					{Path: "_username", Query: "username"},
				},
			},
			expectedFields: map[string]bool{
				"email":    true,
				"username": true,
			},
			checkEncrypted: true,
		},
		{
			name: "Encrypt userpart",
			config: &config.ForwardingConfig{
				Encryption: &config.EncryptionConfig{
					Key: encryptionKey,
				},
				Fields: []config.ForwardingField{
					{Path: "userpart", Query: "user", Filters: config.FilterList{"encrypt"}},
					{Path: "email", Query: "email"},
				},
			},
			expectedFields: map[string]bool{
				"user":  true,
				"email": true,
			},
			checkEncrypted: true,
		},
		{
			name: "Encrypt with compression",
			config: &config.ForwardingConfig{
				Encryption: &config.EncryptionConfig{
					Key: encryptionKey,
				},
				Fields: []config.ForwardingField{
					{Path: "_email", Query: "email", Filters: config.FilterList{"encrypt", "zip"}},
				},
			},
			expectedFields: map[string]bool{
				"email": true,
			},
			checkEncrypted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forwarder := NewForwarder(tt.config, []config.OAuth2Provider{})

			// Test query string forwarding
			baseURL := "http://example.com/path"
			resultURL, err := forwarder.AddToQueryString(baseURL, emailUserInfo)
			if err != nil {
				t.Fatalf("AddToQueryString failed: %v", err)
			}

			// Parse the result URL
			u, err := url.Parse(resultURL)
			if err != nil {
				t.Fatalf("Failed to parse result URL: %v", err)
			}

			// Check that expected fields exist
			for field := range tt.expectedFields {
				value := u.Query().Get(field)
				if value == "" {
					t.Errorf("Expected field %q not found in query string", field)
				}

				// For encrypted fields, check that they don't contain plain text
				if tt.checkEncrypted && field != "email" && field != "username" {
					// Skip non-encrypted fields in this test
					continue
				}

				if tt.checkEncrypted {
					// Check that encrypted value doesn't match original
					originalValue := ""
					switch field {
					case "email":
						originalValue = emailUserInfo.Email
					case "user":
						if up, ok := emailUserInfo.Extra["userpart"].(string); ok {
							originalValue = up
						}
					}

					// Find if this field should be encrypted
					shouldBeEncrypted := false
					for _, f := range tt.config.Fields {
						if (f.Query == field) && len(f.Filters) > 0 {
							for _, filter := range f.Filters {
								if filter == "encrypt" {
									shouldBeEncrypted = true
									break
								}
							}
						}
					}

					if shouldBeEncrypted && value == originalValue {
						t.Errorf("Field %q appears to be unencrypted (value matches original)", field)
					} else if shouldBeEncrypted && !strings.Contains(value, "=") {
						// Encrypted values should be base64 encoded (contain =)
						t.Logf("Note: Encrypted field %q value: %q (length: %d)", field, value, len(value))
					}
				}
			}
		})
	}
}

// TestEmailAuthWithDifferentEmails tests email auth with various email formats
func TestEmailAuthWithDifferentEmails(t *testing.T) {
	testCases := []struct {
		name              string
		email             string
		expectedUserpart  string
		expectedEmail     string
		expectedUsername  string
		expectedAvatarURL string
	}{
		{
			name:              "Simple email",
			email:             "user@example.com",
			expectedUserpart:  "user",
			expectedEmail:     "user@example.com",
			expectedUsername:  "user",
			expectedAvatarURL: "",
		},
		{
			name:              "Email with dots",
			email:             "john.doe@example.com",
			expectedUserpart:  "john.doe",
			expectedEmail:     "john.doe@example.com",
			expectedUsername:  "john.doe",
			expectedAvatarURL: "",
		},
		{
			name:              "Email with plus",
			email:             "user+tag@example.com",
			expectedUserpart:  "user+tag",
			expectedEmail:     "user+tag@example.com",
			expectedUsername:  "user+tag",
			expectedAvatarURL: "",
		},
		{
			name:              "Email with numbers",
			email:             "user123@example.com",
			expectedUserpart:  "user123",
			expectedEmail:     "user123@example.com",
			expectedUsername:  "user123",
			expectedAvatarURL: "",
		},
		{
			name:              "Email with hyphen",
			email:             "first-last@example.com",
			expectedUserpart:  "first-last",
			expectedEmail:     "first-last@example.com",
			expectedUsername:  "first-last",
			expectedAvatarURL: "",
		},
	}

	cfg := &config.ForwardingConfig{
		Fields: []config.ForwardingField{
			{Path: "_email", Query: "email"},
			{Path: "_username", Query: "username"},
			{Path: "_avatar_url", Query: "avatar"},
			{Path: "userpart", Query: "userpart"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userInfo := &UserInfo{
				Email:    tc.email,
				Username: tc.expectedUserpart,
				Provider: "email",
				Extra: map[string]any{
					"_email":      tc.expectedEmail,
					"_username":   tc.expectedUsername,
					"_avatar_url": tc.expectedAvatarURL,
					"userpart":    tc.expectedUserpart,
				},
			}

			forwarder := NewForwarder(cfg, []config.OAuth2Provider{})

			// Test query string forwarding
			baseURL := "http://example.com/path"
			resultURL, err := forwarder.AddToQueryString(baseURL, userInfo)
			if err != nil {
				t.Fatalf("AddToQueryString failed: %v", err)
			}

			// Parse the result URL
			u, err := url.Parse(resultURL)
			if err != nil {
				t.Fatalf("Failed to parse result URL: %v", err)
			}

			// Check query parameters
			if got := u.Query().Get("email"); got != tc.expectedEmail {
				t.Errorf("email = %q, want %q", got, tc.expectedEmail)
			}
			if got := u.Query().Get("username"); got != tc.expectedUsername {
				t.Errorf("username = %q, want %q", got, tc.expectedUsername)
			}
			if got := u.Query().Get("avatar"); got != tc.expectedAvatarURL {
				t.Errorf("avatar = %q, want %q", got, tc.expectedAvatarURL)
			}
			if got := u.Query().Get("userpart"); got != tc.expectedUserpart {
				t.Errorf("userpart = %q, want %q", got, tc.expectedUserpart)
			}
		})
	}
}
