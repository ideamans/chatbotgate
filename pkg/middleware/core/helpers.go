package middleware

import (
	"net/http"
	"net/mail"
	"strings"
)

const (
	redirectCookieName = "_oauth2_redirect" // Cookie name for storing redirect URL
)

// staticResourcePaths are paths that should not trigger authentication or be saved as redirect URLs
var staticResourcePaths = []string{
	"/favicon.ico",
	"/robots.txt",
	"/apple-touch-icon.png",
	"/apple-touch-icon-precomposed.png",
	"/_auth/assets/", // Auth page assets (CSS, icons, etc.)
}

// isStaticResource checks if the request path is for a static resource
func isStaticResource(path string) bool {
	for _, staticPath := range staticResourcePaths {
		if path == staticPath || strings.HasPrefix(path, staticPath) {
			return true
		}
	}
	return false
}

// isValidRedirectURL validates a redirect URL to prevent open redirect attacks
// Only allows relative URLs that start with "/" and do not contain "://" or start with "//"
func isValidRedirectURL(redirectURL string) bool {
	// Empty URL is not valid
	if redirectURL == "" {
		return false
	}

	// Must start with "/" for relative URL
	if !strings.HasPrefix(redirectURL, "/") {
		return false
	}

	// Reject protocol-relative URLs (//example.com)
	if strings.HasPrefix(redirectURL, "//") {
		return false
	}

	// Reject absolute URLs (http://, https://, etc.)
	if strings.Contains(redirectURL, "://") {
		return false
	}

	return true
}

// getRedirectURL retrieves and deletes the redirect URL from cookie
func (m *Middleware) getRedirectURL(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie(redirectCookieName)
	if err != nil {
		return "/" // Default to home if no redirect cookie
	}

	// Delete the redirect cookie
	http.SetCookie(w, &http.Cookie{
		Name:   redirectCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	redirectURL := cookie.Value

	// Security check: only allow valid relative URLs to prevent open redirect
	if !isValidRedirectURL(redirectURL) {
		return "/"
	}

	return redirectURL
}

func normalizeAuthPrefix(prefix string) string {
	if prefix == "" {
		return "/_auth"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if len(prefix) > 1 && strings.HasSuffix(prefix, "/") {
		prefix = strings.TrimSuffix(prefix, "/")
	}
	return prefix
}

// extractPathParam extracts a path parameter from the URL
// For example, extractPathParam(r.URL.Path, "/_auth/oauth2/start/") returns "google" from "/_auth/oauth2/start/google"
func extractPathParam(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	param := strings.TrimPrefix(path, prefix)
	// Remove any trailing slashes and query parameters
	if idx := strings.Index(param, "?"); idx != -1 {
		param = param[:idx]
	}
	param = strings.TrimSuffix(param, "/")
	return param
}

// maskEmail masks an email address for logging purposes
// Example: "user@example.com" -> "u***@example.com"
func maskEmail(email string) string {
	if email == "" {
		return "[EMPTY]"
	}

	atIndex := strings.Index(email, "@")
	if atIndex <= 0 {
		return "[INVALID_EMAIL]"
	}

	localPart := email[:atIndex]
	domain := email[atIndex:]

	if len(localPart) == 1 {
		return "*" + domain
	}

	return string(localPart[0]) + "***" + domain
}

// maskToken masks a token for logging purposes
// Shows only first 8 characters followed by "..."
// Currently unused but kept for potential future use in token-related logging
func maskToken(token string) string { //nolint:unused // Reserved for future token logging
	if token == "" {
		return "[EMPTY]"
	}

	if len(token) <= 8 {
		return "[REDACTED]"
	}

	return token[:8] + "..."
}

// isValidEmail validates an email address and rejects those with control characters
// This prevents SMTP header injection and email-based attacks
func isValidEmail(email string) bool {
	// Use net/mail to parse and validate
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	// Check for control characters that could enable injection
	for _, r := range addr.Address {
		if r < 32 || r == 127 {
			return false
		}
	}

	// Must contain @ symbol
	if !strings.Contains(addr.Address, "@") {
		return false
	}

	return true
}

// sanitizeHeaderValue removes control characters and limits length for header values
// This prevents header injection attacks via user-controlled data
func sanitizeHeaderValue(value string) string { //nolint:unused // Used by forwarding package
	// Remove control characters (including CR/LF)
	cleaned := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 { // Control characters
			return -1 // Remove
		}
		return r
	}, value)

	// Limit length to prevent DoS
	const maxHeaderLength = 8192
	if len(cleaned) > maxHeaderLength {
		cleaned = cleaned[:maxHeaderLength]
	}

	return cleaned
}

// setSecurityHeaders sets security-related HTTP headers
// In development mode, CSP allows unsafe-inline scripts for easier testing
func (m *Middleware) setSecurityHeaders(w http.ResponseWriter) {
	// Content Security Policy - restrict resource loading to prevent XSS
	scriptSrc := "script-src 'self';"
	if m.config.Server.Development {
		// In development mode, allow inline scripts for testing
		scriptSrc = "script-src 'self' 'unsafe-inline';"
	}

	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; "+
			scriptSrc+" "+
			"style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data: https:; "+
			"font-src 'self'; "+
			"connect-src 'self'; "+
			"frame-ancestors 'none'; "+
			"base-uri 'self'; "+
			"form-action 'self'")

	// Prevent browsers from MIME-sniffing
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Prevent clickjacking
	w.Header().Set("X-Frame-Options", "DENY")

	// Enable XSS protection (for older browsers)
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Referrer policy - don't leak URLs
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}
