package oauth2

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/oauth2"
)

// MockProvider is a mock OAuth2 provider for testing
type MockProvider struct {
	name      string
	config    *oauth2.Config
	userEmail string
	err       error
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Config() *oauth2.Config {
	return m.config
}

func (m *MockProvider) GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.userEmail, nil
}

func TestManager_AddAndGetProvider(t *testing.T) {
	manager := NewManager()

	mockProvider := &MockProvider{
		name: "mock",
		config: &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost:4180/oauth2/callback",
		},
		userEmail: "user@example.com",
	}

	manager.AddProvider(mockProvider)

	// Test GetProvider
	provider, err := manager.GetProvider("mock")
	if err != nil {
		t.Fatalf("GetProvider() error = %v", err)
	}

	if provider.Name() != "mock" {
		t.Errorf("provider.Name() = %s, want mock", provider.Name())
	}
}

func TestManager_GetProviderNotFound(t *testing.T) {
	manager := NewManager()

	_, err := manager.GetProvider("nonexistent")
	if err == nil {
		t.Error("GetProvider() should return error for nonexistent provider")
		return
	}

	// Check if error contains ErrProviderNotFound using errors.Is
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("GetProvider() error should contain ErrProviderNotFound, got %v", err)
	}
}

func TestManager_GetProviders(t *testing.T) {
	manager := NewManager()

	mock1 := &MockProvider{name: "mock1"}
	mock2 := &MockProvider{name: "mock2"}

	manager.AddProvider(mock1)
	manager.AddProvider(mock2)

	providers := manager.GetProviders()

	if len(providers) != 2 {
		t.Errorf("GetProviders() returned %d providers, want 2", len(providers))
	}
}

func TestManager_GetAuthURL(t *testing.T) {
	manager := NewManager()

	mockProvider := &MockProvider{
		name: "mock",
		config: &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost:4180/oauth2/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://example.com/auth",
				TokenURL: "https://example.com/token",
			},
		},
	}

	manager.AddProvider(mockProvider)

	url, err := manager.GetAuthURL("mock", "test-state")
	if err != nil {
		t.Fatalf("GetAuthURL() error = %v", err)
	}

	if url == "" {
		t.Error("GetAuthURL() returned empty URL")
	}

	// URL should contain the auth endpoint
	if len(url) < len("https://example.com/auth") {
		t.Errorf("GetAuthURL() returned unexpected URL: %s", url)
	}
}

func TestManager_GetUserEmail(t *testing.T) {
	manager := NewManager()

	mockProvider := &MockProvider{
		name: "mock",
		config: &oauth2.Config{
			ClientID: "test",
		},
		userEmail: "user@example.com",
	}

	manager.AddProvider(mockProvider)

	token := &oauth2.Token{
		AccessToken: "test-token",
	}

	email, err := manager.GetUserEmail(context.Background(), "mock", token)
	if err != nil {
		t.Fatalf("GetUserEmail() error = %v", err)
	}

	if email != "user@example.com" {
		t.Errorf("GetUserEmail() = %s, want user@example.com", email)
	}
}

func TestGenerateState(t *testing.T) {
	state1, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error = %v", err)
	}

	if state1 == "" {
		t.Error("GenerateState() returned empty string")
	}

	// Generate another state and verify they're different
	state2, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() second call error = %v", err)
	}

	if state1 == state2 {
		t.Error("GenerateState() should return different values on each call")
	}

	// State should be reasonably long (base64 encoded 32 bytes)
	if len(state1) < 40 {
		t.Errorf("GenerateState() returned string of length %d, expected at least 40", len(state1))
	}
}
