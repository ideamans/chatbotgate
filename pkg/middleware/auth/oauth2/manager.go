package oauth2

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/oauth2"
)

var (
	// ErrProviderNotFound is returned when a provider is not found
	ErrProviderNotFound = errors.New("OAuth2 provider not found")
	// ErrEmailNotAvailable is returned when the OAuth2 provider does not provide an email address
	ErrEmailNotAvailable = errors.New("OAuth2 provider did not provide an email address")
)

// Manager manages OAuth2 providers and authentication flow
type Manager struct {
	providers map[string]Provider
}

// NewManager creates a new OAuth2 manager
func NewManager() *Manager {
	return &Manager{
		providers: make(map[string]Provider),
	}
}

// AddProvider adds a provider to the manager
func (m *Manager) AddProvider(provider Provider) {
	m.providers[provider.Name()] = provider
}

// GetProvider retrieves a provider by name
func (m *Manager) GetProvider(name string) (Provider, error) {
	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}
	return provider, nil
}

// GetProviders returns all providers
func (m *Manager) GetProviders() []Provider {
	providers := make([]Provider, 0, len(m.providers))
	for _, p := range m.providers {
		providers = append(providers, p)
	}
	return providers
}

// GetAuthURL generates an authorization URL for a provider
func (m *Manager) GetAuthURL(providerName, state string) (string, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return "", err
	}

	config := provider.Config()
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// GetAuthURLWithHost generates an authorization URL for a provider with a custom redirect URL
// based on the request host or base URL. This is useful for dynamic port mapping (e.g., Docker environments).
// The hostOrBaseURL parameter can be:
//   - A full base URL (e.g., "http://localhost:4182")
//   - A host with port (e.g., "localhost:4182")
//   - Just a host (e.g., "example.com")
//
// Deprecated: Use GetAuthURLWithRedirect instead which returns both auth URL and redirect URL
func (m *Manager) GetAuthURLWithHost(providerName, state, hostOrBaseURL, authPathPrefix string) (string, error) {
	authURL, _, err := m.GetAuthURLWithRedirect(providerName, state, hostOrBaseURL, authPathPrefix)
	return authURL, err
}

// GetAuthURLWithRedirect generates an authorization URL and returns both the auth URL and redirect URL.
// This is useful for dynamic port mapping (e.g., Docker environments).
// The hostOrBaseURL parameter can be:
//   - A full base URL (e.g., "http://localhost:4182")
//   - A host with port (e.g., "localhost:4182")
//   - Just a host (e.g., "example.com")
//
// Returns: (authURL, redirectURL, error)
func (m *Manager) GetAuthURLWithRedirect(providerName, state, hostOrBaseURL, authPathPrefix string) (string, string, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return "", "", err
	}

	// Get the original config
	originalConfig := provider.Config()

	// Create a copy of the config with a dynamic redirect URL
	config := &oauth2.Config{
		ClientID:     originalConfig.ClientID,
		ClientSecret: originalConfig.ClientSecret,
		Endpoint:     originalConfig.Endpoint,
		Scopes:       originalConfig.Scopes,
	}

	// Normalize authPathPrefix
	redirectPath := authPathPrefix
	if redirectPath == "" {
		redirectPath = "/_auth"
	}
	if redirectPath[len(redirectPath)-1] == '/' {
		redirectPath = redirectPath[:len(redirectPath)-1]
	}

	// Build redirect URL
	var baseURL string
	if len(hostOrBaseURL) > 7 && (hostOrBaseURL[:7] == "http://" || hostOrBaseURL[:8] == "https://") {
		// It's a full base URL, use it as-is
		baseURL = hostOrBaseURL
	} else {
		// It's just a host or host:port, prepend scheme
		baseURL = "http://" + hostOrBaseURL
	}

	// Remove trailing slash from baseURL if present
	if len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}

	redirectURL := fmt.Sprintf("%s%s/oauth2/callback", baseURL, redirectPath)
	config.RedirectURL = redirectURL

	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return authURL, redirectURL, nil
}

// Exchange exchanges an authorization code for a token
func (m *Manager) Exchange(ctx context.Context, providerName, code string) (*oauth2.Token, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	config := provider.Config()
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return token, nil
}

// ExchangeWithRedirect exchanges an authorization code for a token using a custom redirect URL.
// This is required when the redirect URL used in the authorization request differs from the
// provider's configured redirect URL (e.g., in Docker environments with port mapping).
func (m *Manager) ExchangeWithRedirect(ctx context.Context, providerName, code, redirectURL string) (*oauth2.Token, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	// Get the original config
	originalConfig := provider.Config()

	// Create a copy of the config with the custom redirect URL
	config := &oauth2.Config{
		ClientID:     originalConfig.ClientID,
		ClientSecret: originalConfig.ClientSecret,
		Endpoint:     originalConfig.Endpoint,
		Scopes:       originalConfig.Scopes,
		RedirectURL:  redirectURL,
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code with redirect URL %s: %w", redirectURL, err)
	}

	return token, nil
}

// GetUserInfo retrieves the user's information using a token
func (m *Manager) GetUserInfo(ctx context.Context, providerName string, token *oauth2.Token) (*UserInfo, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return provider.GetUserInfo(ctx, token)
}

// GetUserEmail retrieves the user's email using a token (deprecated, use GetUserInfo)
func (m *Manager) GetUserEmail(ctx context.Context, providerName string, token *oauth2.Token) (string, error) {
	userInfo, err := m.GetUserInfo(ctx, providerName, token)
	if err != nil {
		return "", err
	}
	return userInfo.Email, nil
}

// GenerateState generates a random state string for CSRF protection
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
