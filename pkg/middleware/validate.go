package middleware

import (
	"fmt"
	"strings"

	"github.com/ideamans/chatbotgate/pkg/config"
)

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string // Field name (e.g., "server.port", "session.cookie_secret")
	Message string // Error message
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
// Returns a formatted string of all validation errors
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "no validation errors"
	}

	var buf strings.Builder
	buf.WriteString("configuration validation failed:")
	for i, err := range ve {
		fmt.Fprintf(&buf, "\n  %d. %s: %s", i+1, err.Field, err.Message)
	}
	return buf.String()
}

// ValidateConfig validates the middleware configuration
// Returns a list of validation errors, or nil if validation passes
func ValidateConfig(cfg *config.Config) ValidationErrors {
	var errs ValidationErrors

	// Validate Service
	if cfg.Service.Name == "" {
		errs = append(errs, ValidationError{
			Field:   "service.name",
			Message: "service name is required",
		})
	}

	// Validate Server
	if cfg.Server.AuthPathPrefix != "" {
		prefix := cfg.Server.AuthPathPrefix
		if !strings.HasPrefix(prefix, "/") {
			errs = append(errs, ValidationError{
				Field:   "server.auth_path_prefix",
				Message: "auth path prefix must start with '/'",
			})
		}
		if strings.HasSuffix(prefix, "/") {
			errs = append(errs, ValidationError{
				Field:   "server.auth_path_prefix",
				Message: "auth path prefix must not end with '/'",
			})
		}
		// Check for invalid characters
		if strings.Contains(prefix, " ") || strings.Contains(prefix, "\t") {
			errs = append(errs, ValidationError{
				Field:   "server.auth_path_prefix",
				Message: "auth path prefix must not contain whitespace",
			})
		}
	}

	// Validate Proxy
	if cfg.Proxy.Upstream == "" {
		errs = append(errs, ValidationError{
			Field:   "proxy.upstream",
			Message: "upstream URL is required",
		})
	}

	// Validate Session
	if cfg.Session.CookieName == "" {
		errs = append(errs, ValidationError{
			Field:   "session.cookie_name",
			Message: "cookie name is required",
		})
	}

	if cfg.Session.CookieSecret == "" {
		errs = append(errs, ValidationError{
			Field:   "session.cookie_secret",
			Message: "cookie secret is required",
		})
	} else if len(cfg.Session.CookieSecret) < 32 {
		errs = append(errs, ValidationError{
			Field:   "session.cookie_secret",
			Message: fmt.Sprintf("cookie secret must be at least 32 characters (current: %d)", len(cfg.Session.CookieSecret)),
		})
	}

	if cfg.Session.CookieExpire == "" {
		errs = append(errs, ValidationError{
			Field:   "session.cookie_expire",
			Message: "cookie expiration is required",
		})
	} else {
		// Validate duration format
		if _, err := cfg.Session.GetCookieExpireDuration(); err != nil {
			errs = append(errs, ValidationError{
				Field:   "session.cookie_expire",
				Message: fmt.Sprintf("invalid duration format: %v", err),
			})
		}
	}

	// Validate session store type
	if cfg.Session.StoreType != "" && cfg.Session.StoreType != "memory" && cfg.Session.StoreType != "redis" {
		errs = append(errs, ValidationError{
			Field:   "session.store_type",
			Message: "store type must be 'memory' or 'redis'",
		})
	}

	// Validate Redis configuration if Redis store is used
	if cfg.Session.StoreType == "redis" {
		if cfg.Session.Redis.Addr == "" {
			errs = append(errs, ValidationError{
				Field:   "session.redis.addr",
				Message: "redis address is required when using redis store",
			})
		}
	}

	// Validate OAuth2 - at least one provider must be enabled
	hasEnabledOAuth2Provider := false
	for i, provider := range cfg.OAuth2.Providers {
		if provider.Enabled {
			hasEnabledOAuth2Provider = true
			// Validate enabled provider configuration
			if provider.Name == "" {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("oauth2.providers[%d].name", i),
					Message: "provider name is required",
				})
			}
			if provider.ClientID == "" {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("oauth2.providers[%d].client_id", i),
					Message: fmt.Sprintf("client ID is required for provider '%s'", provider.Name),
				})
			}
			if provider.ClientSecret == "" {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("oauth2.providers[%d].client_secret", i),
					Message: fmt.Sprintf("client secret is required for provider '%s'", provider.Name),
				})
			}

			// Validate custom provider
			providerType := provider.Type
			if providerType == "" {
				providerType = provider.Name
			}
			if providerType == "custom" {
				if provider.AuthURL == "" {
					errs = append(errs, ValidationError{
						Field:   fmt.Sprintf("oauth2.providers[%d].auth_url", i),
						Message: fmt.Sprintf("auth URL is required for custom provider '%s'", provider.Name),
					})
				}
				if provider.TokenURL == "" {
					errs = append(errs, ValidationError{
						Field:   fmt.Sprintf("oauth2.providers[%d].token_url", i),
						Message: fmt.Sprintf("token URL is required for custom provider '%s'", provider.Name),
					})
				}
				if provider.UserInfoURL == "" {
					errs = append(errs, ValidationError{
						Field:   fmt.Sprintf("oauth2.providers[%d].userinfo_url", i),
						Message: fmt.Sprintf("user info URL is required for custom provider '%s'", provider.Name),
					})
				}
			}
		}
	}

	// Validate Email Authentication
	hasEnabledEmailAuth := cfg.EmailAuth.Enabled

	// At least one authentication method must be enabled
	if !hasEnabledOAuth2Provider && !hasEnabledEmailAuth {
		errs = append(errs, ValidationError{
			Field:   "oauth2.providers / email_auth.enabled",
			Message: "at least one authentication method must be enabled (OAuth2 provider or email authentication)",
		})
	}

	// Validate email authentication configuration if enabled
	if hasEnabledEmailAuth {
		if cfg.EmailAuth.SenderType == "" {
			errs = append(errs, ValidationError{
				Field:   "email_auth.sender_type",
				Message: "sender type is required when email authentication is enabled (must be 'smtp' or 'sendgrid')",
			})
		} else if cfg.EmailAuth.SenderType != "smtp" && cfg.EmailAuth.SenderType != "sendgrid" {
			errs = append(errs, ValidationError{
				Field:   "email_auth.sender_type",
				Message: "sender type must be 'smtp' or 'sendgrid'",
			})
		}

		// Validate SMTP configuration
		if cfg.EmailAuth.SenderType == "smtp" {
			if cfg.EmailAuth.SMTP.Host == "" {
				errs = append(errs, ValidationError{
					Field:   "email_auth.smtp.host",
					Message: "SMTP host is required when using SMTP sender",
				})
			}
			if cfg.EmailAuth.SMTP.Port == 0 {
				errs = append(errs, ValidationError{
					Field:   "email_auth.smtp.port",
					Message: "SMTP port is required when using SMTP sender",
				})
			}
			if cfg.EmailAuth.SMTP.From == "" {
				errs = append(errs, ValidationError{
					Field:   "email_auth.smtp.from",
					Message: "SMTP from address is required when using SMTP sender",
				})
			}
		}

		// Validate SendGrid configuration
		if cfg.EmailAuth.SenderType == "sendgrid" {
			if cfg.EmailAuth.SendGrid.APIKey == "" {
				errs = append(errs, ValidationError{
					Field:   "email_auth.sendgrid.api_key",
					Message: "SendGrid API key is required when using SendGrid sender",
				})
			}
			if cfg.EmailAuth.SendGrid.From == "" {
				errs = append(errs, ValidationError{
					Field:   "email_auth.sendgrid.from",
					Message: "SendGrid from address is required when using SendGrid sender",
				})
			}
		}

		// Validate token expiration
		if cfg.EmailAuth.Token.Expire != "" {
			if _, err := cfg.EmailAuth.Token.GetTokenExpireDuration(); err != nil {
				errs = append(errs, ValidationError{
					Field:   "email_auth.token.expire",
					Message: fmt.Sprintf("invalid duration format: %v", err),
				})
			}
		}
	}

	// Return nil if no errors
	if len(errs) == 0 {
		return nil
	}

	return errs
}
