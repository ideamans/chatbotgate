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
		"openid",
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

func TestGoogleProvider_CustomScopes(t *testing.T) {
	// Test with custom scopes - should use only custom scopes (no defaults added)
	customScopes := []string{"https://www.googleapis.com/auth/analytics.readonly"}
	provider := NewGoogleProvider("test-client-id", "test-client-secret", "http://localhost/callback", customScopes, false)

	config := provider.Config()

	// Should have only custom scopes (defaults not added)
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

func TestGoogleProvider_CustomScopesWithResetFlag(t *testing.T) {
	// Test with custom scopes and reset_scopes: true
	// Behavior is same as reset_scopes: false (only custom scopes are used)
	customScopes := []string{"https://www.googleapis.com/auth/analytics.readonly"}
	provider := NewGoogleProvider("test-client-id", "test-client-secret", "http://localhost/callback", customScopes, true)

	config := provider.Config()

	// Should have only custom scopes
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

func TestGoogleProvider_EmptyScopes(t *testing.T) {
	// Test with empty scopes - should use default scopes
	provider := NewGoogleProvider("test-client-id", "test-client-secret", "http://localhost/callback", nil, true)

	config := provider.Config()

	// Should use default scopes when scopes are empty
	expectedScopes := []string{
		"openid",
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
