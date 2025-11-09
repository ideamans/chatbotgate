package password

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/middleware/session"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// createTestSessionStore creates a memory-based session store for testing
func createTestSessionStore() kvs.Store {
	kvsStore, _ := kvs.NewMemoryStore("session:", kvs.MemoryConfig{
		CleanupInterval: 1 * time.Minute,
	})
	return kvsStore
}

// testCookieConfig returns a default CookieConfig for testing
func testCookieConfig() config.CookieConfig {
	return config.CookieConfig{
		Name:     "test-session",
		Secret:   "test-secret-32-characters-long",
		Expire:   "24h",
		Secure:   false,
		HTTPOnly: true,
		SameSite: "lax",
	}
}

// testTranslator returns a default Translator for testing
func testTranslator() *i18n.Translator {
	return i18n.NewTranslator()
}

// testLogger returns a default Logger for testing
func testLogger() logging.Logger {
	return logging.NewSimpleLogger("password-test", logging.LevelError, false)
}

func TestNewHandler(t *testing.T) {
	cfg := config.PasswordAuthConfig{
		Enabled:  true,
		Password: "test-password",
	}

	sessionStore := createTestSessionStore()
	cookieConfig := testCookieConfig()

	handler := NewHandler(cfg, sessionStore, cookieConfig, "/_auth", testTranslator(), testLogger())

	if handler == nil {
		t.Fatal("NewHandler() returned nil")
	}

	if handler.config.Password != "test-password" {
		t.Errorf("Handler password = %v, want %v", handler.config.Password, "test-password")
	}
}

func TestHandleLogin_Success(t *testing.T) {
	cfg := config.PasswordAuthConfig{
		Enabled:  true,
		Password: "correct-password",
	}

	sessionStore := createTestSessionStore()
	handler := NewHandler(cfg, sessionStore, testCookieConfig(), "/_auth", testTranslator(), testLogger())

	// Create request with correct password
	reqBody := map[string]string{
		"password": "correct-password",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/_auth/password/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleLogin(w, req)

	// Should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("HandleLogin() status = %v, want %v", w.Code, http.StatusOK)
	}

	// Should return JSON with redirect_url
	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if redirectURL, ok := response["redirect_url"]; !ok || redirectURL == "" {
		t.Error("Response should contain redirect_url")
	}

	// Should set session cookie
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Error("No cookies set")
	}

	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "test-session" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Error("Session cookie not set")
	}

	// Verify session was created in store
	if sessionCookie != nil {
		sess, err := session.Get(sessionStore, sessionCookie.Value)
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}

		if sess.Email != "password@localhost" {
			t.Errorf("Session email = %v, want %v", sess.Email, "password@localhost")
		}

		if sess.Provider != "password" {
			t.Errorf("Session provider = %v, want %v", sess.Provider, "password")
		}

		if !sess.Authenticated {
			t.Error("Session should be authenticated")
		}
	}
}

func TestHandleLogin_WrongPassword(t *testing.T) {
	cfg := config.PasswordAuthConfig{
		Enabled:  true,
		Password: "correct-password",
	}

	handler := NewHandler(cfg, createTestSessionStore(), testCookieConfig(), "/_auth", testTranslator(), testLogger())

	// Create request with wrong password
	reqBody := map[string]string{
		"password": "wrong-password",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/_auth/password/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleLogin(w, req)

	// Should return 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("HandleLogin() status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleLogin_EmptyPassword(t *testing.T) {
	cfg := config.PasswordAuthConfig{
		Enabled:  true,
		Password: "correct-password",
	}

	handler := NewHandler(cfg, createTestSessionStore(), testCookieConfig(), "/_auth", testTranslator(), testLogger())

	// Create request with empty password
	reqBody := map[string]string{
		"password": "",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/_auth/password/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleLogin(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleLogin() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandleLogin_MissingPasswordField(t *testing.T) {
	cfg := config.PasswordAuthConfig{
		Enabled:  true,
		Password: "correct-password",
	}

	handler := NewHandler(cfg, createTestSessionStore(), testCookieConfig(), "/_auth", testTranslator(), testLogger())

	// Create request without password field
	reqBody := map[string]string{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/_auth/password/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleLogin(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleLogin() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandleLogin_InvalidJSON(t *testing.T) {
	cfg := config.PasswordAuthConfig{
		Enabled:  true,
		Password: "correct-password",
	}

	handler := NewHandler(cfg, createTestSessionStore(), testCookieConfig(), "/_auth", testTranslator(), testLogger())

	// Create request with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/_auth/password/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleLogin(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleLogin() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandleLogin_MethodNotAllowed(t *testing.T) {
	cfg := config.PasswordAuthConfig{
		Enabled:  true,
		Password: "correct-password",
	}

	handler := NewHandler(cfg, createTestSessionStore(), testCookieConfig(), "/_auth", testTranslator(), testLogger())

	// Create GET request (only POST is allowed)
	req := httptest.NewRequest(http.MethodGet, "/_auth/password/login", nil)

	w := httptest.NewRecorder()
	handler.HandleLogin(w, req)

	// Should return 405 Method Not Allowed
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleLogin() status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleLogin_WithRedirectParam(t *testing.T) {
	cfg := config.PasswordAuthConfig{
		Enabled:  true,
		Password: "correct-password",
	}

	handler := NewHandler(cfg, createTestSessionStore(), testCookieConfig(), "/_auth", testTranslator(), testLogger())

	// Create request with redirect parameter
	reqBody := map[string]string{
		"password": "correct-password",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/_auth/password/login?redirect=/dashboard", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleLogin(w, req)

	// Should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("HandleLogin() status = %v, want %v", w.Code, http.StatusOK)
	}

	// Should return JSON with custom redirect_url
	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if redirectURL := response["redirect_url"]; redirectURL != "/dashboard" {
		t.Errorf("redirect_url = %v, want %v", redirectURL, "/dashboard")
	}
}

func TestRenderPasswordForm(t *testing.T) {
	cfg := config.PasswordAuthConfig{
		Enabled:  true,
		Password: "test-password",
	}

	handler := NewHandler(cfg, createTestSessionStore(), testCookieConfig(), "/_auth", testTranslator(), testLogger())

	// Test rendering with English
	htmlEN := handler.RenderPasswordForm(i18n.English)
	if htmlEN == "" {
		t.Error("RenderPasswordForm() returned empty string for English")
	}

	// Should contain password input
	if !bytes.Contains([]byte(htmlEN), []byte(`type="password"`)) {
		t.Error("RenderPasswordForm() should contain password input")
	}

	// Should contain form
	if !bytes.Contains([]byte(htmlEN), []byte(`id="password-form"`)) {
		t.Error("RenderPasswordForm() should contain password-form")
	}

	// Should contain submit button
	if !bytes.Contains([]byte(htmlEN), []byte(`type="submit"`)) {
		t.Error("RenderPasswordForm() should contain submit button")
	}

	// Should contain password icon path
	if !bytes.Contains([]byte(htmlEN), []byte(`/_auth/assets/icons/password.svg`)) {
		t.Error("RenderPasswordForm() should contain password icon path")
	}

	// Test rendering with Japanese
	htmlJA := handler.RenderPasswordForm(i18n.Japanese)
	if htmlJA == "" {
		t.Error("RenderPasswordForm() returned empty string for Japanese")
	}
}

func TestRenderPasswordForm_CustomAuthPrefix(t *testing.T) {
	cfg := config.PasswordAuthConfig{
		Enabled:  true,
		Password: "test-password",
	}

	// Test with custom auth path prefix
	handler := NewHandler(cfg, createTestSessionStore(), testCookieConfig(), "/custom-auth", testTranslator(), testLogger())

	html := handler.RenderPasswordForm(i18n.English)

	// Should use custom prefix for icon path
	if !bytes.Contains([]byte(html), []byte(`/custom-auth/assets/icons/password.svg`)) {
		t.Error("RenderPasswordForm() should use custom auth prefix for icon path")
	}
}

func TestRenderPasswordForm_EmptyAuthPrefix(t *testing.T) {
	cfg := config.PasswordAuthConfig{
		Enabled:  true,
		Password: "test-password",
	}

	// Test with empty auth path prefix (should use default)
	handler := NewHandler(cfg, createTestSessionStore(), testCookieConfig(), "", testTranslator(), testLogger())

	html := handler.RenderPasswordForm(i18n.English)

	// Should use default /_auth prefix
	if !bytes.Contains([]byte(html), []byte(`/_auth/assets/icons/password.svg`)) {
		t.Error("RenderPasswordForm() should use default /_auth prefix when authPathPrefix is empty")
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp
	id2 := generateSessionID()

	// Should generate non-empty IDs
	if id1 == "" || id2 == "" {
		t.Error("generateSessionID() should not return empty string")
	}

	// Should generate unique IDs (due to timestamp difference)
	if id1 == id2 {
		t.Error("generateSessionID() should generate unique IDs")
	}

	// Should have pwd_ prefix
	if !bytes.HasPrefix([]byte(id1), []byte("pwd_")) {
		t.Error("generateSessionID() should have 'pwd_' prefix")
	}
}
