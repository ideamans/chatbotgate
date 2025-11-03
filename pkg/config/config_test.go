package config

import (
	"testing"
	"time"
)

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
				Proxy: ProxyConfig{
					Upstream: "http://localhost:8080",
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
							Enabled:      true,
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
				Proxy: ProxyConfig{
					Upstream: "http://localhost:8080",
				},
				Session: SessionConfig{
					CookieSecret: "this-is-a-secret-key-with-32-characters",
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{Enabled: true},
					},
				},
			},
			wantErr: ErrServiceNameRequired,
		},
		{
			name: "missing upstream",
			config: &Config{
				Service: ServiceConfig{
					Name: "Test",
				},
				Server: ServerConfig{},
				Proxy: ProxyConfig{
					Upstream: "",
				},
				Session: SessionConfig{
					CookieSecret: "this-is-a-secret-key-with-32-characters",
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{Enabled: true},
					},
				},
			},
			wantErr: ErrUpstreamRequired,
		},
		{
			name: "cookie secret too short",
			config: &Config{
				Service: ServiceConfig{
					Name: "Test",
				},
				Server: ServerConfig{},
				Proxy: ProxyConfig{
					Upstream: "http://localhost:8080",
				},
				Session: SessionConfig{
					CookieSecret: "short",
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{Enabled: true},
					},
				},
			},
			wantErr: ErrCookieSecretTooShort,
		},
		{
			name: "no enabled providers",
			config: &Config{
				Service: ServiceConfig{
					Name: "Test",
				},
				Server: ServerConfig{},
				Proxy: ProxyConfig{
					Upstream: "http://localhost:8080",
				},
				Session: SessionConfig{
					CookieSecret: "this-is-a-secret-key-with-32-characters",
				},
				OAuth2: OAuth2Config{
					Providers: []OAuth2Provider{
						{Enabled: false},
					},
				},
			},
			wantErr: ErrNoEnabledProviders,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_ValidateForwarding(t *testing.T) {
	baseConfig := func() *Config {
		return &Config{
			Service: ServiceConfig{Name: "Test Service"},
			Proxy:   ProxyConfig{Upstream: "http://localhost:8080"},
			Session: SessionConfig{CookieSecret: "this-is-a-secret-key-with-32-characters"},
			OAuth2: OAuth2Config{
				Providers: []OAuth2Provider{{Enabled: true}},
			},
		}
	}

	tests := []struct {
		name       string
		forwarding ForwardingConfig
		wantErr    error
	}{
		{
			name: "valid querystring forwarding without encryption",
			forwarding: ForwardingConfig{
				Fields:      []string{"username", "email"},
				QueryString: ForwardingMethodConfig{Enabled: true, Encrypt: false},
			},
			wantErr: nil,
		},
		{
			name: "valid header forwarding without encryption",
			forwarding: ForwardingConfig{
				Fields: []string{"username", "email"},
				Header: ForwardingHeaderConfig{Enabled: true, Encrypt: false, Prefix: "X-Custom-"},
			},
			wantErr: nil,
		},
		{
			name: "valid forwarding with encryption",
			forwarding: ForwardingConfig{
				Fields:      []string{"username", "email"},
				QueryString: ForwardingMethodConfig{Enabled: true, Encrypt: true},
				Header:      ForwardingHeaderConfig{Enabled: true, Encrypt: true},
				Encryption:  EncryptionConfig{Key: "this-is-a-32-character-encryption-key", Algorithm: "aes-256-gcm"},
			},
			wantErr: nil,
		},
		{
			name: "forwarding disabled - no validation",
			forwarding: ForwardingConfig{
				Fields:      []string{}, // Empty fields should be OK when forwarding is disabled
				QueryString: ForwardingMethodConfig{Enabled: false},
				Header:      ForwardingHeaderConfig{Enabled: false},
			},
			wantErr: nil,
		},
		{
			name: "missing fields when forwarding enabled",
			forwarding: ForwardingConfig{
				Fields:      []string{},
				QueryString: ForwardingMethodConfig{Enabled: true},
			},
			wantErr: ErrForwardingFieldsRequired,
		},
		{
			name: "invalid field name",
			forwarding: ForwardingConfig{
				Fields:      []string{"username", "invalid_field"},
				QueryString: ForwardingMethodConfig{Enabled: true},
			},
			wantErr: ErrInvalidForwardingField,
		},
		{
			name: "encryption enabled but key missing",
			forwarding: ForwardingConfig{
				Fields:      []string{"username"},
				QueryString: ForwardingMethodConfig{Enabled: true, Encrypt: true},
				Encryption:  EncryptionConfig{Key: ""},
			},
			wantErr: ErrEncryptionKeyRequired,
		},
		{
			name: "encryption key too short",
			forwarding: ForwardingConfig{
				Fields:      []string{"username"},
				Header:      ForwardingHeaderConfig{Enabled: true, Encrypt: true},
				Encryption:  EncryptionConfig{Key: "short-key"},
			},
			wantErr: ErrEncryptionKeyTooShort,
		},
		{
			name: "only username field",
			forwarding: ForwardingConfig{
				Fields:      []string{"username"},
				QueryString: ForwardingMethodConfig{Enabled: true},
			},
			wantErr: nil,
		},
		{
			name: "only email field",
			forwarding: ForwardingConfig{
				Fields: []string{"email"},
				Header: ForwardingHeaderConfig{Enabled: true},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseConfig()
			cfg.Forwarding = tt.forwarding
			err := cfg.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestForwardingHeaderConfig_GetPrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   string
	}{
		{
			name:   "default prefix",
			prefix: "",
			want:   "X-Chatbotgate-",
		},
		{
			name:   "custom prefix",
			prefix: "X-Custom-",
			want:   "X-Custom-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ForwardingHeaderConfig{Prefix: tt.prefix}
			got := cfg.GetPrefix()
			if got != tt.want {
				t.Errorf("GetPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
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
