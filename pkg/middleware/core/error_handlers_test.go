package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// TestHandleLogout tests the logout handler
func TestHandleLogout(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:     "_test",
				Secure:   false,
				HTTPOnly: true,
			},
		},
	}

	sessionStore, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{})
	defer func() { _ = sessionStore.Close() }()
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		nil, // oauth manager
		nil, // email handler
		nil, // agreement handler
		nil, // authz checker
		nil, // forwarder
		nil, // rules evaluator
		translator,
		logger,
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	tests := []struct {
		name            string
		method          string
		acceptLanguage  string
		wantStatus      int
		checkCookie     bool
		checkBodyString string
	}{
		{
			name:            "GET logout in English",
			method:          "GET",
			acceptLanguage:  "en-US",
			wantStatus:      http.StatusOK,
			checkCookie:     true,
			checkBodyString: "Logged Out",
		},
		{
			name:            "POST logout in Japanese",
			method:          "POST",
			acceptLanguage:  "ja",
			wantStatus:      http.StatusOK,
			checkCookie:     true,
			checkBodyString: "ログアウト",
		},
		{
			name:            "Logout without language header",
			method:          "GET",
			acceptLanguage:  "",
			wantStatus:      http.StatusOK,
			checkCookie:     true,
			checkBodyString: "", // Don't check specific text
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/_auth/logout", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			w := httptest.NewRecorder()

			middleware.handleLogout(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.checkCookie {
				// Check that cookie is cleared (MaxAge should be negative)
				cookies := w.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == "_test" {
						found = true
						if cookie.MaxAge >= 0 {
							t.Errorf("Cookie MaxAge = %d, want negative value to clear cookie", cookie.MaxAge)
						}
						if cookie.Value != "" {
							t.Errorf("Cookie Value = %q, want empty to clear cookie", cookie.Value)
						}
					}
				}
				if !found {
					t.Error("Expected Set-Cookie header to clear session cookie")
				}
			}

			if tt.checkBodyString != "" {
				body := w.Body.String()
				if !strings.Contains(body, tt.checkBodyString) {
					t.Errorf("Expected body to contain %q", tt.checkBodyString)
				}
			}

			// Check that it's HTML
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Errorf("Content-Type = %q, want text/html", contentType)
			}
		})
	}
}

// TestHandle404 tests the 404 error handler
func TestHandle404(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name: "_test",
			},
		},
	}

	sessionStore, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{})
	defer func() { _ = sessionStore.Close() }()
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		nil, // oauth manager
		nil, // email handler
		nil, // agreement handler
		nil, // authz checker
		nil, // forwarder
		nil, // rules evaluator
		translator,
		logger,
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	tests := []struct {
		name           string
		acceptLanguage string
		wantStatus     int
		checkMessage   string
	}{
		{
			name:           "404 in English",
			acceptLanguage: "en-US",
			wantStatus:     http.StatusNotFound,
			checkMessage:   "Not Found",
		},
		{
			name:           "404 in Japanese",
			acceptLanguage: "ja",
			wantStatus:     http.StatusNotFound,
			checkMessage:   "見つかりません",
		},
		{
			name:           "404 without Accept-Language",
			acceptLanguage: "",
			wantStatus:     http.StatusNotFound,
			checkMessage:   "Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/nonexistent", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			w := httptest.NewRecorder()

			middleware.handle404(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.checkMessage) {
				t.Errorf("Expected body to contain %q, but got: %s", tt.checkMessage, body)
			}

			// Check that it's HTML
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Errorf("Content-Type = %q, want text/html", contentType)
			}
		})
	}
}

// TestHandle500 tests the 500 error handler
func TestHandle500(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name: "_test",
			},
		},
	}

	sessionStore, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{})
	defer func() { _ = sessionStore.Close() }()
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		nil, // oauth manager
		nil, // email handler
		nil, // agreement handler
		nil, // authz checker
		nil, // forwarder
		nil, // rules evaluator
		translator,
		logger,
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	tests := []struct {
		name           string
		acceptLanguage string
		err            error
		wantStatus     int
		checkMessage   string
	}{
		{
			name:           "500 in English with error message",
			acceptLanguage: "en-US",
			err:            errors.New("Database connection failed"),
			wantStatus:     http.StatusInternalServerError,
			checkMessage:   "Internal Server Error",
		},
		{
			name:           "500 in Japanese with error message",
			acceptLanguage: "ja",
			err:            errors.New("データベース接続エラー"),
			wantStatus:     http.StatusInternalServerError,
			checkMessage:   "予期しないエラーが発生しました",
		},
		{
			name:           "500 without error message",
			acceptLanguage: "en-US",
			err:            nil,
			wantStatus:     http.StatusInternalServerError,
			checkMessage:   "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/error", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			w := httptest.NewRecorder()

			middleware.handle500(w, req, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.checkMessage) {
				t.Errorf("Expected body to contain %q, but got: %s", tt.checkMessage, body)
			}

			// Error message should be in the body if provided
			if tt.err != nil && !strings.Contains(body, tt.err.Error()) {
				t.Errorf("Expected body to contain error message %q, but got: %s", tt.err.Error(), body)
			}

			// Check that it's HTML
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Errorf("Content-Type = %q, want text/html", contentType)
			}
		})
	}
}
