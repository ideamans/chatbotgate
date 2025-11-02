package config

import "time"

// Config represents the application configuration
type Config struct {
	Service       ServiceConfig       `yaml:"service" json:"service"`
	Server        ServerConfig        `yaml:"server" json:"server"`
	Proxy         ProxyConfig         `yaml:"proxy" json:"proxy"`
	Session       SessionConfig       `yaml:"session" json:"session"`
	OAuth2        OAuth2Config        `yaml:"oauth2" json:"oauth2"`
	EmailAuth     EmailAuthConfig     `yaml:"email_auth" json:"email_auth"`
	Authorization AuthorizationConfig `yaml:"authorization" json:"authorization"`
	Logging       LoggingConfig       `yaml:"logging" json:"logging"`
}

// ServiceConfig contains service-level settings
type ServiceConfig struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	IconURL     string `yaml:"icon_url" json:"icon_url"`     // Icon URL for auth header (48px icon)
	LogoURL     string `yaml:"logo_url" json:"logo_url"`     // Logo URL for auth header (larger logo image)
	LogoWidth   string `yaml:"logo_width" json:"logo_width"`   // Logo width (e.g., "100px", "150px", "200px", default: "200px")
}

// ServerConfig contains authentication server settings
type ServerConfig struct {
	AuthPathPrefix string `yaml:"auth_path_prefix" json:"auth_path_prefix"` // Path prefix for authentication endpoints (default: "/_auth")
	CallbackURL    string `yaml:"callback_url" json:"callback_url"`         // Optional: Override OAuth2 callback URL (useful when behind reverse proxy or different external port)
	BaseURL        string `yaml:"base_url" json:"base_url"`                 // Optional: Override base URL for email links and redirects (e.g., "http://localhost:4181")
}

// GetAuthPathPrefix returns the authentication path prefix
// If not set, returns the default "/_auth"
func (s ServerConfig) GetAuthPathPrefix() string {
	if s.AuthPathPrefix == "" {
		return "/_auth"
	}
	return s.AuthPathPrefix
}

// ProxyConfig contains proxy settings
type ProxyConfig struct {
	Upstream string            `yaml:"upstream" json:"upstream"` // Default upstream (required)
	Hosts    map[string]string `yaml:"hosts" json:"hosts"`    // Host-based routing (optional)
}

// SessionConfig contains session management settings
type SessionConfig struct {
	CookieName     string            `yaml:"cookie_name" json:"cookie_name"`
	CookieSecret   string            `yaml:"cookie_secret" json:"cookie_secret"`
	CookieExpire   string            `yaml:"cookie_expire" json:"cookie_expire"`
	CookieSecure   bool              `yaml:"cookie_secure" json:"cookie_secure"`
	CookieHTTPOnly bool              `yaml:"cookie_httponly" json:"cookie_httponly"`
	CookieSameSite string            `yaml:"cookie_samesite" json:"cookie_samesite"`
	StoreType      string            `yaml:"store_type" json:"store_type"` // "memory" or "redis" (default: "memory")
	Redis          RedisSessionConfig `yaml:"redis" json:"redis"`      // Redis configuration (used when store_type is "redis")
}

// RedisSessionConfig contains Redis session store settings
type RedisSessionConfig struct {
	Addr     string `yaml:"addr" json:"addr"`     // Redis server address (host:port)
	Password string `yaml:"password" json:"password"` // Redis password (optional)
	DB       int    `yaml:"db" json:"db"`       // Redis database number
	Prefix   string `yaml:"prefix" json:"prefix"`   // Key prefix for sessions (default: "session:")
}

// GetCookieExpireDuration returns the cookie expiration as a time.Duration
func (s SessionConfig) GetCookieExpireDuration() (time.Duration, error) {
	return time.ParseDuration(s.CookieExpire)
}

// OAuth2Config contains OAuth2 provider settings
type OAuth2Config struct {
	Providers []OAuth2Provider `yaml:"providers" json:"providers"`
}

// OAuth2Provider represents a single OAuth2 provider configuration
type OAuth2Provider struct {
	Name         string `yaml:"name" json:"name"`
	Type         string `yaml:"type" json:"type"` // "google", "github", "microsoft", "custom" (optional, defaults to name)
	DisplayName  string `yaml:"display_name" json:"display_name"`
	ClientID     string `yaml:"client_id" json:"client_id"`
	ClientSecret string `yaml:"client_secret" json:"client_secret"`
	Enabled      bool   `yaml:"enabled" json:"enabled"`
	IconURL      string `yaml:"icon_url" json:"icon_url"` // Optional custom icon URL (if not set, uses default icon based on provider type)

	// Custom provider settings (only used when Type is "custom")
	AuthURL            string `yaml:"auth_url" json:"auth_url"`              // Custom authorization endpoint
	TokenURL           string `yaml:"token_url" json:"token_url"`             // Custom token endpoint
	UserInfoURL        string `yaml:"userinfo_url" json:"userinfo_url"`          // Custom userinfo endpoint
	JWKSURL            string `yaml:"jwks_url" json:"jwks_url"`              // Optional OIDC JWKS URL
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify" json:"insecure_skip_verify"` // Allow HTTP for testing (default: false)
}

// EmailAuthConfig contains email authentication settings
type EmailAuthConfig struct {
	Enabled       bool             `yaml:"enabled" json:"enabled"`
	SenderType    string           `yaml:"sender_type" json:"sender_type"`     // "smtp" or "sendgrid"
	SMTP          SMTPConfig       `yaml:"smtp" json:"smtp"`
	SendGrid      SendGridConfig   `yaml:"sendgrid" json:"sendgrid"`
	Token         EmailTokenConfig `yaml:"token" json:"token"`
	OTPOutputFile string           `yaml:"otp_output_file" json:"otp_output_file"` // Optional: output OTP to file instead of sending email (for E2E testing)
}

// SMTPConfig contains SMTP server settings
type SMTPConfig struct {
	Host      string `yaml:"host" json:"host"`
	Port      int    `yaml:"port" json:"port"`
	Username  string `yaml:"username" json:"username"`
	Password  string `yaml:"password" json:"password"`
	From      string `yaml:"from" json:"from"`
	FromName  string `yaml:"from_name" json:"from_name"`
	TLS       bool   `yaml:"tls" json:"tls"`
	StartTLS  bool   `yaml:"starttls" json:"starttls"`
}

// SendGridConfig contains SendGrid API settings
type SendGridConfig struct {
	APIKey   string `yaml:"api_key" json:"api_key"`
	From     string `yaml:"from" json:"from"`
	FromName string `yaml:"from_name" json:"from_name"`
}

// EmailTokenConfig contains token expiration settings
type EmailTokenConfig struct {
	Expire string `yaml:"expire" json:"expire"`
}

// GetTokenExpireDuration returns the token expiration as a time.Duration
func (e EmailTokenConfig) GetTokenExpireDuration() (time.Duration, error) {
	if e.Expire == "" {
		return 15 * time.Minute, nil // Default 15 minutes
	}
	return time.ParseDuration(e.Expire)
}

// AuthorizationConfig contains authorization settings
type AuthorizationConfig struct {
	Allowed []string `yaml:"allowed" json:"allowed"` // Email addresses or domains (domain starts with @)
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level       string `yaml:"level" json:"level"`
	ModuleLevel string `yaml:"module_level" json:"module_level"`
	Color       bool   `yaml:"color" json:"color"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Service.Name == "" {
		return ErrServiceNameRequired
	}

	if c.Proxy.Upstream == "" {
		return ErrUpstreamRequired
	}

	if c.Session.CookieSecret == "" {
		return ErrCookieSecretRequired
	}

	if len(c.Session.CookieSecret) < 32 {
		return ErrCookieSecretTooShort
	}

	// Check at least one OAuth2 provider is enabled
	hasEnabledProvider := false
	for _, p := range c.OAuth2.Providers {
		if p.Enabled {
			hasEnabledProvider = true
			break
		}
	}
	if !hasEnabledProvider {
		return ErrNoEnabledProviders
	}

	return nil
}
