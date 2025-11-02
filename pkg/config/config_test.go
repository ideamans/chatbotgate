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
