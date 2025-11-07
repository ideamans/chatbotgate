package forwarding

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
)

var (
	// ErrNoFieldsConfigured is returned when no fields are configured for forwarding
	ErrNoFieldsConfigured = errors.New("no fields configured for forwarding")
)

// sanitizeHeaderValue removes control characters and limits length for header values
// This prevents header injection attacks via user-controlled data
func sanitizeHeaderValue(value string) string {
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

// UserInfo contains user information to be forwarded
type UserInfo struct {
	Username string
	Email    string
	Extra    map[string]interface{} // Additional data from OAuth2 provider
	Provider string                 // OAuth2 provider name
}

// Forwarder is the interface for forwarding user information
type Forwarder interface {
	// AddToHeaders adds user info to HTTP headers
	// Returns a new http.Header with user information added
	AddToHeaders(headers http.Header, userInfo *UserInfo) http.Header

	// AddToQueryString adds user info to a URL's query string
	// Returns the modified URL with user information in query parameters
	AddToQueryString(targetURL string, userInfo *UserInfo) (string, error)
}

// DefaultForwarder is the default implementation of Forwarder
type DefaultForwarder struct {
	config    *config.ForwardingConfig
	encryptor *Encryptor
}

// NewForwarder creates a new DefaultForwarder
func NewForwarder(cfg *config.ForwardingConfig, providers []config.OAuth2Provider) *DefaultForwarder {
	f := &DefaultForwarder{
		config: cfg,
	}

	// Initialize encryptor if encryption config is provided
	if cfg.Encryption != nil && cfg.Encryption.Key != "" {
		f.encryptor = NewEncryptor(cfg.Encryption.Key)
	}

	return f
}

// AddToQueryString adds user info to a URL's query string
// Processes each configured field and adds it as a query parameter
func (f *DefaultForwarder) AddToQueryString(targetURL string, userInfo *UserInfo) (string, error) {
	// Parse the URL
	u, err := url.Parse(targetURL)
	if err != nil {
		return "", err
	}

	// Get existing query parameters
	q := u.Query()

	// Track which query parameters have been set (for priority: first successful path wins)
	setParams := make(map[string]bool)

	// Process each field
	for _, field := range f.config.Fields {
		// Skip if query is not specified for this field
		if field.Query == "" {
			continue
		}

		// Skip if this query parameter was already set by a previous field
		if setParams[field.Query] {
			continue
		}

		// Get the value for this field
		value, err := f.getFieldValue(userInfo, field.Path)
		if err != nil {
			// Skip fields that cannot be retrieved
			continue
		}

		// Apply filters
		processed, err := f.applyFilters(value, field.Filters)
		if err != nil {
			return "", fmt.Errorf("field %s: %w", field.Path, err)
		}

		// Add to query string and mark as set
		q.Set(field.Query, processed)
		setParams[field.Query] = true
	}

	// Update URL with merged query parameters
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// AddToHeaders adds user info to HTTP headers
// Processes each configured field and adds it as an HTTP header
func (f *DefaultForwarder) AddToHeaders(headers http.Header, userInfo *UserInfo) http.Header {
	// Clone headers
	result := make(http.Header)
	for key, values := range headers {
		result[key] = values
	}

	// Track which headers have been set (for priority: first successful path wins)
	setHeaders := make(map[string]bool)

	// Process each field
	for _, field := range f.config.Fields {
		// Skip if header is not specified for this field
		if field.Header == "" {
			continue
		}

		// Skip if this header was already set by a previous field
		if setHeaders[field.Header] {
			continue
		}

		// Get the value for this field
		value, err := f.getFieldValue(userInfo, field.Path)
		if err != nil {
			// Skip fields that cannot be retrieved
			continue
		}

		// Apply filters
		processed, err := f.applyFilters(value, field.Filters)
		if err != nil {
			// Log error but don't fail the request
			continue
		}

		// Sanitize header value to prevent injection attacks
		sanitized := sanitizeHeaderValue(processed)

		// Add to headers and mark as set
		result.Set(field.Header, sanitized)
		setHeaders[field.Header] = true
	}

	return result
}

// getFieldValue retrieves the value for a given path from UserInfo
// Supports dot-separated paths (e.g., "email", "extra.secrets.access_token")
// Special path "." returns the entire UserInfo object as JSON
func (f *DefaultForwarder) getFieldValue(userInfo *UserInfo, path string) (string, error) {
	// Special case: "." means entire object
	if path == "." {
		// Convert entire UserInfo to JSON
		data := map[string]interface{}{
			"username": userInfo.Username,
			"email":    userInfo.Email,
			"provider": userInfo.Provider,
			"extra":    userInfo.Extra,
		}
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return "", fmt.Errorf("failed to marshal userinfo: %w", err)
		}
		return string(jsonBytes), nil
	}

	// Handle standard fields
	switch path {
	case "username":
		if userInfo.Username == "" {
			return "", errors.New("username is empty")
		}
		return userInfo.Username, nil
	case "email", ".email":
		if userInfo.Email == "" {
			return "", errors.New("email is empty")
		}
		return userInfo.Email, nil
	case "provider", ".provider":
		if userInfo.Provider == "" {
			return "", errors.New("provider is empty")
		}
		return userInfo.Provider, nil
	}

	// Handle paths starting with "extra." or ".extra."
	if len(path) > 6 && path[:6] == "extra." {
		return f.getValueFromExtra(userInfo.Extra, path[6:])
	}
	if len(path) > 7 && path[:7] == ".extra." {
		return f.getValueFromExtra(userInfo.Extra, path[7:])
	}

	// Try as extra field without prefix
	return f.getValueFromExtra(userInfo.Extra, path)
}

// getValueFromExtra retrieves a value from the Extra map using dot-separated path
func (f *DefaultForwarder) getValueFromExtra(extra map[string]interface{}, path string) (string, error) {
	if extra == nil {
		return "", errors.New("extra data is nil")
	}

	value := GetValueByPath(extra, path)
	if value == "" {
		return "", fmt.Errorf("field not found: %s", path)
	}

	return value, nil
}

// applyFilters applies the filter chain to the value
func (f *DefaultForwarder) applyFilters(value string, filters config.FilterList) (string, error) {
	if len(filters) == 0 {
		return value, nil
	}

	// Create filter chain
	chain, err := NewFilterChain(filters, f.encryptor)
	if err != nil {
		return "", err
	}

	// Apply filters
	return chain.Apply(value)
}
