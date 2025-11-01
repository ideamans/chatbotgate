package authz

import (
	"strings"

	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
)

// Checker is an interface for authorization checking
type Checker interface {
	IsAllowed(email string) bool
}

// EmailChecker checks authorization based on email whitelist
type EmailChecker struct {
	allowedEmails  map[string]bool
	allowedDomains []string
}

// NewEmailChecker creates a new EmailChecker from configuration
func NewEmailChecker(cfg config.AuthorizationConfig) *EmailChecker {
	// Convert allowed emails to a map for faster lookup
	emailMap := make(map[string]bool)
	for _, email := range cfg.AllowedEmails {
		emailMap[strings.ToLower(email)] = true
	}

	// Store allowed domains (already with @ prefix in config)
	domains := make([]string, len(cfg.AllowedDomains))
	for i, domain := range cfg.AllowedDomains {
		domains[i] = strings.ToLower(domain)
	}

	return &EmailChecker{
		allowedEmails:  emailMap,
		allowedDomains: domains,
	}
}

// IsAllowed checks if an email address is authorized
func (c *EmailChecker) IsAllowed(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))

	// Check if email is in the allowed list
	if c.allowedEmails[email] {
		return true
	}

	// Check if email domain is in the allowed domains
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false // Invalid email format
	}

	domain := "@" + parts[1]
	for _, allowedDomain := range c.allowedDomains {
		if domain == allowedDomain {
			return true
		}
	}

	return false
}
