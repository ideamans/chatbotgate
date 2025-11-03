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
	Extra    map[string]interface{} // Additional data from OAuth2 provider (for custom forwarding)
	Provider string                 // OAuth2 provider name (for provider-specific forwarding)
}

// Forwarder handles user information forwarding
type Forwarder struct {
	config            *config.ForwardingConfig
	providerConfigs   map[string]*config.OAuth2Provider // Provider-specific forwarding configurations
	encryptor         *Encryptor
}

// NewForwarder creates a new Forwarder
func NewForwarder(cfg *config.ForwardingConfig, providers []config.OAuth2Provider) *Forwarder {
	f := &Forwarder{
		config:          cfg,
		providerConfigs: make(map[string]*config.OAuth2Provider),
	}

	// Build provider config map for quick lookup
	for i := range providers {
		f.providerConfigs[providers[i].Name] = &providers[i]
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

	// Add provider-specific custom fields (if provider and Extra data are available)
	if userInfo.Provider != "" && userInfo.Extra != nil {
		if providerCfg, ok := f.providerConfigs[userInfo.Provider]; ok && providerCfg.Forwarding != nil {
			for _, custom := range providerCfg.Forwarding.Custom {
				// Skip if no query parameter specified
				if custom.Query == "" {
					continue
				}

				// Get value from Extra using dot-separated path
				value := GetValueByPath(userInfo.Extra, custom.Path)
				if value != "" {
					if f.config.QueryString.Encrypt {
						// Encrypt the value
						encrypted, err := f.encryptor.Encrypt(value)
						if err != nil {
							return "", err
						}
						q.Set(custom.Query, encrypted)
					} else {
						// Plain text
						q.Set(custom.Query, value)
					}
				}
			}
		}
	}

	// Update URL with merged query parameters
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// AddToHeaders adds user info to HTTP headers as X-ChatbotGate-*
// If encryption is enabled, values are encrypted
// Individual headers: X-ChatbotGate-User (username), X-ChatbotGate-Email
func (f *Forwarder) AddToHeaders(headers http.Header, userInfo *UserInfo) http.Header {
	if !f.config.Header.Enabled {
		return headers
	}

	// Clone headers
	result := make(http.Header)
	for key, values := range headers {
		result[key] = values
	}

	// Add individual X-ChatbotGate-* headers
	for _, field := range f.config.Fields {
		var value string
		var headerName string

		switch field {
		case "username":
			if userInfo.Username != "" {
				value = userInfo.Username
				headerName = "X-ChatbotGate-User"
			}
		case "email":
			if userInfo.Email != "" {
				value = userInfo.Email
				headerName = "X-ChatbotGate-Email"
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

	// Add provider-specific custom fields (if provider and Extra data are available)
	if userInfo.Provider != "" && userInfo.Extra != nil {
		if providerCfg, ok := f.providerConfigs[userInfo.Provider]; ok && providerCfg.Forwarding != nil {
			for _, custom := range providerCfg.Forwarding.Custom {
				// Skip if no header specified
				if custom.Header == "" {
					continue
				}

				// Get value from Extra using dot-separated path
				value := GetValueByPath(userInfo.Extra, custom.Path)
				if value != "" {
					if f.config.Header.Encrypt {
						// Encrypt the value
						encrypted, err := f.encryptor.Encrypt(value)
						if err != nil {
							// Log error but don't fail the request
							continue
						}
						result.Set(custom.Header, encrypted)
					} else {
						// Plain text
						result.Set(custom.Header, value)
					}
				}
			}
		}
	}

	return result
}
