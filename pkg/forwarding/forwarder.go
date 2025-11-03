package forwarding

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/ideamans/chatbotgate/pkg/config"
)

var (
	// ErrNoFieldsConfigured is returned when no fields are configured for forwarding
	ErrNoFieldsConfigured = errors.New("no fields configured for forwarding")
)

// UserInfo contains user information to be forwarded
type UserInfo struct {
	Username string
	Email    string
}

// Forwarder handles user information forwarding
type Forwarder struct {
	config    *config.ForwardingConfig
	encryptor *Encryptor
}

// NewForwarder creates a new Forwarder
func NewForwarder(cfg *config.ForwardingConfig) *Forwarder {
	f := &Forwarder{
		config: cfg,
	}

	// Initialize encryptor if encryption is needed
	if (cfg.QueryString.Encrypt || cfg.Header.Encrypt) && cfg.Encryption.Key != "" {
		f.encryptor = NewEncryptor(cfg.Encryption.Key)
	}

	return f
}

// AddToQueryString adds user info to a URL's query string
// Parameters are added as chatbotgate.user, chatbotgate.email, etc.
// If encryption is enabled, the values are encrypted
// Merges with existing query parameters
func (f *Forwarder) AddToQueryString(targetURL string, userInfo *UserInfo) (string, error) {
	if !f.config.QueryString.Enabled {
		return targetURL, nil
	}

	// Parse the URL
	u, err := url.Parse(targetURL)
	if err != nil {
		return "", err
	}

	// Get existing query parameters (this merges with existing params)
	q := u.Query()

	// Add individual parameters with chatbotgate. prefix
	for _, field := range f.config.Fields {
		var value string
		var paramName string

		switch field {
		case "username":
			if userInfo.Username != "" {
				value = userInfo.Username
				paramName = "chatbotgate.user"
			}
		case "email":
			if userInfo.Email != "" {
				value = userInfo.Email
				paramName = "chatbotgate.email"
			}
		}

		if value != "" {
			if f.config.QueryString.Encrypt {
				// Encrypt the value
				encrypted, err := f.encryptor.Encrypt(value)
				if err != nil {
					return "", err
				}
				q.Set(paramName, encrypted)
			} else {
				// Plain text
				q.Set(paramName, value)
			}
		}
	}

	// Update URL with merged query parameters
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// AddToHeaders adds user info to HTTP headers as X-Forwarded-*
// If encryption is enabled, values are encrypted
// Individual headers: X-Forwarded-User (username), X-Forwarded-Email
func (f *Forwarder) AddToHeaders(headers http.Header, userInfo *UserInfo) http.Header {
	if !f.config.Header.Enabled {
		return headers
	}

	// Clone headers
	result := make(http.Header)
	for key, values := range headers {
		result[key] = values
	}

	// Add individual X-Forwarded-* headers
	for _, field := range f.config.Fields {
		var value string
		var headerName string

		switch field {
		case "username":
			if userInfo.Username != "" {
				value = userInfo.Username
				headerName = "X-Forwarded-User"
			}
		case "email":
			if userInfo.Email != "" {
				value = userInfo.Email
				headerName = "X-Forwarded-Email"
			}
		}

		if value != "" {
			if f.config.Header.Encrypt {
				// Encrypt the value
				encrypted, err := f.encryptor.Encrypt(value)
				if err != nil {
					// Log error but don't fail the request
					continue
				}
				result.Set(headerName, encrypted)
			} else {
				// Plain text
				result.Set(headerName, value)
			}
		}
	}

	return result
}
