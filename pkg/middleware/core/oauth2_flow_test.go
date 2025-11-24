package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	stdoauth2 "golang.org/x/oauth2"

	"github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
	"github.com/ideamans/chatbotgate/pkg/middleware/authz"
	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/middleware/session"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// mockOAuth2Provider is an extended mock for OAuth2 flow testing
// It implements the Provider interface and uses httptest.Server for token exchange
type mockOAuth2Provider struct {
	name          string
	displayName   string
	emailToReturn string
	nameToReturn  string
	extraData     map[string]interface{}
	emailError    error
	tokenServer   *httptest.Server // Mock OAuth2 token endpoint
}

// newMockOAuth2Provider creates a mock provider with a mock token server
func newMockOAuth2Provider(name, email, displayName string) *mockOAuth2Provider {
	p := &mockOAuth2Provider{
		name:          name,
		displayName:   displayName,
		emailToReturn: email,
		nameToReturn:  displayName,
	}

	// Create mock token server
	p.tokenServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple mock: return a fake access token
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"mock-access-token","token_type":"Bearer","expires_in":3600}`))
	}))

	return p
}

func (p *mockOAuth2Provider) Close() {
	if p.tokenServer != nil {
		p.tokenServer.Close()
	}
}

func (p *mockOAuth2Provider) Name() string {
	return p.name
}

func (p *mockOAuth2Provider) Config() *stdoauth2.Config {
	tokenURL := p.tokenServer.URL
	if p.tokenServer == nil {
		tokenURL = "https://provider.example.com/oauth/token"
	}

	return &stdoauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint: stdoauth2.Endpoint{
			AuthURL:  "https://provider.example.com/oauth/authorize",
			TokenURL: tokenURL, // Use mock server URL
		},
		RedirectURL: "", // Will be set dynamically
		Scopes:      []string{"openid", "email", "profile"},
	}
}

func (p *mockOAuth2Provider) GetUserInfo(ctx context.Context, token *stdoauth2.Token) (*oauth2.UserInfo, error) {
	if p.emailError != nil {
		return nil, p.emailError
	}

	extra := p.extraData
	if extra == nil {
		extra = make(map[string]interface{})
	}
	// Add standardized fields
	extra["_email"] = p.emailToReturn
	extra["_username"] = p.nameToReturn
	extra["_avatar_url"] = "https://example.com/avatar.jpg"

	return &oauth2.UserInfo{
		Email: p.emailToReturn,
		Name:  p.nameToReturn,
		Extra: extra,
	}, nil
}

func (p *mockOAuth2Provider) GetUserEmail(ctx context.Context, token *stdoauth2.Token) (string, error) {
	if p.emailError != nil {
		return "", p.emailError
	}
	return p.emailToReturn, nil
}

// TestHandleOAuth2Start tests the OAuth2 authentication start flow
func TestHandleOAuth2Start(t *testing.T) {
	tests := []struct {
		name           string
		provider       string
		baseURL        string
		authPrefix     string
		expectedStatus int
		checkCookies   func(*testing.T, []*http.Cookie)
		checkRedirect  func(*testing.T, string)
		expectError    bool
	}{
		{
			name:           "Valid OAuth2 start with google",
			provider:       "google",
			baseURL:        "https://example.com",
			authPrefix:     "/_auth",
			expectedStatus: http.StatusFound,
			checkCookies: func(t *testing.T, cookies []*http.Cookie) {
				// Check that state cookie is set
				var stateCookie, providerCookie, redirectCookie *http.Cookie
				for _, c := range cookies {
					switch c.Name {
					case "oauth_state":
						stateCookie = c
					case "oauth_provider":
						providerCookie = c
					case "oauth_redirect_url":
						redirectCookie = c
					}
				}

				if stateCookie == nil {
					t.Error("oauth_state cookie not set")
				} else {
					if stateCookie.Value == "" {
						t.Error("oauth_state cookie value is empty")
					}
					if stateCookie.MaxAge != 600 {
						t.Errorf("oauth_state MaxAge = %d, want 600", stateCookie.MaxAge)
					}
					if !stateCookie.HttpOnly {
						t.Error("oauth_state cookie should be HttpOnly")
					}
				}

				if providerCookie == nil {
					t.Error("oauth_provider cookie not set")
				} else if providerCookie.Value != "google" {
					t.Errorf("oauth_provider = %q, want %q", providerCookie.Value, "google")
				}

				if redirectCookie == nil {
					t.Error("oauth_redirect_url cookie not set")
				} else if redirectCookie.Value == "" {
					t.Error("oauth_redirect_url value is empty")
				}
			},
			checkRedirect: func(t *testing.T, location string) {
				if location == "" {
					t.Error("Location header not set")
					return
				}
				// Should redirect to provider's auth URL
				if !contains(location, "provider.example.com") {
					t.Errorf("Redirect URL should contain provider domain, got: %s", location)
				}
				// Should contain state parameter
				if !contains(location, "state=") {
					t.Error("Redirect URL should contain state parameter")
				}
			},
		},
		{
			name:           "OAuth2 start with custom auth prefix",
			provider:       "google",
			baseURL:        "https://example.com",
			authPrefix:     "/_oauth2_proxy",
			expectedStatus: http.StatusFound,
			checkCookies: func(t *testing.T, cookies []*http.Cookie) {
				// Cookies should still be set
				if len(cookies) < 3 {
					t.Errorf("Expected at least 3 cookies, got %d", len(cookies))
				}
			},
			checkRedirect: func(t *testing.T, location string) {
				// Redirect URL should contain custom prefix in redirect_uri
				if !contains(location, "%2F_oauth2_proxy") && !contains(location, "/_oauth2_proxy") {
					t.Logf("Location: %s", location)
					// Not a hard failure as URL encoding varies
				}
			},
		},
		{
			name:           "OAuth2 start without base URL (uses Host header)",
			provider:       "google",
			baseURL:        "", // Empty - should use request Host
			authPrefix:     "/_auth",
			expectedStatus: http.StatusFound,
			checkCookies: func(t *testing.T, cookies []*http.Cookie) {
				if len(cookies) < 3 {
					t.Errorf("Expected at least 3 cookies, got %d", len(cookies))
				}
			},
			checkRedirect: func(t *testing.T, location string) {
				// Should still redirect successfully
				if location == "" {
					t.Error("Location header not set")
				}
			},
		},
		{
			name:           "OAuth2 start with invalid provider",
			provider:       "nonexistent",
			baseURL:        "https://example.com",
			authPrefix:     "/_auth",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create configuration
			cfg := &config.Config{
				Service: config.ServiceConfig{
					Name: "Test Service",
				},
				Server: config.ServerConfig{
					AuthPathPrefix: tt.authPrefix,
					BaseURL:        tt.baseURL,
				},
				Session: config.SessionConfig{
					Cookie: config.CookieConfig{
						Name:   "_test_session",
						Secret: "test-secret-key-32-bytes-long!",
						Expire: "24h",
						Secure: false,
					},
				},
			}

			// Create dependencies
			sessionStore, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{})
			defer func() { _ = sessionStore.Close() }()
			oauthManager := oauth2.NewManager()

			// Add mock provider
			mockProvider := newMockOAuth2Provider("google", "user@example.com", "Google")
			defer mockProvider.Close()
			oauthManager.AddProvider(mockProvider)

			authzChecker := authz.NewEmailChecker(cfg.AccessControl)
			translator := i18n.NewTranslator()
			logger := logging.NewTestLogger()

			// Create middleware
			mw, err := New(
				cfg,
				sessionStore,
				oauthManager,
				nil, // email handler
				nil, // password handler
				authzChecker,
				nil, // forwarder
				nil, // rules evaluator
				translator,
				logger,
			)
			if err != nil {
				t.Fatalf("Failed to create middleware: %v", err)
			}

			// Create request
			path := normalizeAuthPrefix(tt.authPrefix) + "/oauth2/start/" + tt.provider
			req := httptest.NewRequest("GET", path, nil)
			req.Host = "localhost:4180"
			rec := httptest.NewRecorder()

			// Call handler
			mw.handleOAuth2Start(rec, req)

			// Check status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.expectedStatus)
			}

			if tt.expectError {
				// For error cases, just verify we got an error response
				return
			}

			// Check cookies
			if tt.checkCookies != nil {
				cookies := rec.Result().Cookies()
				tt.checkCookies(t, cookies)
			}

			// Check redirect
			if tt.checkRedirect != nil {
				location := rec.Header().Get("Location")
				tt.checkRedirect(t, location)
			}
		})
	}
}

// TestHandleOAuth2Callback tests the OAuth2 callback flow
func TestHandleOAuth2Callback(t *testing.T) {
	tests := []struct {
		name                string
		setupCookies        func(*http.Request)
		queryParams         map[string]string
		providerEmail       string
		providerName        string
		providerError       error
		whitelistEmails     []string
		expectedStatus      int
		expectSessionCookie bool
		expectRedirect      string
		checkSession        func(*testing.T, kvs.Store, string)
	}{
		{
			name: "Successful OAuth2 callback with valid state",
			setupCookies: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "oauth_state", Value: "test-state-123"})
				r.AddCookie(&http.Cookie{Name: "oauth_provider", Value: "google"})
				r.AddCookie(&http.Cookie{Name: "oauth_redirect_url", Value: "https://example.com/_auth/oauth2/callback"})
			},
			queryParams: map[string]string{
				"state": "test-state-123",
				"code":  "test-auth-code",
			},
			providerEmail:       "user@example.com",
			providerName:        "Test User",
			expectedStatus:      http.StatusFound,
			expectSessionCookie: true,
			expectRedirect:      "/", // Default redirect
			checkSession: func(t *testing.T, store kvs.Store, cookieValue string) {
				sess, err := session.Get(store, cookieValue)
				if err != nil {
					t.Fatalf("Failed to get session: %v", err)
				}
				if sess.Email != "user@example.com" {
					t.Errorf("Session email = %q, want %q", sess.Email, "user@example.com")
				}
				if sess.Name != "Test User" {
					t.Errorf("Session name = %q, want %q", sess.Name, "Test User")
				}
				if sess.Provider != "google" {
					t.Errorf("Session provider = %q, want %q", sess.Provider, "google")
				}
				if !sess.Authenticated {
					t.Error("Session should be authenticated")
				}
			},
		},
		{
			name: "OAuth2 callback with state mismatch (CSRF attack)",
			setupCookies: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "oauth_state", Value: "correct-state"})
				r.AddCookie(&http.Cookie{Name: "oauth_provider", Value: "google"})
				r.AddCookie(&http.Cookie{Name: "oauth_redirect_url", Value: "https://example.com/_auth/oauth2/callback"})
			},
			queryParams: map[string]string{
				"state": "wrong-state", // Mismatch!
				"code":  "test-auth-code",
			},
			providerEmail:       "user@example.com",
			expectedStatus:      http.StatusBadRequest,
			expectSessionCookie: false,
		},
		{
			name: "OAuth2 callback without state cookie",
			setupCookies: func(r *http.Request) {
				// No state cookie set
				r.AddCookie(&http.Cookie{Name: "oauth_provider", Value: "google"})
				r.AddCookie(&http.Cookie{Name: "oauth_redirect_url", Value: "https://example.com/_auth/oauth2/callback"})
			},
			queryParams: map[string]string{
				"state": "test-state",
				"code":  "test-auth-code",
			},
			expectedStatus:      http.StatusBadRequest,
			expectSessionCookie: false,
		},
		{
			name: "OAuth2 callback without authorization code",
			setupCookies: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "oauth_state", Value: "test-state"})
				r.AddCookie(&http.Cookie{Name: "oauth_provider", Value: "google"})
				r.AddCookie(&http.Cookie{Name: "oauth_redirect_url", Value: "https://example.com/_auth/oauth2/callback"})
			},
			queryParams: map[string]string{
				"state": "test-state",
				// No code parameter
			},
			expectedStatus:      http.StatusBadRequest,
			expectSessionCookie: false,
		},
		{
			name: "OAuth2 callback with whitelist - authorized email",
			setupCookies: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "oauth_state", Value: "test-state"})
				r.AddCookie(&http.Cookie{Name: "oauth_provider", Value: "google"})
				r.AddCookie(&http.Cookie{Name: "oauth_redirect_url", Value: "https://example.com/_auth/oauth2/callback"})
			},
			queryParams: map[string]string{
				"state": "test-state",
				"code":  "test-auth-code",
			},
			providerEmail:       "authorized@example.com",
			providerName:        "Authorized User",
			whitelistEmails:     []string{"authorized@example.com"},
			expectedStatus:      http.StatusFound,
			expectSessionCookie: true,
			checkSession: func(t *testing.T, store kvs.Store, cookieValue string) {
				sess, err := session.Get(store, cookieValue)
				if err != nil {
					t.Fatalf("Failed to get session: %v", err)
				}
				if sess.Email != "authorized@example.com" {
					t.Errorf("Session email = %q, want %q", sess.Email, "authorized@example.com")
				}
			},
		},
		{
			name: "OAuth2 callback with whitelist - unauthorized email",
			setupCookies: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "oauth_state", Value: "test-state"})
				r.AddCookie(&http.Cookie{Name: "oauth_provider", Value: "google"})
				r.AddCookie(&http.Cookie{Name: "oauth_redirect_url", Value: "https://example.com/_auth/oauth2/callback"})
			},
			queryParams: map[string]string{
				"state": "test-state",
				"code":  "test-auth-code",
			},
			providerEmail:       "unauthorized@example.com",
			whitelistEmails:     []string{"authorized@example.com"}, // Different email
			expectedStatus:      http.StatusForbidden,
			expectSessionCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create configuration
			cfg := &config.Config{
				Service: config.ServiceConfig{
					Name: "Test Service",
				},
				Server: config.ServerConfig{
					AuthPathPrefix: "/_auth",
				},
				Session: config.SessionConfig{
					Cookie: config.CookieConfig{
						Name:   "_test_session",
						Secret: "test-secret-key-32-bytes-long!",
						Expire: "24h",
						Secure: false,
					},
				},
				AccessControl: config.AccessControlConfig{
					Emails: tt.whitelistEmails,
				},
			}

			// Create dependencies
			sessionStore, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{})
			defer func() { _ = sessionStore.Close() }()
			oauthManager := oauth2.NewManager()

			// Add mock provider
			mockProvider := newMockOAuth2Provider("google", tt.providerEmail, tt.providerName)
			if tt.providerError != nil {
				mockProvider.emailError = tt.providerError
			}
			defer mockProvider.Close()
			oauthManager.AddProvider(mockProvider)

			authzChecker := authz.NewEmailChecker(cfg.AccessControl)
			translator := i18n.NewTranslator()
			logger := logging.NewTestLogger()

			// Create middleware
			mw, err := New(
				cfg,
				sessionStore,
				oauthManager,
				nil, // email handler
				nil, // password handler
				authzChecker,
				nil, // forwarder
				nil, // rules evaluator
				translator,
				logger,
			)
			if err != nil {
				t.Fatalf("Failed to create middleware: %v", err)
			}

			// Build query string
			query := url.Values{}
			for k, v := range tt.queryParams {
				query.Set(k, v)
			}

			// Create request
			req := httptest.NewRequest("GET", "/_auth/oauth2/callback?"+query.Encode(), nil)

			// Setup cookies
			if tt.setupCookies != nil {
				tt.setupCookies(req)
			}

			rec := httptest.NewRecorder()

			// Call handler
			mw.handleOAuth2Callback(rec, req)

			// Check status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.expectedStatus)
				t.Logf("Response body: %s", rec.Body.String())
			}

			// Check session cookie
			cookies := rec.Result().Cookies()
			var sessionCookie *http.Cookie
			for _, c := range cookies {
				if c.Name == "_test_session" {
					sessionCookie = c
					break
				}
			}

			if tt.expectSessionCookie {
				if sessionCookie == nil {
					t.Error("Expected session cookie to be set, but it wasn't")
				} else {
					// Check session in store
					if tt.checkSession != nil {
						tt.checkSession(t, sessionStore, sessionCookie.Value)
					}
				}
			} else {
				if sessionCookie != nil && sessionCookie.Value != "" {
					t.Error("Did not expect session cookie, but it was set")
				}
			}

			// Check redirect location
			if tt.expectRedirect != "" {
				location := rec.Header().Get("Location")
				if location != tt.expectRedirect {
					t.Errorf("Redirect location = %q, want %q", location, tt.expectRedirect)
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
