package factory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

func TestNewDefaultFactory(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	if factory == nil {
		t.Fatal("NewDefaultFactory returned nil")
	}

	if factory.host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", factory.host)
	}

	if factory.port != 4180 {
		t.Errorf("Expected port 4180, got %d", factory.port)
	}

	if factory.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestDefaultFactory_CreateTranslator(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	translator := factory.CreateTranslator()
	if translator == nil {
		t.Fatal("CreateTranslator returned nil")
	}
}

func TestDefaultFactory_CreateAuthzChecker(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	cfg := CreateTestConfig()

	checker := factory.CreateAuthzChecker(cfg.AccessControl)
	if checker == nil {
		t.Fatal("CreateAuthzChecker returned nil")
	}

	// Test allowed email
	if !checker.IsAllowed("test@example.com") {
		t.Error("Expected test@example.com to be allowed")
	}

	// Test disallowed
	if checker.IsAllowed("stranger@other.com") {
		t.Error("Expected stranger@other.com to be disallowed")
	}
}

func TestDefaultFactory_CreateForwarder(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	tests := []struct {
		name        string
		configFunc  func() *config.Config
		expectNil   bool
		description string
	}{
		{
			name: "no forwarding configured",
			configFunc: func() *config.Config {
				return CreateTestConfig()
			},
			expectNil:   true,
			description: "Should return nil when no forwarding is configured",
		},
		{
			name: "forwarding configured",
			configFunc: func() *config.Config {
				cfg := CreateTestConfig()
				cfg.Forwarding.Fields = []config.ForwardingField{
					{Path: "email", Header: "X-Email"},
				}
				return cfg
			},
			expectNil:   false,
			description: "Should return forwarder when forwarding is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.configFunc()
			forwarder := factory.CreateForwarder(cfg.Forwarding, cfg.OAuth2.Providers)

			if tt.expectNil {
				if forwarder != nil {
					t.Errorf("%s: expected nil, got %v", tt.description, forwarder)
				}
			} else {
				if forwarder == nil {
					t.Errorf("%s: expected non-nil forwarder", tt.description)
				}
			}
		})
	}
}

func TestDefaultFactory_CreateRulesEvaluator(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	tests := []struct {
		name        string
		configFunc  func() *config.Config
		expectError bool
		description string
	}{
		{
			name: "valid rules configured",
			configFunc: func() *config.Config {
				cfg := CreateTestConfig()
				// CreateTestConfig returns empty rules, which will use defaults
				return cfg
			},
			expectError: false,
			description: "Should create evaluator successfully with default rules",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.configFunc()
			evaluator, err := factory.CreateRulesEvaluator(&cfg.AccessControl.Rules)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error, got nil", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.description, err)
				}
				if evaluator == nil {
					t.Errorf("%s: expected non-nil evaluator", tt.description)
				}
			}
		})
	}
}

func TestDefaultFactory_CreateKVSStores(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	tests := []struct {
		name        string
		configFunc  func() *config.Config
		expectError bool
		description string
	}{
		{
			name: "default KVS configuration",
			configFunc: func() *config.Config {
				return CreateTestConfig()
			},
			expectError: false,
			description: "Should create all stores using default KVS with namespaces",
		},
		{
			name: "dedicated session KVS",
			configFunc: func() *config.Config {
				cfg := CreateTestConfig()
				cfg.KVS.Session = &kvs.Config{
					Type:      "memory",
					Namespace: "custom_session",
				}
				return cfg
			},
			expectError: false,
			description: "Should create dedicated session KVS",
		},
		{
			name: "dedicated token KVS",
			configFunc: func() *config.Config {
				cfg := CreateTestConfig()
				cfg.KVS.Token = &kvs.Config{
					Type:      "memory",
					Namespace: "custom_token",
				}
				return cfg
			},
			expectError: false,
			description: "Should create dedicated token KVS",
		},
		{
			name: "dedicated email quota KVS",
			configFunc: func() *config.Config {
				cfg := CreateTestConfig()
				cfg.KVS.EmailQuota = &kvs.Config{
					Type:      "memory",
					Namespace: "custom_email_quota",
				}
				return cfg
			},
			expectError: false,
			description: "Should create dedicated email quota KVS",
		},
		{
			name: "all dedicated KVS",
			configFunc: func() *config.Config {
				cfg := CreateTestConfig()
				cfg.KVS.Session = &kvs.Config{
					Type:      "memory",
					Namespace: "custom_session",
				}
				cfg.KVS.Token = &kvs.Config{
					Type:      "memory",
					Namespace: "custom_token",
				}
				cfg.KVS.EmailQuota = &kvs.Config{
					Type:      "memory",
					Namespace: "custom_email_quota",
				}
				return cfg
			},
			expectError: false,
			description: "Should create all stores with dedicated KVS configs",
		},
		{
			name: "invalid session KVS type",
			configFunc: func() *config.Config {
				cfg := CreateTestConfig()
				cfg.KVS.Session = &kvs.Config{
					Type:      "invalid-type",
					Namespace: "session",
				}
				return cfg
			},
			expectError: true,
			description: "Should fail when session KVS type is invalid",
		},
		{
			name: "invalid token KVS type",
			configFunc: func() *config.Config {
				cfg := CreateTestConfig()
				cfg.KVS.Token = &kvs.Config{
					Type:      "invalid-type",
					Namespace: "token",
				}
				return cfg
			},
			expectError: true,
			description: "Should fail when token KVS type is invalid",
		},
		{
			name: "invalid email quota KVS type",
			configFunc: func() *config.Config {
				cfg := CreateTestConfig()
				cfg.KVS.EmailQuota = &kvs.Config{
					Type:      "invalid-type",
					Namespace: "email_quota",
				}
				return cfg
			},
			expectError: true,
			description: "Should fail when email quota KVS type is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.configFunc()

			sessionKVS, tokenKVS, emailQuotaKVS, err := factory.CreateKVSStores(cfg)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error, got nil", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
				return
			}

			defer func() { _ = sessionKVS.Close() }()
			defer func() { _ = tokenKVS.Close() }()
			defer func() { _ = emailQuotaKVS.Close() }()

			if sessionKVS == nil {
				t.Error("Expected sessionKVS to be non-nil")
			}
			if tokenKVS == nil {
				t.Error("Expected tokenKVS to be non-nil")
			}
			if emailQuotaKVS == nil {
				t.Error("Expected emailQuotaKVS to be non-nil")
			}

			// Test that stores are functional
			ctx := context.Background()
			err = sessionKVS.Set(ctx, "test", []byte("value"), 3600*time.Second)
			if err != nil {
				t.Errorf("sessionKVS Set failed: %v", err)
			}

			err = tokenKVS.Set(ctx, "test", []byte("value"), 3600*time.Second)
			if err != nil {
				t.Errorf("tokenKVS Set failed: %v", err)
			}

			err = emailQuotaKVS.Set(ctx, "test", []byte("value"), 3600*time.Second)
			if err != nil {
				t.Errorf("emailQuotaKVS Set failed: %v", err)
			}
		})
	}
}

func TestDefaultFactory_CreateSessionStore(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	cfg := CreateTestConfig()
	sessionKVS, _, _, err := factory.CreateKVSStores(cfg)
	if err != nil {
		t.Fatalf("CreateKVSStores failed: %v", err)
	}
	defer func() { _ = sessionKVS.Close() }()

	sessionStore := factory.CreateSessionStore(sessionKVS)
	if sessionStore == nil {
		t.Fatal("CreateSessionStore returned nil")
	}
}

func TestDefaultFactory_CreateOAuth2Manager(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	cfg := CreateTestConfigWithOAuth2()

	manager := factory.CreateOAuth2Manager(cfg.OAuth2, cfg.Server, "localhost", 4180)
	if manager == nil {
		t.Fatal("CreateOAuth2Manager returned nil")
	}
}

func TestDefaultFactory_CreateMiddleware(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	cfg := CreateTestConfigWithOAuth2()

	// Create required dependencies
	sessionKVS, tokenKVS, rateLimitKVS, err := factory.CreateKVSStores(cfg)
	if err != nil {
		t.Fatalf("CreateKVSStores failed: %v", err)
	}
	defer func() {
		_ = sessionKVS.Close()
		_ = tokenKVS.Close()
		_ = rateLimitKVS.Close()
	}()

	sessionStore := factory.CreateSessionStore(sessionKVS)

	// Create test upstream
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	// Create a simple mock proxy handler for testing
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create middleware with KVS stores
	middleware, err := factory.CreateMiddleware(cfg, sessionStore, tokenKVS, rateLimitKVS, proxyHandler, logger)
	if err != nil {
		t.Fatalf("CreateMiddleware failed: %v", err)
	}
	if middleware == nil {
		t.Fatal("CreateMiddleware returned nil")
	}
}

func TestDefaultFactory_CreateEmailHandler(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	tests := []struct {
		name        string
		host        string
		port        int
		baseURL     string
		expectError bool
		description string
	}{
		{
			name:        "localhost development",
			host:        "localhost",
			port:        4180,
			baseURL:     "",
			expectError: false,
			description: "Should use http:// for localhost",
		},
		{
			name:        "127.0.0.1 development",
			host:        "127.0.0.1",
			port:        8080,
			baseURL:     "",
			expectError: false,
			description: "Should use http:// for 127.0.0.1",
		},
		{
			name:        "0.0.0.0 bind address",
			host:        "0.0.0.0",
			port:        4180,
			baseURL:     "",
			expectError: false,
			description: "Should use http://localhost for 0.0.0.0",
		},
		{
			name:        "production domain",
			host:        "example.com",
			port:        443,
			baseURL:     "",
			expectError: false,
			description: "Should use https:// for production domains",
		},
		{
			name:        "with explicit base URL",
			host:        "localhost",
			port:        4180,
			baseURL:     "https://my-custom-domain.com",
			expectError: false,
			description: "Should use explicit base URL when provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := CreateTestConfigWithEmail()
			cfg.Server.BaseURL = tt.baseURL

			// Create required dependencies
			sessionKVS, tokenKVS, rateLimitKVS, err := factory.CreateKVSStores(cfg)
			if err != nil {
				t.Fatalf("CreateKVSStores failed: %v", err)
			}
			defer func() {
				_ = sessionKVS.Close()
				_ = tokenKVS.Close()
				_ = rateLimitKVS.Close()
			}()

			translator := factory.CreateTranslator()
			authzChecker := factory.CreateAuthzChecker(cfg.AccessControl)

			handler, err := factory.CreateEmailHandler(
				cfg.EmailAuth,
				cfg.Service,
				cfg.Server,
				cfg.Session,
				tt.host,
				tt.port,
				authzChecker,
				translator,
				tokenKVS,
				rateLimitKVS,
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error, got nil", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.description, err)
				}
				if handler == nil {
					t.Errorf("%s: expected non-nil handler", tt.description)
				}
			}
		})
	}
}

func TestDefaultFactory_CreatePasswordHandler(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	cfg := CreateTestConfig()
	cfg.PasswordAuth = config.PasswordAuthConfig{
		Enabled:  true,
		Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // "password"
	}

	// Create required dependencies
	sessionKVS, _, _, err := factory.CreateKVSStores(cfg)
	if err != nil {
		t.Fatalf("CreateKVSStores failed: %v", err)
	}
	defer func() { _ = sessionKVS.Close() }()

	translator := factory.CreateTranslator()

	handler := factory.CreatePasswordHandler(
		cfg.PasswordAuth,
		cfg.Session.Cookie,
		cfg.Server.AuthPathPrefix,
		sessionKVS,
		translator,
	)

	if handler == nil {
		t.Fatal("CreatePasswordHandler returned nil")
	}
}
