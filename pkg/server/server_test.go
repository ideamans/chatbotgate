package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ideamans/multi-oauth2-proxy/pkg/auth/email"
	"github.com/ideamans/multi-oauth2-proxy/pkg/auth/oauth2"
	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
	"github.com/ideamans/multi-oauth2-proxy/pkg/logging"
	"github.com/ideamans/multi-oauth2-proxy/pkg/proxy"
	"github.com/ideamans/multi-oauth2-proxy/pkg/session"
	oauth2lib "golang.org/x/oauth2"
)

// MockAuthzChecker is a mock authorization checker
type MockAuthzChecker struct {
	allowed bool
}

func (m *MockAuthzChecker) IsAllowed(email string) bool {
	return m.allowed
}

// MockOAuth2Provider is a mock OAuth2 provider
type MockOAuth2Provider struct {
	name      string
	userEmail string
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

func (m *MockOAuth2Provider) GetUserEmail(ctx context.Context, token *oauth2lib.Token) (string, error) {
	return m.userEmail, nil
}

func setupTestServer(t *testing.T) (*Server, *session.MemoryStore) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name:        "Test Service",
			Description: "Test Description",
		},
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 4180,
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

	server := New(cfg, sessionStore, oauthManager, nil, authzChecker, proxyHandler, logger)

	return server, sessionStore
}

func TestServer_HandleHealth(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handleHealth() status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "OK" {
		t.Errorf("handleHealth() body = %s, want OK", rec.Body.String())
	}
}

func TestServer_HandleReady(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handleReady() status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "READY" {
		t.Errorf("handleReady() body = %s, want READY", rec.Body.String())
	}
}

func TestServer_HandleLogin(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/_auth/login", nil)
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

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
	server, sessionStore := setupTestServer(t)

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

	server.router.ServeHTTP(rec, req)

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
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

	// Should redirect to login
	if rec.Code != http.StatusFound {
		t.Errorf("authMiddleware() status = %d, want %d", rec.Code, http.StatusFound)
	}

	location := rec.Header().Get("Location")
	if location != "/_auth/login" {
		t.Errorf("authMiddleware() redirect = %s, want /login", location)
	}
}

func TestServer_AuthMiddleware_ValidSession(t *testing.T) {
	server, sessionStore := setupTestServer(t)

	// Create a valid session
	sess := &session.Session{
		ID:            "valid-session",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		Authenticated: true,
	}
	sessionStore.Set("valid-session", sess)

	// Create a test backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	// Update proxy to point to test backend
	proxyHandler, _ := proxy.NewHandler(backend.URL)
	server.proxyHandler = proxyHandler

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "_test_session",
		Value: "valid-session",
	})
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("authMiddleware() with valid session status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestServer_AuthMiddleware_ExpiredSession(t *testing.T) {
	server, sessionStore := setupTestServer(t)

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

	server.router.ServeHTTP(rec, req)

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

	form := "email=user@example.com"
	req := httptest.NewRequest(http.MethodPost, "/_auth/email/send", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = http.NoBody
	req.PostForm = map[string][]string{"email": {"user@example.com"}}

	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("handleEmailSend() status = %d, want %d (body: %s)", rec.Code, http.StatusOK, form)
	}

	body := rec.Body.String()
	if !contains(body, "Check Your Email") && !contains(body, "メールを確認してください") {
		t.Error("handleEmailSend() should show success message")
	}
}

func TestServer_HandleEmailVerify_InvalidToken(t *testing.T) {
	server, _ := setupTestServerWithEmail(t)

	req := httptest.NewRequest(http.MethodGet, "/_auth/email/verify?token=invalid-token", nil)
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("handleEmailVerify() with invalid token status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	body := rec.Body.String()
	if !contains(body, "Invalid or Expired") && !contains(body, "無効または期限切れ") {
		t.Error("handleEmailVerify() should show error message for invalid token")
	}
}

func TestServer_HandleOAuth2Start(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/_auth/oauth2/start/google", nil)
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

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
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/_auth/oauth2/start/invalid-provider", nil)
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("handleOAuth2Start() with invalid provider status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServer_HandleLogin_WithEmailAuth(t *testing.T) {
	server, _ := setupTestServerWithEmail(t)

	req := httptest.NewRequest(http.MethodGet, "/_auth/login", nil)
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

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
	server, sessionStore := setupTestServer(t)

	// Step 1: Access a protected path without authentication
	req := httptest.NewRequest(http.MethodGet, "/protected/path", nil)
	rec := httptest.NewRecorder()

	server.router.ServeHTTP(rec, req)

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

	// Step 3: Call getRedirectURL to verify it returns the saved URL
	req2 := httptest.NewRequest(http.MethodGet, "/_auth/oauth2/callback", nil)
	req2.AddCookie(redirectCookie)
	rec2 := httptest.NewRecorder()

	// Get redirect URL
	redirectURL := server.getRedirectURL(rec2, req2)

	if redirectURL != "/protected/path" {
		t.Errorf("getRedirectURL() = %s, want /protected/path", redirectURL)
	}

	// Verify cookie is deleted
	deletedCookies := rec2.Result().Cookies()
	for _, c := range deletedCookies {
		if c.Name == "_oauth2_redirect" && c.MaxAge == -1 {
			// Cookie correctly deleted
			return
		}
	}
	t.Error("Redirect cookie should be deleted")
}

func TestServer_RedirectSecurity_OpenRedirect(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name        string
		redirectURL string
		want        string
	}{
		{
			name:        "Valid relative URL",
			redirectURL: "/protected/resource",
			want:        "/protected/resource",
		},
		{
			name:        "Absolute URL with scheme (should reject)",
			redirectURL: "http://evil.com/steal",
			want:        "/",
		},
		{
			name:        "Protocol-relative URL (should reject)",
			redirectURL: "//evil.com/steal",
			want:        "/",
		},
		{
			name:        "HTTPS URL (should reject)",
			redirectURL: "https://evil.com/steal",
			want:        "/",
		},
		{
			name:        "Empty URL",
			redirectURL: "",
			want:        "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.AddCookie(&http.Cookie{
				Name:  "_oauth2_redirect",
				Value: tt.redirectURL,
			})
			rec := httptest.NewRecorder()

			got := server.getRedirectURL(rec, req)
			if got != tt.want {
				t.Errorf("getRedirectURL() = %s, want %s", got, tt.want)
			}
		})
	}
}

func setupTestServerWithEmail(t *testing.T) (*Server, *session.MemoryStore) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name:        "Test Service",
			Description: "Test Description",
		},
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 4180,
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

	// Create email handler
	emailHandler, err := email.NewHandler(
		cfg.EmailAuth,
		cfg.Service.Name,
		"http://localhost:4180",
		cfg.Server.GetAuthPathPrefix(),
		authzChecker,
		cfg.Session.CookieSecret,
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

	server := New(cfg, sessionStore, oauthManager, emailHandler, authzChecker, proxyHandler, logger)

	return server, sessionStore
}
