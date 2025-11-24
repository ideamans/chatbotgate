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

// TestHandleMainCSS tests the main CSS handler
func TestHandleMainCSS(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/_auth/assets/main.css", nil)
	w := httptest.NewRecorder()

	middleware.handleMainCSS(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/css; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/css; charset=utf-8")
	}

	// Check that CSS content is not empty
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty CSS content")
	}

	// Basic sanity check that it looks like CSS
	if !strings.Contains(body, "{") || !strings.Contains(body, "}") {
		t.Error("CSS content doesn't look like valid CSS")
	}
}

// TestHandleIcon tests the icon handler
func TestHandleIcon(t *testing.T) {
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
		name            string
		path            string
		wantStatus      int
		wantContentType string
	}{
		{
			name:            "SVG icon",
			path:            "/_auth/assets/icons/google.svg",
			wantStatus:      http.StatusOK,
			wantContentType: "image/svg+xml",
		},
		{
			name:            "ChatbotGate icon",
			path:            "/_auth/assets/icons/chatbotgate.svg",
			wantStatus:      http.StatusOK,
			wantContentType: "image/svg+xml",
		},
		{
			name:       "Non-existent icon",
			path:       "/_auth/assets/icons/nonexistent.svg",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "Path traversal attempt",
			path:       "/_auth/assets/icons/../../etc/passwd",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			middleware.handleIcon(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantContentType != "" {
				contentType := w.Header().Get("Content-Type")
				if contentType != tt.wantContentType {
					t.Errorf("Content-Type = %q, want %q", contentType, tt.wantContentType)
				}
			}

			// Check cache headers for successful responses
			if tt.wantStatus == http.StatusOK {
				cacheControl := w.Header().Get("Cache-Control")
				if !strings.Contains(cacheControl, "public") {
					t.Errorf("Cache-Control should contain 'public', got %q", cacheControl)
				}
			}
		})
	}
}
