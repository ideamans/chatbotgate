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
