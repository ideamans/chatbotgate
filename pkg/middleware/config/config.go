package config

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/rules"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
)

// Config represents the application configuration
type Config struct {
	Service       ServiceConfig       `yaml:"service" json:"service"`
	Server        ServerConfig        `yaml:"server" json:"server"`
	Session       SessionConfig       `yaml:"session" json:"session"`
	OAuth2        OAuth2Config        `yaml:"oauth2" json:"oauth2"`
	EmailAuth     EmailAuthConfig     `yaml:"email_auth" json:"email_auth"`
	PasswordAuth  PasswordAuthConfig  `yaml:"password_auth" json:"password_auth"`
	AccessControl AccessControlConfig `yaml:"access_control" json:"access_control"`
	Logging       LoggingConfig       `yaml:"logging" json:"logging"`
	KVS           KVSConfig           `yaml:"kvs" json:"kvs"`               // KVS storage configuration
	Forwarding    ForwardingConfig    `yaml:"forwarding" json:"forwarding"` // User info forwarding configuration
	Assets        AssetsConfig        `yaml:"assets" json:"assets"`         // Assets configuration
}

// ServiceConfig contains service-level settings
type ServiceConfig struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	IconURL     string `yaml:"icon_url" json:"icon_url"`     // Icon URL for auth header (48px icon)
	LogoURL     string `yaml:"logo_url" json:"logo_url"`     // Logo URL for auth header (larger logo image)
	LogoWidth   string `yaml:"logo_width" json:"logo_width"` // Logo width (e.g., "100px", "150px", "200px", default: "200px")
}

// ServerConfig contains authentication server settings
type ServerConfig struct {
	AuthPathPrefix string `yaml:"auth_path_prefix" json:"auth_path_prefix"` // Path prefix for authentication endpoints (default: "/_auth")
	BaseURL        string `yaml:"base_url" json:"base_url"`                 // Optional: Base URL for email links and OAuth2 callback (e.g., "https://example.com:8443" or "http://localhost:4181")
	Development    bool   `yaml:"development" json:"development"`           // Enable development mode (relaxes CSP for inline scripts, default: false)
}

// GetAuthPathPrefix returns the authentication path prefix
// If not set, returns the default "/_auth"
func (s ServerConfig) GetAuthPathPrefix() string {
	if s.AuthPathPrefix == "" {
		return "/_auth"
	}
	return s.AuthPathPrefix
}

// GetCallbackURL returns the OAuth2 callback URL
// Automatically generated from BaseURL and AuthPathPrefix
// Format: {base_url}{auth_path_prefix}/oauth2/callback
// If BaseURL is not set, defaults to http://host:port
func (s ServerConfig) GetCallbackURL(host string, port int) string {
	baseURL := s.BaseURL
	if baseURL == "" {
		// Auto-generate from host and port (defaults to HTTP)
		// For HTTPS, set base_url explicitly (e.g., "https://example.com:8443")
		baseURL = fmt.Sprintf("http://%s:%d", host, port)
		if host == "0.0.0.0" {
			baseURL = fmt.Sprintf("http://localhost:%d", port)
		}
	}

	prefix := s.GetAuthPathPrefix()
	return baseURL + prefix + "/oauth2/callback"
}

// SessionConfig contains session management settings
// Note: Session storage backend is configured via kvs.default or kvs.session
type SessionConfig struct {
	Cookie CookieConfig `yaml:"cookie" json:"cookie"`
}

// CookieConfig contains session cookie settings
type CookieConfig struct {
	Name     string `yaml:"name" json:"name"`
	Secret   string `yaml:"secret" json:"secret"`
	Expire   string `yaml:"expire" json:"expire"`
	Secure   bool   `yaml:"secure" json:"secure"`
	HTTPOnly bool   `yaml:"httponly" json:"httponly"`
	SameSite string `yaml:"samesite" json:"samesite"`
}

// GetExpireDuration returns the cookie expiration as a time.Duration
func (c CookieConfig) GetExpireDuration() (time.Duration, error) {
	return time.ParseDuration(c.Expire)
}

// GetSameSite returns the SameSite cookie attribute based on configuration
func (c CookieConfig) GetSameSite() http.SameSite {
	switch strings.ToLower(c.SameSite) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	case "lax", "":
		return http.SameSiteLaxMode
	default:
		return http.SameSiteLaxMode
	}
}

// OAuth2Config contains OAuth2 provider settings
type OAuth2Config struct {
	Providers []OAuth2Provider `yaml:"providers" json:"providers"`
}

// OAuth2Provider represents a single OAuth2 provider configuration
type OAuth2Provider struct {
	ID           string `yaml:"id" json:"id"`                     // Unique identifier for this provider (required, must be unique)
	Type         string `yaml:"type" json:"type"`                 // Provider type: "google", "github", "microsoft", "custom"
	DisplayName  string `yaml:"display_name" json:"display_name"` // Display name shown in UI
	ClientID     string `yaml:"client_id" json:"client_id"`
	ClientSecret string `yaml:"client_secret" json:"client_secret"`
	Disabled     bool   `yaml:"disabled" json:"disabled"` // If true, provider is hidden from login page
	IconURL      string `yaml:"icon_url" json:"icon_url"` // Optional custom icon URL (if not set, uses default icon based on provider type)

	// Custom provider settings (only used when Type is "custom")
	AuthURL            string `yaml:"auth_url" json:"auth_url"`                         // Custom authorization endpoint
	TokenURL           string `yaml:"token_url" json:"token_url"`                       // Custom token endpoint
	UserInfoURL        string `yaml:"userinfo_url" json:"userinfo_url"`                 // Custom userinfo endpoint
	JWKSURL            string `yaml:"jwks_url" json:"jwks_url"`                         // Optional OIDC JWKS URL
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify" json:"insecure_skip_verify"` // Allow HTTP for testing (default: false)

	// OAuth2 scopes to request
	Scopes      []string `yaml:"scopes" json:"scopes"`             // OAuth2 scopes to request (e.g., ["openid", "email", "profile", "analytics"])
	ResetScopes bool     `yaml:"reset_scopes" json:"reset_scopes"` // If true, replaces default scopes; if false, adds to default scopes (default: false)
}

// EmailAuthConfig contains email authentication settings
type EmailAuthConfig struct {
	Enabled    bool             `yaml:"enabled" json:"enabled"`
	SenderType string           `yaml:"sender_type" json:"sender_type"` // "smtp", "sendgrid", or "sendmail"
	From       string           `yaml:"from" json:"from"`               // From email address (can be RFC 5322 format: "Name <email@example.com>" or just "email@example.com")
	FromName   string           `yaml:"from_name" json:"from_name"`     // From display name (optional, used if From doesn't contain name)
	SMTP       SMTPConfig       `yaml:"smtp" json:"smtp"`
	SendGrid   SendGridConfig   `yaml:"sendgrid" json:"sendgrid"`
	Sendmail   SendmailConfig   `yaml:"sendmail" json:"sendmail"`
	Token      EmailTokenConfig `yaml:"token" json:"token"`
}

// GetFromAddress parses the From field and returns the email address and display name
// Supports RFC 5322 format: "Display Name <email@example.com>" or just "email@example.com"
// Returns (email, displayName)
func (e EmailAuthConfig) GetFromAddress() (string, string) {
	from := strings.TrimSpace(e.From)
	if from == "" {
		return "", ""
	}

	// Check for RFC 5322 format: "Name <email@example.com>"
	if strings.Contains(from, "<") && strings.Contains(from, ">") {
		startIdx := strings.Index(from, "<")
		endIdx := strings.Index(from, ">")
		if startIdx < endIdx {
			email := strings.TrimSpace(from[startIdx+1 : endIdx])
			name := strings.TrimSpace(from[:startIdx])
			// Remove surrounding quotes from name if present
			name = strings.Trim(name, `"`)
			return email, name
		}
	}

	// Plain email format: use FromName if specified
	return from, e.FromName
}

// SMTPConfig contains SMTP server settings
type SMTPConfig struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	From     string `yaml:"from,omitempty" json:"from,omitempty"`           // Optional: Override email_auth.from
	FromName string `yaml:"from_name,omitempty" json:"from_name,omitempty"` // Optional: Override email_auth.from_name
	TLS      bool   `yaml:"tls" json:"tls"`
	StartTLS bool   `yaml:"starttls" json:"starttls"`
}

// GetFromAddress returns the From address and name, with fallback to parent config
// Returns (email, displayName)
func (s SMTPConfig) GetFromAddress(parentEmail, parentName string) (string, string) {
	// Use SMTP-specific config if set (backward compatibility)
	if s.From != "" {
		return s.From, s.FromName
	}
	// Fall back to parent EmailAuthConfig
	return parentEmail, parentName
}

// SendGridConfig contains SendGrid API settings
type SendGridConfig struct {
	APIKey      string `yaml:"api_key" json:"api_key"`
	From        string `yaml:"from,omitempty" json:"from,omitempty"`           // Optional: Override email_auth.from
	FromName    string `yaml:"from_name,omitempty" json:"from_name,omitempty"` // Optional: Override email_auth.from_name
	EndpointURL string `yaml:"endpoint_url" json:"endpoint_url"`               // Optional custom endpoint URL (default: https://api.sendgrid.com)
}

// GetFromAddress returns the From address and name, with fallback to parent config
// Returns (email, displayName)
func (s SendGridConfig) GetFromAddress(parentEmail, parentName string) (string, string) {
	// Use SendGrid-specific config if set (backward compatibility)
	if s.From != "" {
		return s.From, s.FromName
	}
	// Fall back to parent EmailAuthConfig
	return parentEmail, parentName
}

// SendmailConfig contains sendmail command settings
type SendmailConfig struct {
	Path     string `yaml:"path" json:"path"`                               // Path to sendmail binary (default: /usr/sbin/sendmail)
	From     string `yaml:"from,omitempty" json:"from,omitempty"`           // Optional: Override email_auth.from
	FromName string `yaml:"from_name,omitempty" json:"from_name,omitempty"` // Optional: Override email_auth.from_name
}

// GetFromAddress returns the From address and name, with fallback to parent config
// Returns (email, displayName)
func (s SendmailConfig) GetFromAddress(parentEmail, parentName string) (string, string) {
	// Use Sendmail-specific config if set (backward compatibility)
	if s.From != "" {
		return s.From, s.FromName
	}
	// Fall back to parent EmailAuthConfig
	return parentEmail, parentName
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

// PasswordAuthConfig contains password authentication settings
// This is a simple authentication method that requires a password
// Useful for initial setup and testing without requiring email or OAuth2 configuration
type PasswordAuthConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`   // Enable password authentication
	Password string `yaml:"password" json:"password"` // Password for authentication
}

// AccessControlConfig contains access control settings
type AccessControlConfig struct {
	Emails []string     `yaml:"emails" json:"emails"` // Email addresses or domains (domain starts with @)
	Rules  rules.Config `yaml:"rules" json:"rules"`   // Access control rules configuration
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level       string             `yaml:"level" json:"level"`
	ModuleLevel string             `yaml:"module_level" json:"module_level"`
	Color       bool               `yaml:"color" json:"color"`
	File        *FileLoggingConfig `yaml:"file,omitempty" json:"file,omitempty"` // Optional file logging configuration
}

// FileLoggingConfig contains file logging and rotation settings
type FileLoggingConfig struct {
	Path       string `yaml:"path" json:"path"`                                   // Log file path (required)
	MaxSizeMB  int    `yaml:"max_size_mb,omitempty" json:"max_size_mb,omitempty"` // Maximum size in megabytes before rotation (default: 100)
	MaxBackups int    `yaml:"max_backups,omitempty" json:"max_backups,omitempty"` // Maximum number of old log files to retain (default: 3)
	MaxAge     int    `yaml:"max_age,omitempty" json:"max_age,omitempty"`         // Maximum number of days to retain old log files (default: 28)
	Compress   bool   `yaml:"compress,omitempty" json:"compress,omitempty"`       // Whether to compress rotated log files (default: false)
}

// KVSConfig contains the unified KVS configuration with optional overrides.
// This design allows sharing a single KVS backend across multiple use cases
// with namespace isolation, while still supporting dedicated backends when needed.
type KVSConfig struct {
	// Default KVS configuration (shared by all use cases)
	Default kvs.Config `yaml:"default" json:"default"`

	// Optional override for session storage
	// If nil, uses Default with session namespace prefix
	Session *kvs.Config `yaml:"session,omitempty" json:"session,omitempty"`

	// Optional override for token storage
	// If nil, uses Default with token namespace prefix
	Token *kvs.Config `yaml:"token,omitempty" json:"token,omitempty"`

	// Optional override for rate limit storage
	// If nil, uses Default with ratelimit namespace prefix
	RateLimit *kvs.Config `yaml:"ratelimit,omitempty" json:"ratelimit,omitempty"`

	// Namespace prefixes for shared KVS (has defaults)
	Namespaces NamespaceConfig `yaml:"namespaces" json:"namespaces"`
}

// NamespaceConfig defines the key prefixes for each use case when sharing a KVS
type NamespaceConfig struct {
	Session   string `yaml:"session" json:"session"`     // Default: "session"
	Token     string `yaml:"token" json:"token"`         // Default: "token"
	RateLimit string `yaml:"ratelimit" json:"ratelimit"` // Default: "ratelimit"
}

// SetDefaults sets default namespace names if not specified
func (n *NamespaceConfig) SetDefaults() {
	if n.Session == "" {
		n.Session = "session"
	}
	if n.Token == "" {
		n.Token = "token"
	}
	if n.RateLimit == "" {
		n.RateLimit = "ratelimit"
	}
}

// Validate checks if the configuration is valid
// Returns a ValidationError containing all validation errors found
func (c *Config) Validate() error {
	verr := NewValidationError()

	// Validate service name
	if c.Service.Name == "" {
		verr.Add(ErrServiceNameRequired)
	}

	// Validate session cookie secret
	if c.Session.Cookie.Secret == "" {
		verr.Add(ErrCookieSecretRequired)
	} else if len(c.Session.Cookie.Secret) < 32 {
		verr.Add(ErrCookieSecretTooShort)
	}

	// Check at least one authentication method is available (OAuth2, email, or agreement)
	hasAvailableOAuth2 := false
	for _, p := range c.OAuth2.Providers {
		if !p.Disabled {
			hasAvailableOAuth2 = true
			break
		}
	}
	hasEmailAuth := c.EmailAuth.Enabled
	hasPasswordAuth := c.PasswordAuth.Enabled

	// At least one authentication method must be enabled
	if !hasAvailableOAuth2 && !hasEmailAuth && !hasPasswordAuth {
		verr.Add(ErrNoAuthMethod)
	}

	// Validate forwarding configuration
	if err := c.validateForwarding(); err != nil {
		verr.Add(err)
	}

	// Validate rules configuration
	if err := c.AccessControl.Rules.Validate(); err != nil {
		verr.Add(fmt.Errorf("access_control.rules: %w", err))
	}

	return verr.ErrorOrNil()
}

// validateForwarding validates the forwarding configuration
func (c *Config) validateForwarding() error {
	fwd := &c.Forwarding

	// No fields defined, nothing to validate
	if len(fwd.Fields) == 0 {
		return nil
	}

	verr := NewValidationError()

	// Check if encryption is needed
	needsEncryption := false
	for _, field := range fwd.Fields {
		for _, filter := range field.Filters {
			if filter == "encrypt" {
				needsEncryption = true
				break
			}
		}
		if needsEncryption {
			break
		}
	}

	// Validate encryption config if needed
	if needsEncryption {
		if fwd.Encryption == nil {
			verr.Add(ErrEncryptionConfigRequired)
		} else {
			if fwd.Encryption.Key == "" {
				verr.Add(ErrEncryptionKeyRequired)
			} else if len(fwd.Encryption.Key) < 32 {
				verr.Add(ErrEncryptionKeyTooShort)
			}
		}
	}

	// Validate each field
	for i, field := range fwd.Fields {
		// Path is required
		if field.Path == "" {
			verr.Add(fmt.Errorf("forwarding.fields[%d]: path is required", i))
			continue
		}

		// At least one of Query or Header must be specified
		if field.Query == "" && field.Header == "" {
			verr.Add(fmt.Errorf("forwarding.fields[%d]: at least one of 'query' or 'header' must be specified", i))
		}

		// Validate filters
		// Note: base64 filter is auto-added by the system when needed, explicit specification is allowed but redundant
		validFilters := map[string]bool{"encrypt": true, "zip": true, "base64": true}
		for _, filter := range field.Filters {
			if !validFilters[filter] {
				verr.Add(fmt.Errorf("forwarding.fields[%d]: invalid filter '%s' (valid: encrypt, zip, base64)", i, filter))
			}
		}
	}

	return verr.ErrorOrNil()
}

// ForwardingConfig contains user info forwarding settings
type ForwardingConfig struct {
	Encryption *EncryptionConfig `yaml:"encryption,omitempty" json:"encryption,omitempty"` // Optional encryption settings
	Fields     []ForwardingField `yaml:"fields" json:"fields"`                             // Field forwarding definitions
}

// ForwardingField defines how to forward a single field
type ForwardingField struct {
	Path    string     `yaml:"path" json:"path"`                           // Dot-separated path to field (e.g., "email", "userinfo.avatar_url", "." for entire object)
	Query   string     `yaml:"query,omitempty" json:"query,omitempty"`     // Query parameter name for login redirect (optional)
	Header  string     `yaml:"header,omitempty" json:"header,omitempty"`   // HTTP header name for all requests (optional)
	Filters FilterList `yaml:"filters,omitempty" json:"filters,omitempty"` // Filters to apply (e.g., "encrypt,zip" or ["encrypt", "zip"])
}

// FilterList represents a list of filters (can be comma-separated string or array)
type FilterList []string

// UnmarshalYAML implements custom YAML unmarshaling to support both string and array formats
func (f *FilterList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try array format first
	var arr []string
	if err := unmarshal(&arr); err == nil {
		*f = arr
		return nil
	}

	// Try comma-separated string format
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}

	// Split by comma and trim spaces
	if str == "" {
		*f = []string{}
		return nil
	}

	parts := strings.Split(str, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	*f = result
	return nil
}

// EncryptionConfig contains encryption settings
type EncryptionConfig struct {
	Key       string `yaml:"key" json:"key"`                                 // Encryption key (required if encrypt filter is used)
	Algorithm string `yaml:"algorithm,omitempty" json:"algorithm,omitempty"` // Encryption algorithm (default: "aes-256-gcm")
}

// GetAlgorithm returns the encryption algorithm with default value
func (e EncryptionConfig) GetAlgorithm() string {
	if e.Algorithm == "" {
		return "aes-256-gcm"
	}
	return e.Algorithm
}

// AssetsConfig contains assets configuration
type AssetsConfig struct {
	Optimization OptimizationConfig `yaml:"optimization" json:"optimization"` // Optimization settings
}

// OptimizationConfig contains optimization settings for assets
type OptimizationConfig struct {
	Dify bool `yaml:"dify" json:"dify"` // If true, load dify.css for iframe optimizations
}
