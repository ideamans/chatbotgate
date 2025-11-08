package factory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
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

	checker := factory.CreateAuthzChecker(cfg.Authorization)
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
			evaluator, err := factory.CreateRulesEvaluator(&cfg.Rules)

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

	cfg := CreateTestConfig()

	sessionKVS, tokenKVS, rateLimitKVS, err := factory.CreateKVSStores(cfg)
	if err != nil {
		t.Fatalf("CreateKVSStores failed: %v", err)
	}
	defer func() { _ = sessionKVS.Close() }()
	defer func() { _ = tokenKVS.Close() }()
	defer func() { _ = rateLimitKVS.Close() }()

	if sessionKVS == nil {
		t.Error("Expected sessionKVS to be non-nil")
	}
	if tokenKVS == nil {
		t.Error("Expected tokenKVS to be non-nil")
	}
	if rateLimitKVS == nil {
		t.Error("Expected rateLimitKVS to be non-nil")
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

	err = rateLimitKVS.Set(ctx, "test", []byte("value"), 3600*time.Second)
	if err != nil {
		t.Errorf("rateLimitKVS Set failed: %v", err)
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
