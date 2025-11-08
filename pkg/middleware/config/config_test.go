package config

import (
	"errors"
	"testing"
	"time"
)

func TestEmailAuthConfig_GetFromAddress(t *testing.T) {
	tests := []struct {
		name         string
		from         string
		fromName     string
		wantEmail    string
		wantFromName string
	}{
		{
			name:         "RFC 5322 format with name",
			from:         "ChatbotGate <noreply@example.com>",
			fromName:     "",
			wantEmail:    "noreply@example.com",
			wantFromName: "ChatbotGate",
		},
		{
			name:         "RFC 5322 format with quoted name",
			from:         `"Chat Bot Gate" <noreply@example.com>`,
			fromName:     "",
			wantEmail:    "noreply@example.com",
			wantFromName: "Chat Bot Gate",
		},
		{
			name:         "Plain email with separate from_name",
			from:         "noreply@example.com",
			fromName:     "ChatbotGate",
			wantEmail:    "noreply@example.com",
			wantFromName: "ChatbotGate",
		},
		{
			name:         "Plain email without from_name",
			from:         "noreply@example.com",
			fromName:     "",
			wantEmail:    "noreply@example.com",
			wantFromName: "",
		},
		{
			name:         "Empty from",
			from:         "",
			fromName:     "ChatbotGate",
			wantEmail:    "",
			wantFromName: "",
		},
		{
			name:         "RFC 5322 with spaces",
			from:         "  ChatbotGate  <  noreply@example.com  >  ",
			fromName:     "",
			wantEmail:    "noreply@example.com",
			wantFromName: "ChatbotGate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := EmailAuthConfig{
				From:     tt.from,
				FromName: tt.fromName,
			}

			gotEmail, gotName := cfg.GetFromAddress()

			if gotEmail != tt.wantEmail {
				t.Errorf("GetFromAddress() email = %q, want %q", gotEmail, tt.wantEmail)
			}

			if gotName != tt.wantFromName {
				t.Errorf("GetFromAddress() name = %q, want %q", gotName, tt.wantFromName)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{
			name: "valid configuration",
			config: &Config{
				Service: ServiceConfig{
					Name:        "Test Service",
					Description: "Test Description",
				},
				Server: ServerConfig{
					AuthPathPrefix: "/_auth",
				},
				Session: SessionConfig{
					CookieSecret: "this-is-a-secret-key-with-32-characters",
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{
							Name:         "google",
							DisplayName:  "Google",
							ClientID:     "test-client-id",
							ClientSecret: "test-client-secret",
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "missing service name",
			config: &Config{
				Service: ServiceConfig{
					Name: "",
				},
				Server: ServerConfig{},
				Session: SessionConfig{
					CookieSecret: "this-is-a-secret-key-with-32-characters",
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{},
					},
				},
			},
			wantErr: ErrServiceNameRequired,
		},
		{
			name: "cookie secret too short",
			config: &Config{
				Service: ServiceConfig{
					Name: "Test",
				},
				Server: ServerConfig{},
				Session: SessionConfig{
					CookieSecret: "short",
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{},
					},
				},
			},
			wantErr: ErrCookieSecretTooShort,
		},
		{
			name: "no auth method (OAuth2 disabled, email disabled)",
			config: &Config{
				Service: ServiceConfig{
					Name: "Test",
				},
				Server: ServerConfig{},
				Session: SessionConfig{
					CookieSecret: "this-is-a-secret-key-with-32-characters",
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{Disabled: true},
					},
				},
				EmailAuth: EmailAuthConfig{
					Enabled: false,
				},
			},
			wantErr: ErrNoAuthMethod,
		},
		{
			name: "valid with only email auth (OAuth2 disabled)",
			config: &Config{
				Service: ServiceConfig{
					Name:        "Test Service",
					Description: "Test Description",
				},
				Server: ServerConfig{
					AuthPathPrefix: "/_auth",
				},
				Session: SessionConfig{
					CookieSecret: "this-is-a-secret-key-with-32-characters",
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{Disabled: true},
					},
				},
				EmailAuth: EmailAuthConfig{
					Enabled: true,
				},
			},
			wantErr: nil,
		},
		{
			name: "valid with only OAuth2 (email disabled)",
			config: &Config{
				Service: ServiceConfig{
					Name:        "Test Service",
					Description: "Test Description",
				},
				Server: ServerConfig{
					AuthPathPrefix: "/_auth",
				},
				Session: SessionConfig{
					CookieSecret: "this-is-a-secret-key-with-32-characters",
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{
							Name:         "google",
							DisplayName:  "Google",
							ClientID:     "test-client-id",
							ClientSecret: "test-client-secret",
							Disabled:     false,
						},
					},
				},
				EmailAuth: EmailAuthConfig{
					Enabled: false,
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_ValidateForwarding(t *testing.T) {
	baseConfig := func() *Config {
		return &Config{
			Service: ServiceConfig{Name: "Test Service"},
			Session: SessionConfig{CookieSecret: "this-is-a-secret-key-with-32-characters"},
			OAuth2: OAuth2Config{
				Providers: []OAuth2Provider{{Name: "google", ClientID: "test", ClientSecret: "test"}},
			},
		}
	}

	tests := []struct {
		name       string
		forwarding ForwardingConfig
		wantErr    error
	}{
		{
			name: "valid plain text field",
			forwarding: ForwardingConfig{
				Fields: []ForwardingField{
					{Path: "email", Query: "email"},
				},
			},
			wantErr: nil,
		},
		{
			name: "valid header forwarding",
			forwarding: ForwardingConfig{
				Fields: []ForwardingField{
					{Path: "username", Header: "X-User"},
				},
			},
			wantErr: nil,
		},
		{
			name: "valid both query and header",
			forwarding: ForwardingConfig{
				Fields: []ForwardingField{
					{Path: "email", Query: "email", Header: "X-Email"},
				},
			},
			wantErr: nil,
		},
		{
			name: "valid with encrypt filter",
			forwarding: ForwardingConfig{
				Encryption: &EncryptionConfig{
					Key:       "this-is-a-32-character-encryption-key",
					Algorithm: "aes-256-gcm",
				},
				Fields: []ForwardingField{
					{Path: "email", Header: "X-Email", Filters: []string{"encrypt"}},
				},
			},
			wantErr: nil,
		},
		{
			name: "valid with multiple filters",
			forwarding: ForwardingConfig{
				Encryption: &EncryptionConfig{
					Key: "this-is-a-32-character-encryption-key",
				},
				Fields: []ForwardingField{
					{Path: "email", Query: "email", Filters: []string{"encrypt", "zip"}},
				},
			},
			wantErr: nil,
		},
		{
			name: "valid entire object",
			forwarding: ForwardingConfig{
				Fields: []ForwardingField{
					{Path: ".", Query: "userinfo"},
				},
			},
			wantErr: nil,
		},
		{
			name: "valid nested path",
			forwarding: ForwardingConfig{
				Fields: []ForwardingField{
					{Path: "extra.avatar_url", Header: "X-Avatar"},
				},
			},
			wantErr: nil,
		},
		{
			name: "missing path",
			forwarding: ForwardingConfig{
				Fields: []ForwardingField{
					{Query: "test"},
				},
			},
			wantErr: errors.New("path is required"),
		},
		{
			name: "missing query and header",
			forwarding: ForwardingConfig{
				Fields: []ForwardingField{
					{Path: "email"},
				},
			},
			wantErr: errors.New("at least one of 'query' or 'header' must be specified"),
		},
		{
			name: "encrypt filter without encryption config",
			forwarding: ForwardingConfig{
				Fields: []ForwardingField{
					{Path: "email", Query: "email", Filters: []string{"encrypt"}},
				},
			},
			wantErr: ErrEncryptionConfigRequired,
		},
		{
			name: "invalid filter name",
			forwarding: ForwardingConfig{
				Fields: []ForwardingField{
					{Path: "email", Query: "email", Filters: []string{"invalid"}},
				},
			},
			wantErr: errors.New("invalid filter"),
		},
		{
			name: "encryption key too short",
			forwarding: ForwardingConfig{
				Encryption: &EncryptionConfig{
					Key: "short",
				},
				Fields: []ForwardingField{
					{Path: "email", Query: "email", Filters: []string{"encrypt"}},
				},
			},
			wantErr: ErrEncryptionKeyTooShort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseConfig()
			cfg.Forwarding = tt.forwarding
			err := cfg.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error containing %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) && !containsError(err.Error(), tt.wantErr.Error()) {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

// Helper function to check if error message contains expected text
func containsError(got, want string) bool {
	return len(want) > 0 && len(got) >= len(want) &&
		(got == want || containsSubstring(got, want))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestEncryptionConfig_GetAlgorithm(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
		want      string
	}{
		{
			name:      "default algorithm",
			algorithm: "",
			want:      "aes-256-gcm",
		},
		{
			name:      "custom algorithm",
			algorithm: "aes-128-gcm",
			want:      "aes-128-gcm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := EncryptionConfig{Algorithm: tt.algorithm}
			got := cfg.GetAlgorithm()
			if got != tt.want {
				t.Errorf("GetAlgorithm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSessionConfig_GetCookieExpireDuration(t *testing.T) {
	tests := []struct {
		name    string
		expire  string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "valid duration",
			expire:  "168h",
			want:    168 * time.Hour,
			wantErr: false,
		},
		{
			name:    "invalid duration",
			expire:  "invalid",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SessionConfig{
				CookieExpire: tt.expire,
			}
			got, err := cfg.GetCookieExpireDuration()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCookieExpireDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetCookieExpireDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
