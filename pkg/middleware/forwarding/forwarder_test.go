package forwarding

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
)

func TestForwarder_AddToQueryString_Disabled(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields:      []string{"username", "email"},
		QueryString: config.ForwardingMethodConfig{Enabled: false},
	}

	forwarder := NewForwarder(cfg, nil)
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

	forwarder := NewForwarder(cfg, nil)
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

	forwarder := NewForwarder(cfg, nil)
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

			forwarder := NewForwarder(cfg, nil)
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

	forwarder := NewForwarder(cfg, nil)
	userInfo := &UserInfo{Username: "john", Email: "john@example.com"}

	headers := make(http.Header)
	headers.Set("Existing-Header", "value")

	result := forwarder.AddToHeaders(headers, userInfo)

	// Should only have existing header
	if result.Get("Existing-Header") != "value" {
		t.Error("Existing header was modified")
	}
	if result.Get("X-ChatbotGate-User") != "" {
		t.Error("User header should not be added when disabled")
	}
	if result.Get("X-ChatbotGate-Email") != "" {
		t.Error("Email header should not be added when disabled")
	}
}

func TestForwarder_AddToHeaders_PlainText(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields: []string{"username", "email"},
		Header: config.ForwardingHeaderConfig{Enabled: true, Encrypt: false},
	}

	forwarder := NewForwarder(cfg, nil)
	userInfo := &UserInfo{Username: "john", Email: "john@example.com"}

	headers := make(http.Header)
	result := forwarder.AddToHeaders(headers, userInfo)

	if result.Get("X-ChatbotGate-User") != "john" {
		t.Errorf("X-ChatbotGate-User header = %v, want %v", result.Get("X-ChatbotGate-User"), "john")
	}
	if result.Get("X-ChatbotGate-Email") != "john@example.com" {
		t.Errorf("X-ChatbotGate-Email header = %v, want %v", result.Get("X-ChatbotGate-Email"), "john@example.com")
	}
}

func TestForwarder_AddToHeaders_Encrypted(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Fields: []string{"username", "email"},
		Header: config.ForwardingHeaderConfig{Enabled: true, Encrypt: true},
		Encryption: config.EncryptionConfig{Key: "this-is-a-32-character-encryption-key"},
	}

	forwarder := NewForwarder(cfg, nil)
	userInfo := &UserInfo{Username: "john", Email: "john@example.com"}

	headers := make(http.Header)
	result := forwarder.AddToHeaders(headers, userInfo)

	// Check that individual encrypted headers exist
	encryptedUser := result.Get("X-ChatbotGate-User")
	if encryptedUser == "" {
		t.Error("X-ChatbotGate-User header not found")
	}

	encryptedEmail := result.Get("X-ChatbotGate-Email")
	if encryptedEmail == "" {
		t.Error("X-ChatbotGate-Email header not found")
	}

	// Verify we can decrypt individual values
	decryptedUser, err := forwarder.encryptor.Decrypt(encryptedUser)
	if err != nil {
		t.Fatalf("Failed to decrypt X-ChatbotGate-User header: %v", err)
	}
	if decryptedUser != "john" {
		t.Errorf("decrypted username = %v, want %v", decryptedUser, "john")
	}

	decryptedEmail, err := forwarder.encryptor.Decrypt(encryptedEmail)
	if err != nil {
		t.Fatalf("Failed to decrypt X-ChatbotGate-Email header: %v", err)
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

			forwarder := NewForwarder(cfg, nil)
			headers := make(http.Header)
			result := forwarder.AddToHeaders(headers, tt.userInfo)

			hasUsername := result.Get("X-ChatbotGate-User") != ""
			hasEmail := result.Get("X-ChatbotGate-Email") != ""

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

	forwarder := NewForwarder(cfg, nil)
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

func TestForwarder_CustomFields_QueryString_PlainText(t *testing.T) {
	cfg := &config.ForwardingConfig{
		QueryString: config.ForwardingMethodConfig{Enabled: true, Encrypt: false},
	}
	providers := []config.OAuth2Provider{
		{
			Name: "test-provider",
			Forwarding: &config.ProviderForwardingConfig{
				Custom: []config.CustomFieldForwarding{
					{Path: "secrets.access_token", Query: "access_token"},
					{Path: "analytics.user_id", Query: "user_id"},
					{Path: "nonexistent.field", Query: "should_not_exist"},
				},
			},
		},
	}

	forwarder := NewForwarder(cfg, providers)
	userInfo := &UserInfo{
		Provider: "test-provider",
		Extra: map[string]interface{}{
			"secrets": map[string]interface{}{
				"access_token": "secret-token-123",
			},
			"analytics": map[string]interface{}{
				"user_id": "user-456",
			},
		},
	}

	result, err := forwarder.AddToQueryString("http://example.com/path", userInfo)
	if err != nil {
		t.Fatalf("AddToQueryString() error = %v", err)
	}

	u, _ := url.Parse(result)
	if u.Query().Get("access_token") != "secret-token-123" {
		t.Errorf("access_token = %v, want %v", u.Query().Get("access_token"), "secret-token-123")
	}
	if u.Query().Get("user_id") != "user-456" {
		t.Errorf("user_id = %v, want %v", u.Query().Get("user_id"), "user-456")
	}
	if u.Query().Get("should_not_exist") != "" {
		t.Errorf("should_not_exist should be empty, got %v", u.Query().Get("should_not_exist"))
	}
}

func TestForwarder_CustomFields_QueryString_Encrypted(t *testing.T) {
	cfg := &config.ForwardingConfig{
		QueryString: config.ForwardingMethodConfig{Enabled: true, Encrypt: true},
		Encryption:  config.EncryptionConfig{Key: "this-is-a-32-character-encryption-key"},
	}
	providers := []config.OAuth2Provider{
		{
			Name: "test-provider",
			Forwarding: &config.ProviderForwardingConfig{
				Custom: []config.CustomFieldForwarding{
					{Path: "secrets.access_token", Query: "access_token"},
				},
			},
		},
	}

	forwarder := NewForwarder(cfg, providers)
	userInfo := &UserInfo{
		Provider: "test-provider",
		Extra: map[string]interface{}{
			"secrets": map[string]interface{}{
				"access_token": "secret-token-123",
			},
		},
	}

	result, err := forwarder.AddToQueryString("http://example.com/path", userInfo)
	if err != nil {
		t.Fatalf("AddToQueryString() error = %v", err)
	}

	u, _ := url.Parse(result)
	encryptedToken := u.Query().Get("access_token")
	if encryptedToken == "" {
		t.Fatal("access_token parameter not found")
	}
	if encryptedToken == "secret-token-123" {
		t.Error("access_token should be encrypted, but got plain text")
	}

	// Verify decryption
	decrypted, err := forwarder.encryptor.Decrypt(encryptedToken)
	if err != nil {
		t.Fatalf("Failed to decrypt access_token: %v", err)
	}
	if decrypted != "secret-token-123" {
		t.Errorf("decrypted access_token = %v, want %v", decrypted, "secret-token-123")
	}
}

func TestForwarder_CustomFields_Headers_PlainText(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Header: config.ForwardingHeaderConfig{Enabled: true, Encrypt: false},
	}
	providers := []config.OAuth2Provider{
		{
			Name: "test-provider",
			Forwarding: &config.ProviderForwardingConfig{
				Custom: []config.CustomFieldForwarding{
					{Path: "secrets.access_token", Header: "X-Access-Token"},
					{Path: "analytics.user_id", Header: "X-Analytics-User-Id"},
					{Path: "nonexistent.field", Header: "X-Should-Not-Exist"},
				},
			},
		},
	}

	forwarder := NewForwarder(cfg, providers)
	userInfo := &UserInfo{
		Provider: "test-provider",
		Extra: map[string]interface{}{
			"secrets": map[string]interface{}{
				"access_token": "secret-token-123",
			},
			"analytics": map[string]interface{}{
				"user_id": "user-456",
			},
		},
	}

	headers := make(http.Header)
	result := forwarder.AddToHeaders(headers, userInfo)

	if result.Get("X-Access-Token") != "secret-token-123" {
		t.Errorf("X-Access-Token = %v, want %v", result.Get("X-Access-Token"), "secret-token-123")
	}
	if result.Get("X-Analytics-User-Id") != "user-456" {
		t.Errorf("X-Analytics-User-Id = %v, want %v", result.Get("X-Analytics-User-Id"), "user-456")
	}
	if result.Get("X-Should-Not-Exist") != "" {
		t.Errorf("X-Should-Not-Exist should be empty, got %v", result.Get("X-Should-Not-Exist"))
	}
}

func TestForwarder_CustomFields_Headers_Encrypted(t *testing.T) {
	cfg := &config.ForwardingConfig{
		Header:     config.ForwardingHeaderConfig{Enabled: true, Encrypt: true},
		Encryption: config.EncryptionConfig{Key: "this-is-a-32-character-encryption-key"},
	}
	providers := []config.OAuth2Provider{
		{
			Name: "test-provider",
			Forwarding: &config.ProviderForwardingConfig{
				Custom: []config.CustomFieldForwarding{
					{Path: "secrets.access_token", Header: "X-Access-Token"},
				},
			},
		},
	}

	forwarder := NewForwarder(cfg, providers)
	userInfo := &UserInfo{
		Provider: "test-provider",
		Extra: map[string]interface{}{
			"secrets": map[string]interface{}{
				"access_token": "secret-token-123",
			},
		},
	}

	headers := make(http.Header)
	result := forwarder.AddToHeaders(headers, userInfo)

	encryptedToken := result.Get("X-Access-Token")
	if encryptedToken == "" {
		t.Fatal("X-Access-Token header not found")
	}
	if encryptedToken == "secret-token-123" {
		t.Error("X-Access-Token should be encrypted, but got plain text")
	}

	// Verify decryption
	decrypted, err := forwarder.encryptor.Decrypt(encryptedToken)
	if err != nil {
		t.Fatalf("Failed to decrypt X-Access-Token: %v", err)
	}
	if decrypted != "secret-token-123" {
		t.Errorf("decrypted X-Access-Token = %v, want %v", decrypted, "secret-token-123")
	}
}
