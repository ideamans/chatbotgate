package config

import "time"

// Config represents the application configuration
type Config struct {
	Service       ServiceConfig       `yaml:"service"`
	Server        ServerConfig        `yaml:"server"`
	Proxy         ProxyConfig         `yaml:"proxy"`
	Session       SessionConfig       `yaml:"session"`
	OAuth2        OAuth2Config        `yaml:"oauth2"`
	EmailAuth     EmailAuthConfig     `yaml:"email_auth"`
	Authorization AuthorizationConfig `yaml:"authorization"`
	Logging       LoggingConfig       `yaml:"logging"`
}

// ServiceConfig contains service-level settings
type ServiceConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	AuthPathPrefix string `yaml:"auth_path_prefix"` // Path prefix for authentication endpoints (default: "/_auth")
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
	Upstream string            `yaml:"upstream"` // Default upstream (required)
	Hosts    map[string]string `yaml:"hosts"`    // Host-based routing (optional)
}

// SessionConfig contains session management settings
type SessionConfig struct {
	CookieName     string            `yaml:"cookie_name"`
	CookieSecret   string            `yaml:"cookie_secret"`
	CookieExpire   string            `yaml:"cookie_expire"`
	CookieSecure   bool              `yaml:"cookie_secure"`
	CookieHTTPOnly bool              `yaml:"cookie_httponly"`
	CookieSameSite string            `yaml:"cookie_samesite"`
	StoreType      string            `yaml:"store_type"` // "memory" or "redis" (default: "memory")
	Redis          RedisSessionConfig `yaml:"redis"`      // Redis configuration (used when store_type is "redis")
}

// RedisSessionConfig contains Redis session store settings
type RedisSessionConfig struct {
	Addr     string `yaml:"addr"`     // Redis server address (host:port)
	Password string `yaml:"password"` // Redis password (optional)
	DB       int    `yaml:"db"`       // Redis database number
	Prefix   string `yaml:"prefix"`   // Key prefix for sessions (default: "session:")
}

// GetCookieExpireDuration returns the cookie expiration as a time.Duration
func (s SessionConfig) GetCookieExpireDuration() (time.Duration, error) {
	return time.ParseDuration(s.CookieExpire)
}

// OAuth2Config contains OAuth2 provider settings
type OAuth2Config struct {
	Providers []OAuth2Provider `yaml:"providers"`
}

// OAuth2Provider represents a single OAuth2 provider configuration
type OAuth2Provider struct {
	Name         string `yaml:"name"`
	Type         string `yaml:"type"` // "google", "github", "microsoft", "custom" (optional, defaults to name)
	DisplayName  string `yaml:"display_name"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	Enabled      bool   `yaml:"enabled"`

	// Custom provider settings (only used when Type is "custom")
	AuthURL            string `yaml:"auth_url"`              // Custom authorization endpoint
	TokenURL           string `yaml:"token_url"`             // Custom token endpoint
	UserInfoURL        string `yaml:"userinfo_url"`          // Custom userinfo endpoint
	JWKSURL            string `yaml:"jwks_url"`              // Optional OIDC JWKS URL
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"` // Allow HTTP for testing (default: false)
}

// EmailAuthConfig contains email authentication settings
type EmailAuthConfig struct {
	Enabled       bool             `yaml:"enabled"`
	SenderType    string           `yaml:"sender_type"`     // "smtp" or "sendgrid"
	SMTP          SMTPConfig       `yaml:"smtp"`
	SendGrid      SendGridConfig   `yaml:"sendgrid"`
	Token         EmailTokenConfig `yaml:"token"`
	OTPOutputFile string           `yaml:"otp_output_file"` // Optional: output OTP to file instead of sending email (for E2E testing)
}

// SMTPConfig contains SMTP server settings
type SMTPConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	From      string `yaml:"from"`
	FromName  string `yaml:"from_name"`
	TLS       bool   `yaml:"tls"`
	StartTLS  bool   `yaml:"starttls"`
}

// SendGridConfig contains SendGrid API settings
type SendGridConfig struct {
	APIKey   string `yaml:"api_key"`
	From     string `yaml:"from"`
	FromName string `yaml:"from_name"`
}

// EmailTokenConfig contains token expiration settings
type EmailTokenConfig struct {
	Expire string `yaml:"expire"`
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
	AllowedEmails  []string `yaml:"allowed_emails"`
	AllowedDomains []string `yaml:"allowed_domains"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level       string `yaml:"level"`
	ModuleLevel string `yaml:"module_level"`
	Color       bool   `yaml:"color"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Service.Name == "" {
		return ErrServiceNameRequired
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return ErrInvalidPort
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
