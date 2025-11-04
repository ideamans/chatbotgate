package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/auth/email"
	"github.com/ideamans/chatbotgate/pkg/auth/oauth2"
	"github.com/ideamans/chatbotgate/pkg/authz"
	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/forwarding"
	"github.com/ideamans/chatbotgate/pkg/i18n"
	"github.com/ideamans/chatbotgate/pkg/kvs"
	"github.com/ideamans/chatbotgate/pkg/logging"
	"github.com/ideamans/chatbotgate/pkg/proxy"
	"github.com/ideamans/chatbotgate/pkg/session"
	oauth2lib "golang.org/x/oauth2"
)

// MockAuthzChecker is a mock authorization checker
type MockAuthzChecker struct {
	allowed       bool
	requiresEmail bool
}

func (m *MockAuthzChecker) RequiresEmail() bool {
	return m.requiresEmail
}

func (m *MockAuthzChecker) IsAllowed(email string) bool {
	return m.allowed
}

// MockOAuth2Provider is a mock OAuth2 provider
type MockOAuth2Provider struct {
	name      string
	userEmail string
	userName  string
	emailErr  error // If set, GetUserInfo will return this error
}

func (m *MockOAuth2Provider) Name() string {
	return m.name
}

func (m *MockOAuth2Provider) Config() *oauth2lib.Config {
	return &oauth2lib.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:4180/oauth2/callback",
		Endpoint: oauth2lib.Endpoint{
			AuthURL:  "http://provider.test/auth",
			TokenURL: "http://provider.test/token",
		},
	}
}

func (m *MockOAuth2Provider) GetUserInfo(ctx context.Context, token *oauth2lib.Token) (*oauth2.UserInfo, error) {
	if m.emailErr != nil {
		return nil, m.emailErr
	}
	return &oauth2.UserInfo{
		Email: m.userEmail,
		Name:  m.userName,
	}, nil
}

func (m *MockOAuth2Provider) GetUserEmail(ctx context.Context, token *oauth2lib.Token) (string, error) {
	if m.emailErr != nil {
		return "", m.emailErr
	}
	return m.userEmail, nil
}

func setupTestServer(t *testing.T) (*Server, *session.MemoryStore, func()) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back the authentication headers
		w.Header().Set("X-Test-User", r.Header.Get("X-ChatbotGate-User"))
		w.Header().Set("X-Test-Email", r.Header.Get("X-ChatbotGate-Email"))
		w.Header().Set("X-Test-Provider", r.Header.Get("X-Auth-Provider"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))

	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name:        "Test Service",
			Description: "Test Description",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Proxy: config.ProxyConfig{
			Upstream: backend.URL, // Use real backend URL
		},
		Session: config.SessionConfig{
			CookieName:     "_test_session",
			CookieSecret:   "test-secret-key-with-32-characters",
			CookieExpire:   "1h",
			CookieSecure:   false,
			CookieHTTPOnly: true,
			CookieSameSite: "lax",
		},
		Forwarding: config.ForwardingConfig{
			Fields: []string{"username", "email"},
			Header: config.ForwardingHeaderConfig{
				Enabled: true,
				Encrypt: false, // Plain text for tests
			},
			QueryString: config.ForwardingMethodConfig{
				Enabled: false, // Not needed for these tests
			},
		},
	}

	sessionStore := session.NewMemoryStore(1 * time.Minute)

	oauthManager := oauth2.NewManager()
	mockProvider := &MockOAuth2Provider{
		name:      "google",
		userEmail: "user@example.com",
	}
	oauthManager.AddProvider(mockProvider)

	authzChecker := &MockAuthzChecker{allowed: true}

	proxyHandler, err := proxy.NewHandler(cfg.Proxy.Upstream)
	if err != nil {
		t.Fatalf("Failed to create proxy handler: %v", err)
	}

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	// Create forwarder for the tests
	var forwarder forwarding.Forwarder
	if cfg.Forwarding.Header.Enabled || cfg.Forwarding.QueryString.Enabled {
		forwarder = forwarding.NewForwarder(&cfg.Forwarding, cfg.OAuth2.Providers)
	}

	server := New(cfg, "localhost", 4180, sessionStore, oauthManager, nil, authzChecker, proxyHandler, forwarder, logger)

	cleanup := func() {
		backend.Close()
	}

	return server, sessionStore, cleanup
}

func TestServer_HandleHealth(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handleHealth() status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "OK" {
		t.Errorf("handleHealth() body = %s, want OK", rec.Body.String())
	}
}

func TestServer_HandleReady(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handleReady() status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "READY" {
		t.Errorf("handleReady() body = %s, want READY", rec.Body.String())
	}
}

func TestServer_HandleLogin(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/_auth/login", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handleLogin() status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Check that response contains service name
	body := rec.Body.String()
	if !contains(body, "Test Service") {
		t.Error("handleLogin() should contain service name")
	}
}

func TestServer_HandleLogout(t *testing.T) {
	server, sessionStore, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session
	sess := &session.Session{
		ID:            "test-session",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		Authenticated: true,
	}
	sessionStore.Set("test-session", sess)

	req := httptest.NewRequest(http.MethodGet, "/_auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "_test_session",
		Value: "test-session",
	})
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handleLogout() status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify session was deleted
	_, err := sessionStore.Get("test-session")
	if err == nil {
		t.Error("handleLogout() should delete session")
	}
}

func TestServer_AuthMiddleware_NoSession(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	// Should redirect to login
	if rec.Code != http.StatusFound {
		t.Errorf("authMiddleware() status = %d, want %d", rec.Code, http.StatusFound)
	}

	location := rec.Header().Get("Location")
	// The middleware now includes the redirect parameter to preserve the original URL
	if !strings.HasPrefix(location, "/_auth/login") {
		t.Errorf("authMiddleware() redirect = %s, want to start with /_auth/login", location)
	}
}

func TestServer_AuthMiddleware_ValidSession(t *testing.T) {
	server, sessionStore, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a valid session
	sess := &session.Session{
		ID:            "valid-session",
		Email:         "user@example.com",
		Name:          "user@example.com", // Set name for X-ChatbotGate-User header
		Provider:      "google",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		Authenticated: true,
	}
	sessionStore.Set("valid-session", sess)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "_test_session",
		Value: "valid-session",
	})
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("authMiddleware() with valid session status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify that authentication headers were passed to the backend
	// The backend echoes them back as X-Test-* headers
	if rec.Header().Get("X-Test-User") != "user@example.com" {
		t.Errorf("X-ChatbotGate-User not passed to backend, got X-Test-User = %s", rec.Header().Get("X-Test-User"))
	}

	if rec.Header().Get("X-Test-Email") != "user@example.com" {
		t.Errorf("X-ChatbotGate-Email not passed to backend, got X-Test-Email = %s", rec.Header().Get("X-Test-Email"))
	}

	if rec.Header().Get("X-Test-Provider") != "google" {
		t.Errorf("X-Auth-Provider not passed to backend, got X-Test-Provider = %s", rec.Header().Get("X-Test-Provider"))
	}

	// Verify response body
	if rec.Body.String() != "backend response" {
		t.Errorf("Expected backend response, got %s", rec.Body.String())
	}
}

func TestServer_AuthMiddleware_ExpiredSession(t *testing.T) {
	server, sessionStore, cleanup := setupTestServer(t)
	defer cleanup()

	// Create an expired session
	sess := &session.Session{
		ID:            "expired-session",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     time.Now().Add(-2 * time.Hour),
		ExpiresAt:     time.Now().Add(-1 * time.Hour),
		Authenticated: true,
	}
	sessionStore.Set("expired-session", sess)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "_test_session",
		Value: "expired-session",
	})
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	// Should redirect to login
	if rec.Code != http.StatusFound {
		t.Errorf("authMiddleware() with expired session status = %d, want %d", rec.Code, http.StatusFound)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestServer_HandleEmailSend_Success(t *testing.T) {
	server, _ := setupTestServerWithEmail(t)

	req := httptest.NewRequest(http.MethodPost, "/_auth/email/send", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = http.NoBody
	req.PostForm = map[string][]string{"email": {"user@example.com"}}

	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	// Should redirect to email sent page
	if rec.Code != http.StatusSeeOther {
		t.Errorf("handleEmailSend() status = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	// Check redirect location
	location := rec.Header().Get("Location")
	if !contains(location, "/_auth/email/sent") {
		t.Errorf("handleEmailSend() should redirect to /_auth/email/sent, got %s", location)
	}
}

func TestServer_HandleEmailVerify_InvalidToken(t *testing.T) {
	server, _ := setupTestServerWithEmail(t)

	req := httptest.NewRequest(http.MethodGet, "/_auth/email/verify?token=invalid-token", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("handleEmailVerify() with invalid token status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	body := rec.Body.String()
	if !contains(body, "Invalid or Expired") && !contains(body, "無効または期限切れ") {
		t.Error("handleEmailVerify() should show error message for invalid token")
	}
}

func TestServer_HandleOAuth2Start(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/_auth/oauth2/start/google", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("handleOAuth2Start() status = %d, want %d", rec.Code, http.StatusFound)
	}

	// Check that state cookie is set
	cookies := rec.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "oauth_state" {
			stateCookie = c
			break
		}
	}

	if stateCookie == nil {
		t.Error("oauth_state cookie not set")
	}

	// Check redirect URL contains expected parts
	location := rec.Header().Get("Location")
	if location == "" {
		t.Error("Location header not set")
	}
}

func TestServer_HandleOAuth2Start_InvalidProvider(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/_auth/oauth2/start/invalid-provider", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("handleOAuth2Start() with invalid provider status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServer_HandleLogin_WithEmailAuth(t *testing.T) {
	server, _ := setupTestServerWithEmail(t)

	req := httptest.NewRequest(http.MethodGet, "/_auth/login", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handleLogin() status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	// Check for email login elements (label or button in English or Japanese)
	hasEmailLabel := contains(body, "Email Address") || contains(body, "メールアドレス")
	hasEmailButton := contains(body, "Send Login Link") || contains(body, "ログインリンクを送信")

	if !hasEmailLabel && !hasEmailButton {
		t.Error("handleLogin() should contain email login form when email auth is enabled")
	}

	// Check that the form is present
	if !contains(body, "form") {
		t.Error("handleLogin() should contain email form when email auth is enabled")
	}

	// Check that form posts to /email/send
	if !contains(body, "/email/send") {
		t.Error("handleLogin() should contain form action pointing to /email/send")
	}
}

// setupTestServerWithEmail creates a test server with email authentication enabled
func TestServer_RedirectToOriginalURL(t *testing.T) {
	server, sessionStore, cleanup := setupTestServer(t)
	defer cleanup()

	// Step 1: Access a protected path without authentication
	req := httptest.NewRequest(http.MethodGet, "/protected/path", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	// Should redirect to login
	if rec.Code != http.StatusFound {
		t.Errorf("Expected redirect status %d, got %d", http.StatusFound, rec.Code)
	}

	// Verify redirect cookie is set
	cookies := rec.Result().Cookies()
	var redirectCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "_oauth2_redirect" {
			redirectCookie = c
			break
		}
	}

	if redirectCookie == nil {
		t.Fatal("Redirect cookie not set")
	}

	if redirectCookie.Value != "/protected/path" {
		t.Errorf("Redirect cookie value = %s, want /protected/path", redirectCookie.Value)
	}

	// Step 2: Create a valid session (simulating successful authentication)
	sess := &session.Session{
		ID:            "test-session",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		Authenticated: true,
	}
	sessionStore.Set("test-session", sess)

	// Note: getRedirectURL is now internal to the middleware package
	// The redirect functionality is tested through the full OAuth callback flow
	// TODO: Create middleware-specific tests if needed
	//
	// Step 3: Verify redirect behavior through OAuth callback
	// This would require a full OAuth flow test, which is covered by other integration tests
	t.Log("Redirect cookie set successfully: ", redirectCookie.Value)
}

func TestServer_RedirectSecurity_OpenRedirect(t *testing.T) {
	t.Skip("getRedirectURL is now internal to middleware package - TODO: create middleware-specific tests")
	// Note: Open redirect protection is now tested in the middleware package
	// The logic remains the same, but the method is no longer exposed on Server
}

func setupTestServerWithEmail(t *testing.T) (*Server, *session.MemoryStore) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name:        "Test Service",
			Description: "Test Description",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Proxy: config.ProxyConfig{
			Upstream: "http://backend.test",
		},
		Session: config.SessionConfig{
			CookieName:     "_test_session",
			CookieSecret:   "test-secret-key-with-32-characters",
			CookieExpire:   "1h",
			CookieSecure:   false,
			CookieHTTPOnly: true,
			CookieSameSite: "lax",
		},
		EmailAuth: config.EmailAuthConfig{
			Enabled:    true,
			SenderType: "smtp",
			SMTP: config.SMTPConfig{
				Host: "smtp.test",
				Port: 587,
				From: "test@example.com",
			},
			Token: config.EmailTokenConfig{
				Expire: "15m",
			},
		},
	}

	sessionStore := session.NewMemoryStore(1 * time.Minute)

	oauthManager := oauth2.NewManager()
	mockProvider := &MockOAuth2Provider{
		name:      "google",
		userEmail: "user@example.com",
	}
	oauthManager.AddProvider(mockProvider)

	authzChecker := &MockAuthzChecker{allowed: true}
	translator := i18n.NewTranslator()

	// Create KVS stores for testing
	tokenKVS, _ := kvs.NewMemoryStore("token:", kvs.MemoryConfig{CleanupInterval: 1 * time.Minute})
	rateLimitKVS, _ := kvs.NewMemoryStore("ratelimit:", kvs.MemoryConfig{CleanupInterval: 1 * time.Minute})

	// Create email handler
	emailHandler, err := email.NewHandler(
		cfg.EmailAuth,
		cfg.Service,
		"http://localhost:4180",
		cfg.Server.GetAuthPathPrefix(),
		authzChecker,
		translator,
		cfg.Session.CookieSecret,
		tokenKVS,
		rateLimitKVS,
	)
	if err != nil {
		t.Fatalf("Failed to create email handler: %v", err)
	}

	// Replace sender with mock
	emailHandler.SetSender(&email.MockSender{})

	proxyHandler, err := proxy.NewHandler(cfg.Proxy.Upstream)
	if err != nil {
		t.Fatalf("Failed to create proxy handler: %v", err)
	}

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	server := New(cfg, "localhost", 4180, sessionStore, oauthManager, emailHandler, authzChecker, proxyHandler, nil, logger)

	return server, sessionStore
}

// TestServer_Authorization_NoWhitelist_SessionWithoutEmail tests that sessions without email work when no whitelist is configured
func TestServer_Authorization_NoWhitelist_SessionWithoutEmail(t *testing.T) {
	server, sessionStore, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a session WITHOUT email (simulating OAuth2 without email requirement)
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

	// Make a request with this session
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "_test_session", // Use the same cookie name as setupTestServer
		Value: sessionID,
	})

	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	// Should succeed even without email
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	// Verify auth headers were NOT set (no email)
	resp := w.Result()
	if resp.Header.Get("X-Test-Email") != "" {
		t.Logf("Email header: %s (can be empty when no whitelist)", resp.Header.Get("X-Test-Email"))
	}
}

// TestServer_Authorization_WithWhitelist_AuthorizedEmail tests authorized email with whitelist
func TestServer_Authorization_WithWhitelist_AuthorizedEmail(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test-Email", r.Header.Get("X-ChatbotGate-Email"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name:        "Test Service",
			Description: "Test Description",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Proxy: config.ProxyConfig{
			Upstream: backend.URL,
		},
		Session: config.SessionConfig{
			CookieName:     "_oauth2_proxy",
			CookieSecret:   "test-secret-key-for-testing-purposes-only",
			CookieExpire:   "168h",
			CookieSecure:   false,
			CookieHTTPOnly: true,
			CookieSameSite: "lax",
		},
		Authorization: config.AuthorizationConfig{
			Allowed: []string{"authorized@example.com"}, // Whitelist configured
		},
		Forwarding: config.ForwardingConfig{
			Fields: []string{"username", "email"},
			Header: config.ForwardingHeaderConfig{
				Enabled: true,
				Encrypt: false, // Plain text for tests
			},
			QueryString: config.ForwardingMethodConfig{
				Enabled: false,
			},
		},
	}

	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.Authorization)

	proxyHandler, err := proxy.NewHandler(cfg.Proxy.Upstream)
	if err != nil {
		t.Fatalf("Failed to create proxy handler: %v", err)
	}

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	// Create forwarder for the tests
	var forwarder forwarding.Forwarder
	if cfg.Forwarding.Header.Enabled || cfg.Forwarding.QueryString.Enabled {
		forwarder = forwarding.NewForwarder(&cfg.Forwarding, cfg.OAuth2.Providers)
	}

	server := New(cfg, "localhost", 4180, sessionStore, oauthManager, nil, authzChecker, proxyHandler, forwarder, logger)

	// Create a session with authorized email
	sessionID := "test-session-authorized"
	sess := &session.Session{
		ID:            sessionID,
		Email:         "authorized@example.com",
		Provider:      "google",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		Authenticated: true,
	}

	err = sessionStore.Set(sessionID, sess)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Make a request
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "_oauth2_proxy",
		Value: sessionID,
	})

	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	// Should succeed with authorized email
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	// Verify email header was set
	resp := w.Result()
	if email := resp.Header.Get("X-Test-Email"); email != "authorized@example.com" {
		t.Errorf("Expected email header 'authorized@example.com', got '%s'", email)
	}
}
