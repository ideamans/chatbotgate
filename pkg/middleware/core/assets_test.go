package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
	"github.com/ideamans/chatbotgate/pkg/middleware/authz"
	"github.com/ideamans/chatbotgate/pkg/middleware/config"
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
			Cookie: config.CookieConfig{
				Name: "_test",
			},
		},
		Assets: config.AssetsConfig{
			Optimization: config.OptimizationConfig{
				Dify: false, // Disabled
			},
		},
	}

	sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.AccessControl)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		oauthManager,
		nil, // email handler
		nil, // agreement handler
		authzChecker,
		nil, // forwarder
		nil, // rules evaluator not needed for this test
		translator,
		logger,
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

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
			Cookie: config.CookieConfig{
				Name: "_test",
			},
		},
		Assets: config.AssetsConfig{
			Optimization: config.OptimizationConfig{
				Dify: true, // Enabled
			},
		},
	}

	sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.AccessControl)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		oauthManager,
		nil, // email handler
		nil, // agreement handler
		authzChecker,
		nil, // forwarder
		nil, // rules evaluator not needed for this test
		translator,
		logger,
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

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
			Cookie: config.CookieConfig{
				Name: "_test",
			},
		},
		Assets: config.AssetsConfig{
			Optimization: config.OptimizationConfig{
				Dify: true,
			},
		},
	}

	sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.AccessControl)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		oauthManager,
		nil, // email handler
		nil, // agreement handler
		authzChecker,
		nil, // forwarder
		nil, // rules evaluator not needed for this test
		translator,
		logger,
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

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
			Cookie: config.CookieConfig{
				Name: "_test",
			},
		},
	}

	sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.AccessControl)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		oauthManager,
		nil, // email handler
		nil, // agreement handler
		authzChecker,
		nil, // forwarder
		nil, // rules evaluator not needed for this test
		translator,
		logger,
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

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
			Cookie: config.CookieConfig{
				Name: "_test",
			},
		},
	}

	sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.AccessControl)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		oauthManager,
		nil, // email handler
		nil, // agreement handler
		authzChecker,
		nil, // forwarder
		nil, // rules evaluator not needed for this test
		translator,
		logger,
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

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
			Cookie: config.CookieConfig{
				Name: "_test",
			},
		},
	}

	sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
	oauthManager := oauth2.NewManager()
	authzChecker := authz.NewEmailChecker(cfg.AccessControl)
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		oauthManager,
		nil, // email handler
		nil, // agreement handler
		authzChecker,
		nil, // forwarder
		nil, // rules evaluator not needed for this test
		translator,
		logger,
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

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

// TestBuildAuthHeader tests the buildAuthHeader method with different configurations
func TestBuildAuthHeader(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		iconURL     string
		logoURL     string
		logoWidth   string
		wantLogoImg bool
		wantIconImg bool
		wantTitle   bool
		wantWidth   string
	}{
		{
			name:        "title only (no icon, no logo)",
			serviceName: "Test Service",
			iconURL:     "",
			logoURL:     "",
			logoWidth:   "",
			wantLogoImg: false,
			wantIconImg: false,
			wantTitle:   true,
			wantWidth:   "",
		},
		{
			name:        "icon and title (no logo)",
			serviceName: "Test Service",
			iconURL:     "https://example.com/icon.svg",
			logoURL:     "",
			logoWidth:   "",
			wantLogoImg: false,
			wantIconImg: true,
			wantTitle:   true,
			wantWidth:   "",
		},
		{
			name:        "logo and title (no icon)",
			serviceName: "Test Service",
			iconURL:     "",
			logoURL:     "https://example.com/logo.svg",
			logoWidth:   "",
			wantLogoImg: true,
			wantIconImg: false,
			wantTitle:   true,
			wantWidth:   "200px", // default
		},
		{
			name:        "logo with custom width (icon ignored)",
			serviceName: "Test Service",
			iconURL:     "https://example.com/icon.svg", // This should be ignored when logo is present
			logoURL:     "https://example.com/logo.svg",
			logoWidth:   "150px",
			wantLogoImg: true,
			wantIconImg: false, // Icon should not be rendered when logo is present
			wantTitle:   true,
			wantWidth:   "150px",
		},
		{
			name:        "logo with default width (icon ignored)",
			serviceName: "Test Service",
			iconURL:     "https://example.com/icon.svg", // This should be ignored when logo is present
			logoURL:     "https://example.com/logo.svg",
			logoWidth:   "",
			wantLogoImg: true,
			wantIconImg: false, // Icon should not be rendered when logo is present
			wantTitle:   true,
			wantWidth:   "200px", // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Service: config.ServiceConfig{
					Name:      tt.serviceName,
					IconURL:   tt.iconURL,
					LogoURL:   tt.logoURL,
					LogoWidth: tt.logoWidth,
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

			sessionStore := func() kvs.Store { store, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{}); return store }()
			oauthManager := oauth2.NewManager()
			authzChecker := authz.NewEmailChecker(cfg.AccessControl)
			translator := i18n.NewTranslator()
			logger := logging.NewTestLogger()

			middleware, err := New(
				cfg,
				sessionStore,
				oauthManager,
				nil, // email handler
				nil, // agreement handler
				authzChecker,
				nil, // forwarder
				nil, // rules evaluator
				translator,
				logger,
			)
			if err != nil {
				t.Fatalf("Failed to create middleware: %v", err)
			}

			result := middleware.buildAuthHeader("/_auth")

			// Check logo image
			if tt.wantLogoImg {
				if !strings.Contains(result, `<img src="`+tt.logoURL+`"`) {
					t.Errorf("Expected logo image with src '%s', got: %s", tt.logoURL, result)
				}
				if !strings.Contains(result, `class="auth-logo"`) {
					t.Errorf("Expected auth-logo class, got: %s", result)
				}
				if !strings.Contains(result, `--auth-logo-width: `+tt.wantWidth) {
					t.Errorf("Expected logo width '%s', got: %s", tt.wantWidth, result)
				}
			} else {
				if strings.Contains(result, `class="auth-logo"`) {
					t.Errorf("Unexpected logo image in result: %s", result)
				}
			}

			// Check icon image
			if tt.wantIconImg {
				if !strings.Contains(result, `<img src="`+tt.iconURL+`"`) {
					t.Errorf("Expected icon image with src '%s', got: %s", tt.iconURL, result)
				}
				if !strings.Contains(result, `class="auth-icon"`) {
					t.Errorf("Expected auth-icon class, got: %s", result)
				}
				if !strings.Contains(result, `class="auth-header"`) {
					t.Errorf("Expected auth-header wrapper div, got: %s", result)
				}
			} else {
				if strings.Contains(result, `class="auth-icon"`) {
					t.Errorf("Unexpected icon image in result: %s", result)
				}
			}

			// Check title
			if tt.wantTitle {
				if !strings.Contains(result, `<h1 class="auth-title">`+tt.serviceName+`</h1>`) {
					t.Errorf("Expected title '%s', got: %s", tt.serviceName, result)
				}
			}
		})
	}
}
