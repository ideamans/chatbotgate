package forwarding

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
)

func TestForwarder_AddToQueryString_PlainText(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields: []config.ForwardingField{
			{Path: "email", Query: "email"},
			{Path: "username", Query: "user"},
		},
	}

	forwarder := NewForwarder(cfg, nil)
	userInfo := &UserInfo{Username: "john", Email: "john@example.com"}

	result, err := forwarder.AddToQueryString("http://example.com/path", userInfo)
	if err != nil {
		t.Fatalf("AddToQueryString() error = %v", err)
	}

	u, _ := url.Parse(result)
	if u.Query().Get("email") != "john@example.com" {
		t.Errorf("email = %v, want %v", u.Query().Get("email"), "john@example.com")
	}
	if u.Query().Get("user") != "john" {
		t.Errorf("user = %v, want %v", u.Query().Get("user"), "john")
	}
}

func TestForwarder_AddToHeaders_PlainText(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields: []config.ForwardingField{
			{Path: "email", Header: "X-Email"},
			{Path: "username", Header: "X-User"},
		},
	}

	forwarder := NewForwarder(cfg, nil)
	userInfo := &UserInfo{Username: "john", Email: "john@example.com"}

	headers := make(http.Header)
	result := forwarder.AddToHeaders(headers, userInfo)

	if result.Get("X-Email") != "john@example.com" {
		t.Errorf("X-Email = %v, want %v", result.Get("X-Email"), "john@example.com")
	}
	if result.Get("X-User") != "john" {
		t.Errorf("X-User = %v, want %v", result.Get("X-User"), "john")
	}
}

func TestForwarder_AddToQueryString_WithEncrypt(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Encryption: &config.EncryptionConfig{
			Key: "this-is-a-32-character-encryption-key-12345",
		},
		Fields: []config.ForwardingField{
			{Path: "email", Query: "email", Filters: []string{"encrypt"}},
		},
	}

	forwarder := NewForwarder(cfg, nil)
	userInfo := &UserInfo{Email: "john@example.com"}

	result, err := forwarder.AddToQueryString("http://example.com/path", userInfo)
	if err != nil {
		t.Fatalf("AddToQueryString() error = %v", err)
	}

	u, _ := url.Parse(result)
	encryptedEmail := u.Query().Get("email")
	if encryptedEmail == "" {
		t.Fatal("email parameter not found")
	}
	if encryptedEmail == "john@example.com" {
		t.Error("email should be encrypted, but got plain text")
	}
}

func TestForwarder_PathResolution(t *testing.T) {
	userInfo := &UserInfo{
		Username: "john",
		Email:    "john@example.com",
		Provider: "google",
		Extra: map[string]interface{}{
			"avatar_url": "https://example.com/avatar.png",
			"secrets": map[string]interface{}{
				"access_token": "secret-token-123",
			},
		},
	}

	tests := []struct {
		name      string
		path      string
		want      string
		wantError bool
	}{
		{
			name: "username",
			path: "username",
			want: "john",
		},
		{
			name: "email",
			path: "email",
			want: "john@example.com",
		},
		{
			name: "provider",
			path: "provider",
			want: "google",
		},
		{
			name: "extra.avatar_url",
			path: "extra.avatar_url",
			want: "https://example.com/avatar.png",
		},
		{
			name: "extra.secrets.access_token",
			path: "extra.secrets.access_token",
			want: "secret-token-123",
		},
		{
			name:      "nonexistent",
			path:      "nonexistent",
			wantError: true,
		},
	}

	cfg := &config.ForwardingConfig{Fields: []config.ForwardingField{}}
	forwarder := NewForwarder(cfg, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := forwarder.getFieldValue(userInfo, tt.path)
			if (err != nil) != tt.wantError {
				t.Errorf("getFieldValue() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && got != tt.want {
				t.Errorf("getFieldValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForwarder_EntireObject(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields: []config.ForwardingField{
			{Path: ".", Query: "userinfo"},
		},
	}

	forwarder := NewForwarder(cfg, nil)
	userInfo := &UserInfo{
		Username: "john",
		Email:    "john@example.com",
		Provider: "google",
	}

	result, err := forwarder.AddToQueryString("http://example.com/path", userInfo)
	if err != nil {
		t.Fatalf("AddToQueryString() error = %v", err)
	}

	u, _ := url.Parse(result)
	userInfoJSON := u.Query().Get("userinfo")
	if userInfoJSON == "" {
		t.Fatal("userinfo parameter not found")
	}

	// Should contain JSON-encoded user info
	if userInfoJSON == "" || userInfoJSON[0] != '{' {
		t.Errorf("userinfo should be JSON, got %v", userInfoJSON)
	}
}

func TestForwarder_NonExistentPathsNotAdded(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields: []config.ForwardingField{
			{Path: "email", Query: "email"},
			{Path: "nonexistent_field", Query: "missing"},
			{Path: "extra.deep.nested.missing", Query: "deep"},
			{Path: "username", Header: "X-User"},
			{Path: "invalid_field", Header: "X-Invalid"},
		},
	}

	forwarder := NewForwarder(cfg, nil)
	userInfo := &UserInfo{
		Username: "john",
		Email:    "john@example.com",
		Extra:    map[string]interface{}{},
	}

	// Test query parameters
	result, err := forwarder.AddToQueryString("http://example.com/path", userInfo)
	if err != nil {
		t.Fatalf("AddToQueryString() error = %v", err)
	}

	u, _ := url.Parse(result)

	// Valid field should be present
	if u.Query().Get("email") != "john@example.com" {
		t.Errorf("email should be present, got %v", u.Query().Get("email"))
	}

	// Non-existent fields should NOT be present
	if u.Query().Has("missing") {
		t.Errorf("missing should not be present, got %v", u.Query().Get("missing"))
	}
	if u.Query().Has("deep") {
		t.Errorf("deep should not be present, got %v", u.Query().Get("deep"))
	}

	// Test headers
	headers := make(http.Header)
	resultHeaders := forwarder.AddToHeaders(headers, userInfo)

	// Valid header should be present
	if resultHeaders.Get("X-User") != "john" {
		t.Errorf("X-User should be present, got %v", resultHeaders.Get("X-User"))
	}

	// Non-existent field should NOT be present
	if resultHeaders.Get("X-Invalid") != "" {
		t.Errorf("X-Invalid should not be present, got %v", resultHeaders.Get("X-Invalid"))
	}
}

func TestForwarder_MultiplePathsSameDestination(t *testing.T) {
	tests := []struct {
		name             string
		fields           []config.ForwardingField
		userInfo         *UserInfo
		expectQuery      map[string]string
		expectHeader     map[string]string
		notPresentQuery  []string
		notPresentHeader []string
	}{
		{
			name: "First path exists - use it",
			fields: []config.ForwardingField{
				{Path: "email", Query: "user_email"},
				{Path: "extra.backup_email", Query: "user_email"},
			},
			userInfo: &UserInfo{
				Email: "primary@example.com",
				Extra: map[string]interface{}{
					"backup_email": "backup@example.com",
				},
			},
			expectQuery: map[string]string{
				"user_email": "primary@example.com",
			},
		},
		{
			name: "First path missing - use second",
			fields: []config.ForwardingField{
				{Path: "extra.primary_email", Header: "X-Email"},
				{Path: "email", Header: "X-Email"},
			},
			userInfo: &UserInfo{
				Email: "john@example.com",
				Extra: map[string]interface{}{},
			},
			expectHeader: map[string]string{
				"X-Email": "john@example.com",
			},
		},
		{
			name: "All paths missing - nothing added",
			fields: []config.ForwardingField{
				{Path: "extra.email1", Query: "email"},
				{Path: "extra.email2", Query: "email"},
				{Path: "extra.email3", Query: "email"},
			},
			userInfo: &UserInfo{
				Email: "john@example.com",
				Extra: map[string]interface{}{},
			},
			notPresentQuery: []string{"email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ForwardingConfig{
				Fields: tt.fields,
			}

			forwarder := NewForwarder(cfg, nil)

			// Test query string
			if len(tt.expectQuery) > 0 || len(tt.notPresentQuery) > 0 {
				result, err := forwarder.AddToQueryString("http://example.com/path", tt.userInfo)
				if err != nil {
					t.Fatalf("AddToQueryString() error = %v", err)
				}

				u, _ := url.Parse(result)

				for key, expectedValue := range tt.expectQuery {
					if u.Query().Get(key) != expectedValue {
						t.Errorf("Query %s = %v, want %v", key, u.Query().Get(key), expectedValue)
					}
				}

				for _, key := range tt.notPresentQuery {
					if u.Query().Has(key) {
						t.Errorf("Query %s should not be present, got %v", key, u.Query().Get(key))
					}
				}
			}

			// Test headers
			if len(tt.expectHeader) > 0 || len(tt.notPresentHeader) > 0 {
				headers := make(http.Header)
				resultHeaders := forwarder.AddToHeaders(headers, tt.userInfo)

				for key, expectedValue := range tt.expectHeader {
					if resultHeaders.Get(key) != expectedValue {
						t.Errorf("Header %s = %v, want %v", key, resultHeaders.Get(key), expectedValue)
					}
				}

				for _, key := range tt.notPresentHeader {
					if resultHeaders.Get(key) != "" {
						t.Errorf("Header %s should not be present, got %v", key, resultHeaders.Get(key))
					}
				}
			}
		})
	}
}
