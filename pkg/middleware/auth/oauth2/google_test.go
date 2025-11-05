package oauth2

import (
	"testing"
)

func TestNewGoogleProvider(t *testing.T) {
	provider := NewGoogleProvider("test-client-id", "test-client-secret", "http://localhost/callback", nil, false)

	if provider == nil {
		t.Fatal("NewGoogleProvider() returned nil")
	}

	if provider.Name() != "google" {
		t.Errorf("Name() = %s, want google", provider.Name())
	}

	config := provider.Config()
	if config.ClientID != "test-client-id" {
		t.Errorf("ClientID = %s, want test-client-id", config.ClientID)
	}

	if config.ClientSecret != "test-client-secret" {
		t.Errorf("ClientSecret = %s, want test-client-secret", config.ClientSecret)
	}

	// Should have default scopes
	expectedScopes := []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}
	if len(config.Scopes) != len(expectedScopes) {
		t.Errorf("Scopes length = %d, want %d", len(config.Scopes), len(expectedScopes))
	}
	for i, scope := range expectedScopes {
		if i >= len(config.Scopes) || config.Scopes[i] != scope {
			t.Errorf("Scopes = %v, want %v", config.Scopes, expectedScopes)
			break
		}
	}
}

func TestGoogleProvider_AddScopes(t *testing.T) {
	// Test adding scopes to default scopes (reset_scopes: false)
	customScopes := []string{"https://www.googleapis.com/auth/analytics.readonly"}
	provider := NewGoogleProvider("test-client-id", "test-client-secret", "http://localhost/callback", customScopes, false)

	config := provider.Config()

	// Should have default scopes + custom scopes
	expectedScopes := []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
		"https://www.googleapis.com/auth/analytics.readonly",
	}

	if len(config.Scopes) != len(expectedScopes) {
		t.Errorf("Scopes length = %d, want %d", len(config.Scopes), len(expectedScopes))
	}

	for i, scope := range expectedScopes {
		if i >= len(config.Scopes) || config.Scopes[i] != scope {
			t.Errorf("Scopes[%d] = %s, want %s", i, config.Scopes[i], scope)
		}
	}
}

func TestGoogleProvider_ResetScopes(t *testing.T) {
	// Test resetting scopes (reset_scopes: true)
	customScopes := []string{"https://www.googleapis.com/auth/analytics.readonly"}
	provider := NewGoogleProvider("test-client-id", "test-client-secret", "http://localhost/callback", customScopes, true)

	config := provider.Config()

	// Should have only custom scopes (default scopes replaced)
	expectedScopes := []string{
		"https://www.googleapis.com/auth/analytics.readonly",
	}

	if len(config.Scopes) != len(expectedScopes) {
		t.Errorf("Scopes length = %d, want %d", len(config.Scopes), len(expectedScopes))
	}

	for i, scope := range expectedScopes {
		if i >= len(config.Scopes) || config.Scopes[i] != scope {
			t.Errorf("Scopes[%d] = %s, want %s", i, config.Scopes[i], scope)
		}
	}
}

func TestGoogleProvider_ResetScopesEmpty(t *testing.T) {
	// Test resetting with empty scopes - should fallback to default
	provider := NewGoogleProvider("test-client-id", "test-client-secret", "http://localhost/callback", nil, true)

	config := provider.Config()

	// Should fallback to default scopes
	expectedScopes := []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}

	if len(config.Scopes) != len(expectedScopes) {
		t.Errorf("Scopes length = %d, want %d", len(config.Scopes), len(expectedScopes))
	}

	for i, scope := range expectedScopes {
		if i >= len(config.Scopes) || config.Scopes[i] != scope {
			t.Errorf("Scopes[%d] = %s, want %s", i, config.Scopes[i], scope)
		}
	}
}
