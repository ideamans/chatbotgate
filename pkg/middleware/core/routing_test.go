package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/middleware/rules"
	"github.com/ideamans/chatbotgate/pkg/middleware/session"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// TestServeHTTP_Routing tests the main routing logic
func TestServeHTTP_Routing(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:   "_test",
				Secure: false,
			},
		},
	}

	sessionStore, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{})
	defer func() { _ = sessionStore.Close() }()
	oauthManager := oauth2.NewManager()
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		oauthManager,
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

	// Set middleware as ready for health check
	middleware.SetReady()

	tests := []struct {
		name           string
		path           string
		wantStatus     int
		checkRedirect  bool
		redirectPrefix string
	}{
		{
			name:          "Login page",
			path:          "/_auth/login",
			wantStatus:    http.StatusOK,
			checkRedirect: false,
		},
		{
			name:          "Logout page",
			path:          "/_auth/logout",
			wantStatus:    http.StatusOK,
			checkRedirect: false,
		},
		{
			name:          "404 page",
			path:          "/_auth/404",
			wantStatus:    http.StatusNotFound,
			checkRedirect: false,
		},
		{
			name:          "500 page",
			path:          "/_auth/500",
			wantStatus:    http.StatusInternalServerError,
			checkRedirect: false,
		},
		{
			name:          "Health endpoint",
			path:          "/_auth/health",
			wantStatus:    http.StatusOK,
			checkRedirect: false,
		},
		{
			name:           "Protected path without session",
			path:           "/protected",
			wantStatus:     http.StatusFound,
			checkRedirect:  true,
			redirectPrefix: "/_auth/login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.checkRedirect {
				location := w.Header().Get("Location")
				if location == "" {
					t.Error("Expected redirect, but Location header is empty")
				} else if tt.redirectPrefix != "" {
					// Check if location starts with the redirect prefix
					if len(location) < len(tt.redirectPrefix) || location[:len(tt.redirectPrefix)] != tt.redirectPrefix {
						t.Errorf("Location = %q, want prefix %q", location, tt.redirectPrefix)
					}
				}
			}
		})
	}
}

// TestServeHTTP_WithRules tests routing with access control rules
func TestServeHTTP_WithRules(t *testing.T) {
	rulesConfig := rules.Config{
		{
			Prefix: "/public/",
			Action: rules.ActionAllow,
		},
		{
			Prefix: "/forbidden/",
			Action: rules.ActionDeny,
		},
		{
			Prefix: "/protected/",
			Action: rules.ActionAuth,
		},
	}

	rulesEvaluator, err := rules.NewEvaluator(&rulesConfig)
	if err != nil {
		t.Fatalf("Failed to create rules evaluator: %v", err)
	}

	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:   "_test",
				Secure: false,
			},
		},
	}

	sessionStore, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{})
	defer func() { _ = sessionStore.Close() }()
	oauthManager := oauth2.NewManager()
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		oauthManager,
		nil, // email handler
		nil, // agreement handler
		nil, // authz checker
		nil, // forwarder
		rulesEvaluator,
		translator,
		logger,
	)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	tests := []struct {
		name       string
		path       string
		wantStatus int
		checkBody  string
	}{
		{
			name:       "Allow public path without authentication",
			path:       "/public/page",
			wantStatus: http.StatusOK,
			checkBody:  "Allowed",
		},
		{
			name:       "Deny forbidden path",
			path:       "/forbidden/page",
			wantStatus: http.StatusForbidden,
			checkBody:  "Access Denied\n",
		},
		{
			name:       "Redirect protected path to login",
			path:       "/protected/page",
			wantStatus: http.StatusFound,
			checkBody:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.checkBody != "" {
				body := w.Body.String()
				if body != tt.checkBody {
					t.Errorf("Body = %q, want %q", body, tt.checkBody)
				}
			}
		})
	}
}

// TestRequireAuth tests the authentication requirement logic
func TestRequireAuth(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:   "_test",
				Secure: false,
			},
		},
	}

	sessionStore, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{})
	defer func() { _ = sessionStore.Close() }()
	oauthManager := oauth2.NewManager()
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		oauthManager,
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

	// Set middleware as ready for health check
	middleware.SetReady()

	tests := []struct {
		name          string
		setupSession  func() string // Returns session ID
		addCookie     bool
		wantStatus    int
		checkRedirect bool
		checkBody     string
	}{
		{
			name: "No session cookie - redirect to login",
			setupSession: func() string {
				return ""
			},
			addCookie:     false,
			wantStatus:    http.StatusFound,
			checkRedirect: true,
		},
		{
			name: "Invalid session ID - redirect to login",
			setupSession: func() string {
				return "invalid-session-id"
			},
			addCookie:     true,
			wantStatus:    http.StatusFound,
			checkRedirect: true,
		},
		{
			name: "Expired session - redirect to login",
			setupSession: func() string {
				sessionID := "expired-session"
				sess := &session.Session{
					ID:            sessionID,
					Email:         "user@example.com",
					Provider:      "google",
					CreatedAt:     time.Now().Add(-48 * time.Hour),
					ExpiresAt:     time.Now().Add(-24 * time.Hour), // Expired
					Authenticated: true,
				}
				_ = session.Set(sessionStore, sessionID, sess)
				return sessionID
			},
			addCookie:     true,
			wantStatus:    http.StatusFound,
			checkRedirect: true,
		},
		{
			name: "Valid session - allow access",
			setupSession: func() string {
				sessionID := "valid-session"
				sess := &session.Session{
					ID:            sessionID,
					Email:         "user@example.com",
					Provider:      "google",
					CreatedAt:     time.Now(),
					ExpiresAt:     time.Now().Add(24 * time.Hour),
					Authenticated: true,
				}
				_ = session.Set(sessionStore, sessionID, sess)
				return sessionID
			},
			addCookie:     true,
			wantStatus:    http.StatusOK,
			checkRedirect: false,
			checkBody:     "Authenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID := tt.setupSession()

			req := httptest.NewRequest("GET", "/protected", nil)
			if tt.addCookie && sessionID != "" {
				req.AddCookie(&http.Cookie{
					Name:  "_test",
					Value: sessionID,
				})
			}
			w := httptest.NewRecorder()

			middleware.requireAuth(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.checkRedirect {
				location := w.Header().Get("Location")
				if location == "" {
					t.Error("Expected redirect, but Location header is empty")
				}
			}

			if tt.checkBody != "" {
				body := w.Body.String()
				if body != tt.checkBody {
					t.Errorf("Body = %q, want %q", body, tt.checkBody)
				}
			}
		})
	}
}

// TestRequireAuth_WithNextHandler tests authentication with a next handler
func TestRequireAuth_WithNextHandler(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:   "_test",
				Secure: false,
			},
		},
	}

	sessionStore, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{})
	defer func() { _ = sessionStore.Close() }()
	oauthManager := oauth2.NewManager()
	translator := i18n.NewTranslator()
	logger := logging.NewTestLogger()

	middleware, err := New(
		cfg,
		sessionStore,
		oauthManager,
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

	// Set middleware as ready for health check
	middleware.SetReady()

	// Set up next handler
	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Next handler called"))
	})
	wrappedHandler := middleware.Wrap(nextHandler)

	// Create valid session
	sessionID := "valid-session"
	sess := &session.Session{
		ID:            sessionID,
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		Authenticated: true,
	}
	_ = session.Set(sessionStore, sessionID, sess)

	// Test with valid session
	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "_test",
		Value: sessionID,
	})
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if !nextHandlerCalled {
		t.Error("Expected next handler to be called, but it wasn't")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body != "Next handler called" {
		t.Errorf("Body = %q, want %q", body, "Next handler called")
	}
}
