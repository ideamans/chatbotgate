package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	stdoauth2 "golang.org/x/oauth2"

	"github.com/ideamans/multi-oauth2-proxy/pkg/auth/oauth2"
	"github.com/ideamans/multi-oauth2-proxy/pkg/authz"
	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
	"github.com/ideamans/multi-oauth2-proxy/pkg/i18n"
	"github.com/ideamans/multi-oauth2-proxy/pkg/logging"
	"github.com/ideamans/multi-oauth2-proxy/pkg/session"
)

// mockProvider is a mock implementation of oauth2.Provider
type mockProvider struct {
	name          string
	emailToReturn string
	emailError    error
}

func (p *mockProvider) Name() string {
	return p.name
}

func (p *mockProvider) Config() *stdoauth2.Config {
	return &stdoauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint: stdoauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: "https://example.com/token",
		},
	}
}

func (p *mockProvider) GetUserEmail(ctx context.Context, token *stdoauth2.Token) (string, error) {
	if p.emailError != nil {
		return "", p.emailError
	}
	return p.emailToReturn, nil
}

// TestMiddleware_RequiresEmail tests the RequiresEmail logic in middleware
func TestMiddleware_RequiresEmail(t *testing.T) {
	tests := []struct {
		name           string
		authzConfig    config.AuthorizationConfig
		expectRequired bool
	}{
		{
			name: "no whitelist - email not required",
			authzConfig: config.AuthorizationConfig{
				Allowed: []string{},
			},
			expectRequired: false,
		},
		{
			name: "with allowed emails - email required",
			authzConfig: config.AuthorizationConfig{
				Allowed: []string{"user@example.com"},
			},
			expectRequired: true,
		},
		{
			name: "with allowed domains - email required",
			authzConfig: config.AuthorizationConfig{
				Allowed: []string{"@example.com"},
			},
			expectRequired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Service: config.ServiceConfig{
					Name: "Test Service",
				},
				Server: config.ServerConfig{
					AuthPathPrefix: "/_auth",
				},
				Session: config.SessionConfig{
					CookieName:   "_test",
					CookieSecure: false,
				},
				Authorization: tt.authzConfig,
			}

			sessionStore := session.NewMemoryStore(1 * time.Minute)
			defer sessionStore.Close()

			oauthManager := oauth2.NewManager()
			oauthManager.AddProvider(&mockProvider{
				name:          "google",
				emailToReturn: "user@example.com",
			})

			authzChecker := authz.NewEmailChecker(tt.authzConfig)
			translator := i18n.NewTranslator()
			logger := logging.NewTestLogger()

			middleware := New(
				cfg,
				sessionStore,
				oauthManager,
				nil, // email handler
				authzChecker,
				translator,
				logger,
			)

			if middleware.authzChecker.RequiresEmail() != tt.expectRequired {
				t.Errorf("RequiresEmail() = %v, want %v", middleware.authzChecker.RequiresEmail(), tt.expectRequired)
			}
		})
	}
}

// TestMiddleware_Authorization_NoWhitelist tests authorization when no whitelist is configured
func TestMiddleware_Authorization_NoWhitelist(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			CookieName:     "_test",
			CookieExpire:   "24h",
			CookieSecure:   false,
			CookieHTTPOnly: true,
		},
		Authorization: config.AuthorizationConfig{
			Allowed: []string{}, // No whitelist
		},
	}

	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.Authorization)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware := New(
		cfg,
		sessionStore,
		oauthManager,
		nil,
		authzChecker,
		translator,
		logger,
	)

	// When no whitelist is configured, RequiresEmail should return false
	if middleware.authzChecker.RequiresEmail() {
		t.Error("RequiresEmail() should return false with no whitelist")
	}

	// Any email should be allowed when no whitelist
	if !middleware.authzChecker.IsAllowed("any@email.com") {
		t.Error("IsAllowed() should return true for any email when no whitelist")
	}

	if !middleware.authzChecker.IsAllowed("") {
		t.Error("IsAllowed() should return true even for empty email when no whitelist")
	}

	// Create a session without email (simulating authentication without email requirement)
	sessionID := "test-session-no-email"
	sess := &session.Session{
		ID:            sessionID,
		Email:         "", // No email when whitelist not configured
		Provider:      "google",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		Authenticated: true,
	}

	err := sessionStore.Set(sessionID, sess)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create a request with this session
	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "_test",
		Value: sessionID,
	})

	w := httptest.NewRecorder()

	// Create a simple handler that the middleware will call
	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrappedHandler := middleware.Wrap(nextHandler)
	wrappedHandler.ServeHTTP(w, req)

	// Request should succeed even though session has no email
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	if !nextCalled {
		t.Error("Expected next handler to be called")
	}
}

// TestMiddleware_Authorization_WithWhitelist tests authorization when whitelist is configured
func TestMiddleware_Authorization_WithWhitelist(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			CookieName:   "_test",
			CookieExpire: "24h",
			CookieSecure: false,
		},
		Authorization: config.AuthorizationConfig{
			Allowed: []string{"authorized@example.com"}, // Whitelist configured
		},
	}

	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.Authorization)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware := New(
		cfg,
		sessionStore,
		oauthManager,
		nil,
		authzChecker,
		translator,
		logger,
	)

	// When whitelist is configured, RequiresEmail should return true
	if !middleware.authzChecker.RequiresEmail() {
		t.Error("RequiresEmail() should return true with whitelist configured")
	}

	// Authorized email should be allowed
	if !middleware.authzChecker.IsAllowed("authorized@example.com") {
		t.Error("IsAllowed() should return true for authorized email")
	}

	// Unauthorized email should NOT be allowed
	if middleware.authzChecker.IsAllowed("unauthorized@example.com") {
		t.Error("IsAllowed() should return false for unauthorized email")
	}

	// Empty email should NOT be allowed when whitelist is configured
	if middleware.authzChecker.IsAllowed("") {
		t.Error("IsAllowed() should return false for empty email when whitelist is configured")
	}

	// Test 1: Create a session with authorized email - should work
	authorizedSessionID := "test-session-authorized"
	authorizedSess := &session.Session{
		ID:            authorizedSessionID,
		Email:         "authorized@example.com",
		Provider:      "google",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		Authenticated: true,
	}

	err := sessionStore.Set(authorizedSessionID, authorizedSess)
	if err != nil {
		t.Fatalf("Failed to create authorized session: %v", err)
	}

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "_test",
		Value: authorizedSessionID,
	})

	w := httptest.NewRecorder()

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrappedHandler := middleware.Wrap(nextHandler)
	wrappedHandler.ServeHTTP(w, req)

	// Request should succeed for authorized email
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for authorized email, got %d", w.Code)
	}

	if !nextCalled {
		t.Error("Expected next handler to be called for authorized email")
	}
}
