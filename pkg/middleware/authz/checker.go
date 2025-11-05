package authz

import (
	"strings"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
)

// Checker is an interface for authorization checking
type Checker interface {
	// RequiresEmail returns true if email-based authorization is required
	// If false, authentication alone is sufficient (no whitelist configured)
	RequiresEmail() bool

	// IsAllowed checks if an email address is authorized
	// If RequiresEmail() is false, this always returns true
	IsAllowed(email string) bool
}

// EmailChecker checks authorization based on email whitelist
type EmailChecker struct {
	allowedEmails  map[string]bool
	allowedDomains []string
}

// NewEmailChecker creates a new EmailChecker from configuration
func NewEmailChecker(cfg config.AuthorizationConfig) *EmailChecker {
	// Convert allowed entries to emails and domains
	// Entries starting with @ are domains, others are email addresses
	emailMap := make(map[string]bool)
	var domains []string

	for _, entry := range cfg.Allowed {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if strings.HasPrefix(entry, "@") {
			// Domain entry
			domains = append(domains, strings.ToLower(entry))
		} else {
			// Email address entry
			emailMap[strings.ToLower(entry)] = true
		}
	}

	return &EmailChecker{
		allowedEmails:  emailMap,
		allowedDomains: domains,
	}
}

// RequiresEmail returns true if email-based authorization is required
// Returns false if no whitelist is configured (authentication alone is sufficient)
func (c *EmailChecker) RequiresEmail() bool {
	return len(c.allowedEmails) > 0 || len(c.allowedDomains) > 0
}

// IsAllowed checks if an email address is authorized
// If no whitelist is configured, always returns true
func (c *EmailChecker) IsAllowed(email string) bool {
	// If no whitelist is configured, allow all authenticated users
	if !c.RequiresEmail() {
		return true
	}

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
