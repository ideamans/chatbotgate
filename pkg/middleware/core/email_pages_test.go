package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// TestHandleEmailSent tests the email sent confirmation page
func TestHandleEmailSent(t *testing.T) {
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
		checkText      string
	}{
		{
			name:           "Email sent page in English",
			acceptLanguage: "en-US",
			wantStatus:     http.StatusOK,
			checkText:      "Check Your Email",
		},
		{
			name:           "Email sent page in Japanese",
			acceptLanguage: "ja",
			wantStatus:     http.StatusOK,
			checkText:      "メールを確認してください",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/_auth/email/sent", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			w := httptest.NewRecorder()

			middleware.handleEmailSent(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.checkText) {
				t.Errorf("Expected body to contain %q", tt.checkText)
			}

			// Check that it's HTML
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Errorf("Content-Type = %q, want text/html", contentType)
			}

			// Check that OTP form is present
			if !strings.Contains(body, "verify-otp") {
				t.Error("Expected OTP verification form in body")
			}
		})
	}
}

// TestHandleForbidden tests the forbidden (403) error page
func TestHandleForbidden(t *testing.T) {
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
		checkText      string
	}{
		{
			name:           "Forbidden page in English",
			acceptLanguage: "en-US",
			wantStatus:     http.StatusForbidden,
			checkText:      "Access Denied",
		},
		{
			name:           "Forbidden page in Japanese",
			acceptLanguage: "ja",
			wantStatus:     http.StatusForbidden,
			checkText:      "アクセス拒否",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/forbidden", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			w := httptest.NewRecorder()

			middleware.handleForbidden(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.checkText) {
				t.Errorf("Expected body to contain %q", tt.checkText)
			}

			// Check that it's HTML
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Errorf("Content-Type = %q, want text/html", contentType)
			}
		})
	}
}

// TestHandleEmailFetchError tests the email required error page
func TestHandleEmailFetchError(t *testing.T) {
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
		checkText      string
	}{
		{
			name:           "Email fetch error in English",
			acceptLanguage: "en-US",
			wantStatus:     http.StatusBadRequest,
			checkText:      "Email Address Required",
		},
		{
			name:           "Email fetch error in Japanese",
			acceptLanguage: "ja",
			wantStatus:     http.StatusBadRequest,
			checkText:      "メールアドレスが必要です",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/error", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			w := httptest.NewRecorder()

			middleware.handleEmailFetchError(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.checkText) {
				t.Errorf("Expected body to contain %q", tt.checkText)
			}

			// Check that it's HTML
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Errorf("Content-Type = %q, want text/html", contentType)
			}
		})
	}
}
