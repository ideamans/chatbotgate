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

// GetUserEmail retrieves the user's email using a token
func (m *Manager) GetUserEmail(ctx context.Context, providerName string, token *oauth2.Token) (string, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return "", err
	}

	return provider.GetUserEmail(ctx, token)
}

// GenerateState generates a random state string for CSRF protection
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
