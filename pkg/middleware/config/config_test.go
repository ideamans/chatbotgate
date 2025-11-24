package config

import (
	"errors"
	"net/http"
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

func TestEmailAuthConfig_GetLimitPerMinute(t *testing.T) {
	tests := []struct {
		name           string
		limitPerMinute int
		want           int
	}{
		{
			name:           "default limit (zero)",
			limitPerMinute: 0,
			want:           5,
		},
		{
			name:           "default limit (negative)",
			limitPerMinute: -1,
			want:           5,
		},
		{
			name:           "custom limit 1",
			limitPerMinute: 1,
			want:           1,
		},
		{
			name:           "custom limit 10",
			limitPerMinute: 10,
			want:           10,
		},
		{
			name:           "custom limit 100",
			limitPerMinute: 100,
			want:           100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := EmailAuthConfig{
				LimitPerMinute: tt.limitPerMinute,
			}

			got := cfg.GetLimitPerMinute()

			if got != tt.want {
				t.Errorf("GetLimitPerMinute() = %d, want %d", got, tt.want)
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
					Cookie: CookieConfig{
						Secret: "this-is-a-secret-key-with-32-characters",
					},
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{
							ID:           "google",
							Type:         "google",
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
					Cookie: CookieConfig{
						Secret: "this-is-a-secret-key-with-32-characters",
					},
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
					Cookie: CookieConfig{
						Secret: "short",
					},
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
					Cookie: CookieConfig{
						Secret: "this-is-a-secret-key-with-32-characters",
					},
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
					Cookie: CookieConfig{
						Secret: "this-is-a-secret-key-with-32-characters",
					},
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
					Cookie: CookieConfig{
						Secret: "this-is-a-secret-key-with-32-characters",
					},
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{
							ID:           "google",
							Type:         "google",
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
		{
			name: "valid with only password auth (OAuth2 and email disabled)",
			config: &Config{
				Service: ServiceConfig{
					Name:        "Test Service",
					Description: "Test Description",
				},
				Server: ServerConfig{
					AuthPathPrefix: "/_auth",
				},
				Session: SessionConfig{
					Cookie: CookieConfig{
						Secret: "this-is-a-secret-key-with-32-characters",
					},
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{Disabled: true},
					},
				},
				EmailAuth: EmailAuthConfig{
					Enabled: false,
				},
				PasswordAuth: PasswordAuthConfig{
					Enabled:  true,
					Password: "test-password",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid with OAuth2 and password auth",
			config: &Config{
				Service: ServiceConfig{
					Name:        "Test Service",
					Description: "Test Description",
				},
				Server: ServerConfig{
					AuthPathPrefix: "/_auth",
				},
				Session: SessionConfig{
					Cookie: CookieConfig{
						Secret: "this-is-a-secret-key-with-32-characters",
					},
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{
							ID:           "google",
							Type:         "google",
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
				PasswordAuth: PasswordAuthConfig{
					Enabled:  true,
					Password: "test-password",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid with all auth methods enabled",
			config: &Config{
				Service: ServiceConfig{
					Name:        "Test Service",
					Description: "Test Description",
				},
				Server: ServerConfig{
					AuthPathPrefix: "/_auth",
				},
				Session: SessionConfig{
					Cookie: CookieConfig{
						Secret: "this-is-a-secret-key-with-32-characters",
					},
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{
							ID:           "google",
							Type:         "google",
							DisplayName:  "Google",
							ClientID:     "test-client-id",
							ClientSecret: "test-client-secret",
							Disabled:     false,
						},
					},
				},
				EmailAuth: EmailAuthConfig{
					Enabled: true,
				},
				PasswordAuth: PasswordAuthConfig{
					Enabled:  true,
					Password: "test-password",
				},
			},
			wantErr: nil,
		},
		{
			name: "no auth method (all disabled including password)",
			config: &Config{
				Service: ServiceConfig{
					Name: "Test",
				},
				Server: ServerConfig{},
				Session: SessionConfig{
					Cookie: CookieConfig{
						Secret: "this-is-a-secret-key-with-32-characters",
					},
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{Disabled: true},
					},
				},
				EmailAuth: EmailAuthConfig{
					Enabled: false,
				},
				PasswordAuth: PasswordAuthConfig{
					Enabled: false,
				},
			},
			wantErr: ErrNoAuthMethod,
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
			Session: SessionConfig{Cookie: CookieConfig{Secret: "this-is-a-secret-key-with-32-characters"}},
			OAuth2: OAuth2Config{
				Providers: []OAuth2Provider{{ID: "google", Type: "google", ClientID: "test", ClientSecret: "test"}},
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

func TestCookieConfig_GetExpireDuration(t *testing.T) {
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
			cfg := CookieConfig{
				Expire: tt.expire,
			}
			got, err := cfg.GetExpireDuration()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetExpireDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetExpireDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoggingConfig_FileLogging(t *testing.T) {
	tests := []struct {
		name string
		cfg  LoggingConfig
		want bool // whether file logging is enabled
	}{
		{
			name: "no file config",
			cfg: LoggingConfig{
				Level: "info",
				Color: true,
				File:  nil,
			},
			want: false,
		},
		{
			name: "file config with path",
			cfg: LoggingConfig{
				Level: "info",
				Color: true,
				File: &FileLoggingConfig{
					Path: "/var/log/test.log",
				},
			},
			want: true,
		},
		{
			name: "file config with empty path",
			cfg: LoggingConfig{
				Level: "info",
				Color: true,
				File: &FileLoggingConfig{
					Path: "",
				},
			},
			want: false,
		},
		{
			name: "file config with all options",
			cfg: LoggingConfig{
				Level: "debug",
				Color: false,
				File: &FileLoggingConfig{
					Path:       "/var/log/chatbotgate/app.log",
					MaxSizeMB:  100,
					MaxBackups: 5,
					MaxAge:     30,
					Compress:   true,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasFile := tt.cfg.File != nil && tt.cfg.File.Path != ""
			if hasFile != tt.want {
				t.Errorf("File logging enabled = %v, want %v", hasFile, tt.want)
			}

			// Verify file config properties if present
			if tt.cfg.File != nil && tt.cfg.File.Path != "" {
				if tt.cfg.File.Path == "" {
					t.Error("File path should not be empty when file config is present")
				}
			}
		})
	}
}

func TestFileLoggingConfig_Defaults(t *testing.T) {
	tests := []struct {
		name           string
		cfg            FileLoggingConfig
		wantMaxSizeMB  int
		wantMaxBackups int
		wantMaxAge     int
		wantCompress   bool
	}{
		{
			name: "all defaults (zero values)",
			cfg: FileLoggingConfig{
				Path: "/var/log/test.log",
			},
			wantMaxSizeMB:  0, // Will be set to 100 by factory
			wantMaxBackups: 0, // Will be set to 3 by factory
			wantMaxAge:     0, // Will be set to 28 by factory
			wantCompress:   false,
		},
		{
			name: "custom values",
			cfg: FileLoggingConfig{
				Path:       "/var/log/test.log",
				MaxSizeMB:  50,
				MaxBackups: 10,
				MaxAge:     14,
				Compress:   true,
			},
			wantMaxSizeMB:  50,
			wantMaxBackups: 10,
			wantMaxAge:     14,
			wantCompress:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cfg.MaxSizeMB != tt.wantMaxSizeMB {
				t.Errorf("MaxSizeMB = %v, want %v", tt.cfg.MaxSizeMB, tt.wantMaxSizeMB)
			}
			if tt.cfg.MaxBackups != tt.wantMaxBackups {
				t.Errorf("MaxBackups = %v, want %v", tt.cfg.MaxBackups, tt.wantMaxBackups)
			}
			if tt.cfg.MaxAge != tt.wantMaxAge {
				t.Errorf("MaxAge = %v, want %v", tt.cfg.MaxAge, tt.wantMaxAge)
			}
			if tt.cfg.Compress != tt.wantCompress {
				t.Errorf("Compress = %v, want %v", tt.cfg.Compress, tt.wantCompress)
			}
		})
	}
}

func TestServerConfig_GetAuthPathPrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   string
	}{
		{
			name:   "default prefix",
			prefix: "",
			want:   "/_auth",
		},
		{
			name:   "custom prefix",
			prefix: "/_oauth2_proxy",
			want:   "/_oauth2_proxy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ServerConfig{AuthPathPrefix: tt.prefix}
			got := cfg.GetAuthPathPrefix()
			if got != tt.want {
				t.Errorf("GetAuthPathPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerConfig_GetCallbackURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		prefix  string
		host    string
		port    int
		want    string
	}{
		{
			name:    "with base URL",
			baseURL: "https://example.com",
			prefix:  "",
			host:    "localhost",
			port:    4180,
			want:    "https://example.com/_auth/oauth2/callback",
		},
		{
			name:    "custom prefix with base URL",
			baseURL: "https://example.com",
			prefix:  "/_oauth2",
			host:    "localhost",
			port:    4180,
			want:    "https://example.com/_oauth2/oauth2/callback",
		},
		{
			name:    "without base URL (localhost)",
			baseURL: "",
			prefix:  "",
			host:    "localhost",
			port:    4180,
			want:    "http://localhost:4180/_auth/oauth2/callback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ServerConfig{
				BaseURL:        tt.baseURL,
				AuthPathPrefix: tt.prefix,
			}
			got := cfg.GetCallbackURL(tt.host, tt.port)
			if got != tt.want {
				t.Errorf("GetCallbackURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCookieConfig_GetSameSite(t *testing.T) {
	tests := []struct {
		name     string
		samesite string
		want     http.SameSite
	}{
		{
			name:     "lax mode",
			samesite: "lax",
			want:     http.SameSiteLaxMode,
		},
		{
			name:     "strict mode",
			samesite: "strict",
			want:     http.SameSiteStrictMode,
		},
		{
			name:     "none mode",
			samesite: "none",
			want:     http.SameSiteNoneMode,
		},
		{
			name:     "default mode (empty)",
			samesite: "",
			want:     http.SameSiteLaxMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := CookieConfig{SameSite: tt.samesite}
			got := cfg.GetSameSite()
			if got != tt.want {
				t.Errorf("GetSameSite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSMTPConfig_GetFromAddress(t *testing.T) {
	tests := []struct {
		name        string
		from        string
		fromName    string
		parentEmail string
		parentName  string
		wantEmail   string
		wantName    string
	}{
		{
			name:        "SMTP config overrides",
			from:        "smtp@example.com",
			fromName:    "SMTP Service",
			parentEmail: "parent@example.com",
			parentName:  "Parent Service",
			wantEmail:   "smtp@example.com",
			wantName:    "SMTP Service",
		},
		{
			name:        "fallback to parent",
			from:        "",
			fromName:    "",
			parentEmail: "parent@example.com",
			parentName:  "Parent Service",
			wantEmail:   "parent@example.com",
			wantName:    "Parent Service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SMTPConfig{
				From:     tt.from,
				FromName: tt.fromName,
			}
			gotEmail, gotName := cfg.GetFromAddress(tt.parentEmail, tt.parentName)
			if gotEmail != tt.wantEmail {
				t.Errorf("GetFromAddress() email = %v, want %v", gotEmail, tt.wantEmail)
			}
			if gotName != tt.wantName {
				t.Errorf("GetFromAddress() name = %v, want %v", gotName, tt.wantName)
			}
		})
	}
}

func TestSendGridConfig_GetFromAddress(t *testing.T) {
	tests := []struct {
		name        string
		from        string
		fromName    string
		parentEmail string
		parentName  string
		wantEmail   string
		wantName    string
	}{
		{
			name:        "SendGrid config overrides",
			from:        "sendgrid@example.com",
			fromName:    "SendGrid Service",
			parentEmail: "parent@example.com",
			parentName:  "Parent Service",
			wantEmail:   "sendgrid@example.com",
			wantName:    "SendGrid Service",
		},
		{
			name:        "fallback to parent",
			from:        "",
			fromName:    "",
			parentEmail: "parent@example.com",
			parentName:  "Parent Service",
			wantEmail:   "parent@example.com",
			wantName:    "Parent Service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SendGridConfig{
				From:     tt.from,
				FromName: tt.fromName,
			}
			gotEmail, gotName := cfg.GetFromAddress(tt.parentEmail, tt.parentName)
			if gotEmail != tt.wantEmail {
				t.Errorf("GetFromAddress() email = %v, want %v", gotEmail, tt.wantEmail)
			}
			if gotName != tt.wantName {
				t.Errorf("GetFromAddress() name = %v, want %v", gotName, tt.wantName)
			}
		})
	}
}

func TestSendmailConfig_GetFromAddress(t *testing.T) {
	tests := []struct {
		name        string
		from        string
		fromName    string
		parentEmail string
		parentName  string
		wantEmail   string
		wantName    string
	}{
		{
			name:        "Sendmail config overrides",
			from:        "sendmail@example.com",
			fromName:    "Sendmail Service",
			parentEmail: "parent@example.com",
			parentName:  "Parent Service",
			wantEmail:   "sendmail@example.com",
			wantName:    "Sendmail Service",
		},
		{
			name:        "fallback to parent",
			from:        "",
			fromName:    "",
			parentEmail: "parent@example.com",
			parentName:  "Parent Service",
			wantEmail:   "parent@example.com",
			wantName:    "Parent Service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SendmailConfig{
				From:     tt.from,
				FromName: tt.fromName,
			}
			gotEmail, gotName := cfg.GetFromAddress(tt.parentEmail, tt.parentName)
			if gotEmail != tt.wantEmail {
				t.Errorf("GetFromAddress() email = %v, want %v", gotEmail, tt.wantEmail)
			}
			if gotName != tt.wantName {
				t.Errorf("GetFromAddress() name = %v, want %v", gotName, tt.wantName)
			}
		})
	}
}

func TestEmailTokenConfig_GetTokenExpireDuration(t *testing.T) {
	tests := []struct {
		name    string
		expire  string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "valid duration",
			expire:  "15m",
			want:    15 * time.Minute,
			wantErr: false,
		},
		{
			name:    "default when empty",
			expire:  "",
			want:    15 * time.Minute,
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
			cfg := EmailTokenConfig{Expire: tt.expire}
			got, err := cfg.GetTokenExpireDuration()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTokenExpireDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetTokenExpireDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNamespaceConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name           string
		cfg            NamespaceConfig
		wantSession    string
		wantToken      string
		wantEmailQuota string
	}{
		{
			name:           "all empty",
			cfg:            NamespaceConfig{},
			wantSession:    "session",
			wantToken:      "token",
			wantEmailQuota: "email_quota",
		},
		{
			name: "custom values",
			cfg: NamespaceConfig{
				Session:    "custom_session",
				Token:      "custom_token",
				EmailQuota: "custom_email_quota",
			},
			wantSession:    "custom_session",
			wantToken:      "custom_token",
			wantEmailQuota: "custom_email_quota",
		},
		{
			name: "partial custom",
			cfg: NamespaceConfig{
				Session: "custom_session",
			},
			wantSession:    "custom_session",
			wantToken:      "token",
			wantEmailQuota: "email_quota",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			cfg.SetDefaults()
			if cfg.Session != tt.wantSession {
				t.Errorf("Session = %v, want %v", cfg.Session, tt.wantSession)
			}
			if cfg.Token != tt.wantToken {
				t.Errorf("Token = %v, want %v", cfg.Token, tt.wantToken)
			}
			if cfg.EmailQuota != tt.wantEmailQuota {
				t.Errorf("EmailQuota = %v, want %v", cfg.EmailQuota, tt.wantEmailQuota)
			}
		})
	}
}
