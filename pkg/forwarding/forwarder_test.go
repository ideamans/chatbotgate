package forwarding

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/config"
)

func TestForwarder_AddToQueryString_Disabled(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields:      []string{"username", "email"},
		QueryString: config.ForwardingMethodConfig{Enabled: false},
	}

	forwarder := NewForwarder(cfg)
	userInfo := &UserInfo{Username: "john", Email: "john@example.com"}

	result, err := forwarder.AddToQueryString("http://example.com/path", userInfo)
	if err != nil {
		t.Fatalf("AddToQueryString() error = %v", err)
	}

	// Should return original URL unchanged
	if result != "http://example.com/path" {
		t.Errorf("AddToQueryString() = %v, want %v", result, "http://example.com/path")
	}
}

func TestForwarder_AddToQueryString_PlainText(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields:      []string{"username", "email"},
		QueryString: config.ForwardingMethodConfig{Enabled: true, Encrypt: false},
	}

	forwarder := NewForwarder(cfg)
	userInfo := &UserInfo{Username: "john", Email: "john@example.com"}

	result, err := forwarder.AddToQueryString("http://example.com/path", userInfo)
	if err != nil {
		t.Fatalf("AddToQueryString() error = %v", err)
	}

	// Parse result URL
	u, err := url.Parse(result)
	if err != nil {
		t.Fatalf("Failed to parse result URL: %v", err)
	}

	// Check query parameters (chatbotgate.* format)
	if u.Query().Get("chatbotgate.user") != "john" {
		t.Errorf("chatbotgate.user = %v, want %v", u.Query().Get("chatbotgate.user"), "john")
	}
	if u.Query().Get("chatbotgate.email") != "john@example.com" {
		t.Errorf("chatbotgate.email = %v, want %v", u.Query().Get("chatbotgate.email"), "john@example.com")
	}
}

func TestForwarder_AddToQueryString_Encrypted(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields:      []string{"username", "email"},
		QueryString: config.ForwardingMethodConfig{Enabled: true, Encrypt: true},
		Encryption:  config.EncryptionConfig{Key: "this-is-a-32-character-encryption-key"},
	}

	forwarder := NewForwarder(cfg)
	userInfo := &UserInfo{Username: "john", Email: "john@example.com"}

	result, err := forwarder.AddToQueryString("http://example.com/path?existing=param", userInfo)
	if err != nil {
		t.Fatalf("AddToQueryString() error = %v", err)
	}

	// Parse result URL
	u, err := url.Parse(result)
	if err != nil {
		t.Fatalf("Failed to parse result URL: %v", err)
	}

	// Check that individual encrypted parameters exist
	encryptedUser := u.Query().Get("chatbotgate.user")
	if encryptedUser == "" {
		t.Error("chatbotgate.user parameter not found")
	}

	encryptedEmail := u.Query().Get("chatbotgate.email")
	if encryptedEmail == "" {
		t.Error("chatbotgate.email parameter not found")
	}

	// Check that original parameter is preserved
	if u.Query().Get("existing") != "param" {
		t.Errorf("existing parameter = %v, want %v", u.Query().Get("existing"), "param")
	}

	// Verify we can decrypt individual values
	decryptedUser, err := forwarder.encryptor.Decrypt(encryptedUser)
	if err != nil {
		t.Fatalf("Failed to decrypt chatbotgate.user parameter: %v", err)
	}
	if decryptedUser != "john" {
		t.Errorf("decrypted username = %v, want %v", decryptedUser, "john")
	}

	decryptedEmail, err := forwarder.encryptor.Decrypt(encryptedEmail)
	if err != nil {
		t.Fatalf("Failed to decrypt chatbotgate.email parameter: %v", err)
	}
	if decryptedEmail != "john@example.com" {
		t.Errorf("decrypted email = %v, want %v", decryptedEmail, "john@example.com")
	}
}

func TestForwarder_AddToQueryString_SelectedFields(t *testing.T) {
	tests := []struct {
		name          string
		fields        []string
		userInfo      *UserInfo
		wantUsername  bool
		wantEmail     bool
	}{
		{
			name:         "only username",
			fields:       []string{"username"},
			userInfo:     &UserInfo{Username: "john", Email: "john@example.com"},
			wantUsername: true,
			wantEmail:    false,
		},
		{
			name:         "only email",
			fields:       []string{"email"},
			userInfo:     &UserInfo{Username: "john", Email: "john@example.com"},
			wantUsername: false,
			wantEmail:    true,
		},
		{
			name:         "both fields",
			fields:       []string{"username", "email"},
			userInfo:     &UserInfo{Username: "john", Email: "john@example.com"},
			wantUsername: true,
			wantEmail:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ForwardingConfig{
				Fields:      tt.fields,
				QueryString: config.ForwardingMethodConfig{Enabled: true, Encrypt: false},
			}

			forwarder := NewForwarder(cfg)
			result, err := forwarder.AddToQueryString("http://example.com/", tt.userInfo)
			if err != nil {
				t.Fatalf("AddToQueryString() error = %v", err)
			}

			u, _ := url.Parse(result)
			hasUsername := u.Query().Get("chatbotgate.user") != ""
			hasEmail := u.Query().Get("chatbotgate.email") != ""

			if hasUsername != tt.wantUsername {
				t.Errorf("username presence = %v, want %v", hasUsername, tt.wantUsername)
			}
			if hasEmail != tt.wantEmail {
				t.Errorf("email presence = %v, want %v", hasEmail, tt.wantEmail)
			}
		})
	}
}

func TestForwarder_AddToHeaders_Disabled(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields: []string{"username", "email"},
		Header: config.ForwardingHeaderConfig{Enabled: false},
	}

	forwarder := NewForwarder(cfg)
	userInfo := &UserInfo{Username: "john", Email: "john@example.com"}

	headers := make(http.Header)
	headers.Set("Existing-Header", "value")

	result := forwarder.AddToHeaders(headers, userInfo)

	// Should only have existing header
	if result.Get("Existing-Header") != "value" {
		t.Error("Existing header was modified")
	}
	if result.Get("X-Forwarded-User") != "" {
		t.Error("User header should not be added when disabled")
	}
	if result.Get("X-Forwarded-Email") != "" {
		t.Error("Email header should not be added when disabled")
	}
}

func TestForwarder_AddToHeaders_PlainText(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields: []string{"username", "email"},
		Header: config.ForwardingHeaderConfig{Enabled: true, Encrypt: false},
	}

	forwarder := NewForwarder(cfg)
	userInfo := &UserInfo{Username: "john", Email: "john@example.com"}

	headers := make(http.Header)
	result := forwarder.AddToHeaders(headers, userInfo)

	if result.Get("X-Forwarded-User") != "john" {
		t.Errorf("X-Forwarded-User header = %v, want %v", result.Get("X-Forwarded-User"), "john")
	}
	if result.Get("X-Forwarded-Email") != "john@example.com" {
		t.Errorf("X-Forwarded-Email header = %v, want %v", result.Get("X-Forwarded-Email"), "john@example.com")
	}
}

func TestForwarder_AddToHeaders_Encrypted(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields: []string{"username", "email"},
		Header: config.ForwardingHeaderConfig{Enabled: true, Encrypt: true},
		Encryption: config.EncryptionConfig{Key: "this-is-a-32-character-encryption-key"},
	}

	forwarder := NewForwarder(cfg)
	userInfo := &UserInfo{Username: "john", Email: "john@example.com"}

	headers := make(http.Header)
	result := forwarder.AddToHeaders(headers, userInfo)

	// Check that individual encrypted headers exist
	encryptedUser := result.Get("X-Forwarded-User")
	if encryptedUser == "" {
		t.Error("X-Forwarded-User header not found")
	}

	encryptedEmail := result.Get("X-Forwarded-Email")
	if encryptedEmail == "" {
		t.Error("X-Forwarded-Email header not found")
	}

	// Verify we can decrypt individual values
	decryptedUser, err := forwarder.encryptor.Decrypt(encryptedUser)
	if err != nil {
		t.Fatalf("Failed to decrypt X-Forwarded-User header: %v", err)
	}
	if decryptedUser != "john" {
		t.Errorf("decrypted username = %v, want %v", decryptedUser, "john")
	}

	decryptedEmail, err := forwarder.encryptor.Decrypt(encryptedEmail)
	if err != nil {
		t.Fatalf("Failed to decrypt X-Forwarded-Email header: %v", err)
	}
	if decryptedEmail != "john@example.com" {
		t.Errorf("decrypted email = %v, want %v", decryptedEmail, "john@example.com")
	}
}

func TestForwarder_AddToHeaders_SelectedFields(t *testing.T) {
	tests := []struct {
		name         string
		fields       []string
		userInfo     *UserInfo
		wantUsername bool
		wantEmail    bool
	}{
		{
			name:         "only username",
			fields:       []string{"username"},
			userInfo:     &UserInfo{Username: "john", Email: "john@example.com"},
			wantUsername: true,
			wantEmail:    false,
		},
		{
			name:         "only email",
			fields:       []string{"email"},
			userInfo:     &UserInfo{Username: "john", Email: "john@example.com"},
			wantUsername: false,
			wantEmail:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ForwardingConfig{
				Fields: tt.fields,
				Header: config.ForwardingHeaderConfig{Enabled: true, Encrypt: false},
			}

			forwarder := NewForwarder(cfg)
			headers := make(http.Header)
			result := forwarder.AddToHeaders(headers, tt.userInfo)

			hasUsername := result.Get("X-Forwarded-User") != ""
			hasEmail := result.Get("X-Forwarded-Email") != ""

			if hasUsername != tt.wantUsername {
				t.Errorf("username presence = %v, want %v", hasUsername, tt.wantUsername)
			}
			if hasEmail != tt.wantEmail {
				t.Errorf("email presence = %v, want %v", hasEmail, tt.wantEmail)
			}
		})
	}
}

func TestForwarder_AddToQueryString_PreservesFragment(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields:      []string{"username"},
		QueryString: config.ForwardingMethodConfig{Enabled: true, Encrypt: false},
	}

	forwarder := NewForwarder(cfg)
	userInfo := &UserInfo{Username: "john"}

	result, err := forwarder.AddToQueryString("http://example.com/path#fragment", userInfo)
	if err != nil {
		t.Fatalf("AddToQueryString() error = %v", err)
	}

	// Check that fragment is preserved
	if !strings.Contains(result, "#fragment") {
		t.Errorf("Fragment was not preserved: %v", result)
	}
}
