package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
	"github.com/ideamans/chatbotgate/pkg/middleware/authz"
	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/middleware/passthrough"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// TestBuildStyleLinks_DifyDisabled tests buildStyleLinks when dify optimization is disabled
func TestBuildStyleLinks_DifyDisabled(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			CookieName: "_test",
		},
		Assets: config.AssetsConfig{
			Optimization: config.OptimizationConfig{
				Dify: false, // Disabled
			},
		},
	}

	sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.Authorization)
	passthroughMatcher := passthrough.NewMatcher(&cfg.Passthrough)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware := New(
		cfg,
		sessionStore,
		oauthManager,
		nil,
		authzChecker,
		nil,
		passthroughMatcher,
		translator,
		logger,
	)

	result := middleware.buildStyleLinks()

	// Should only contain main.css
	if !strings.Contains(result, "/_auth/assets/main.css") {
		t.Error("buildStyleLinks() should contain main.css link")
	}

	// Should NOT contain dify.css
	if strings.Contains(result, "dify.css") {
		t.Error("buildStyleLinks() should not contain dify.css link when optimization is disabled")
	}
}

// TestBuildStyleLinks_DifyEnabled tests buildStyleLinks when dify optimization is enabled
func TestBuildStyleLinks_DifyEnabled(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			CookieName: "_test",
		},
		Assets: config.AssetsConfig{
			Optimization: config.OptimizationConfig{
				Dify: true, // Enabled
			},
		},
	}

	sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.Authorization)
	passthroughMatcher := passthrough.NewMatcher(&cfg.Passthrough)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware := New(
		cfg,
		sessionStore,
		oauthManager,
		nil,
		authzChecker,
		nil,
		passthroughMatcher,
		translator,
		logger,
	)

	result := middleware.buildStyleLinks()

	// Should contain both main.css and dify.css
	if !strings.Contains(result, "/_auth/assets/main.css") {
		t.Error("buildStyleLinks() should contain main.css link")
	}

	if !strings.Contains(result, "/_auth/assets/dify.css") {
		t.Error("buildStyleLinks() should contain dify.css link when optimization is enabled")
	}
}

// TestBuildStyleLinks_CustomPrefix tests buildStyleLinks with custom auth path prefix
func TestBuildStyleLinks_CustomPrefix(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_custom_auth", // Custom prefix
		},
		Session: config.SessionConfig{
			CookieName: "_test",
		},
		Assets: config.AssetsConfig{
			Optimization: config.OptimizationConfig{
				Dify: true,
			},
		},
	}

	sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.Authorization)
	passthroughMatcher := passthrough.NewMatcher(&cfg.Passthrough)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware := New(
		cfg,
		sessionStore,
		oauthManager,
		nil,
		authzChecker,
		nil,
		passthroughMatcher,
		translator,
		logger,
	)

	result := middleware.buildStyleLinks()

	// Should use custom prefix
	if !strings.Contains(result, "/_custom_auth/assets/main.css") {
		t.Error("buildStyleLinks() should use custom auth path prefix for main.css")
	}

	if !strings.Contains(result, "/_custom_auth/assets/dify.css") {
		t.Error("buildStyleLinks() should use custom auth path prefix for dify.css")
	}
}

// TestHandleDifyCSS tests the dify.css handler
func TestHandleDifyCSS(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			CookieName: "_test",
		},
	}

	sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.Authorization)
	passthroughMatcher := passthrough.NewMatcher(&cfg.Passthrough)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware := New(
		cfg,
		sessionStore,
		oauthManager,
		nil,
		authzChecker,
		nil,
		passthroughMatcher,
		translator,
		logger,
	)

	req := httptest.NewRequest("GET", "/_auth/assets/dify.css", nil)
	w := httptest.NewRecorder()

	middleware.handleDifyCSS(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/css; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/css; charset=utf-8', got '%s'", contentType)
	}

	// Check Cache-Control header
	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "public, max-age=31536000" {
		t.Errorf("Expected Cache-Control 'public, max-age=31536000', got '%s'", cacheControl)
	}

	// Check that body is not empty
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty response body")
	}
}

// TestMiddleware_DifyCSSRoute tests that dify.css route is properly registered
func TestMiddleware_DifyCSSRoute(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			CookieName: "_test",
		},
	}

	sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.Authorization)
	passthroughMatcher := passthrough.NewMatcher(&cfg.Passthrough)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware := New(
		cfg,
		sessionStore,
		oauthManager,
		nil,
		authzChecker,
		nil,
		passthroughMatcher,
		translator,
		logger,
	)

	req := httptest.NewRequest("GET", "/_auth/assets/dify.css", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Should successfully serve dify.css
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for dify.css route, got %d", w.Code)
	}

	// Should have CSS content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/css; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/css; charset=utf-8', got '%s'", contentType)
	}
}

// TestMiddleware_DifyCSSRoute_CustomPrefix tests dify.css route with custom prefix
func TestMiddleware_DifyCSSRoute_CustomPrefix(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_custom", // Custom prefix
		},
		Session: config.SessionConfig{
			CookieName: "_test",
		},
	}

	sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.Authorization)
	passthroughMatcher := passthrough.NewMatcher(&cfg.Passthrough)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware := New(
		cfg,
		sessionStore,
		oauthManager,
		nil,
		authzChecker,
		nil,
		passthroughMatcher,
		translator,
		logger,
	)

	req := httptest.NewRequest("GET", "/_custom/assets/dify.css", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Should successfully serve dify.css with custom prefix
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for custom prefix dify.css route, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/css; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/css; charset=utf-8', got '%s'", contentType)
	}
}
