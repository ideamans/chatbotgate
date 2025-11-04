package factory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/logging"
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
				cfg.Forwarding.Fields = []string{"email"}
				cfg.Forwarding.Header.Enabled = true
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

func TestDefaultFactory_CreatePassthroughMatcher(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	tests := []struct {
		name        string
		configFunc  func() *config.Config
		expectNil   bool
		description string
	}{
		{
			name: "passthrough configured",
			configFunc: func() *config.Config {
				cfg := CreateTestConfig()
				cfg.Passthrough.Prefix = []string{"/health", "/metrics"}
				return cfg
			},
			expectNil:   false,
			description: "Should return matcher when passthrough paths are configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.configFunc()
			matcher := factory.CreatePassthroughMatcher(cfg.Passthrough)

			if tt.expectNil {
				if matcher != nil {
					t.Errorf("%s: expected nil, got %v", tt.description, matcher)
				}
			} else {
				if matcher == nil {
					t.Errorf("%s: expected non-nil matcher", tt.description)
				}
			}
		})
	}
}

func TestDefaultFactory_CreateTokenKVS(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	kvsStore := factory.CreateTokenKVS()
	if kvsStore == nil {
		t.Fatal("CreateTokenKVS returned nil")
	}
	defer kvsStore.Close()

	// Test basic KVS operations with context
	ctx := context.Background()
	err := kvsStore.Set(ctx, "test-key", []byte("test-value"), 3600*time.Second)
	if err != nil {
		t.Errorf("Failed to set value: %v", err)
	}

	value, err := kvsStore.Get(ctx, "test-key")
	if err != nil {
		t.Errorf("Failed to get value: %v", err)
	}
	if string(value) != "test-value" {
		t.Errorf("Expected value 'test-value', got '%s'", string(value))
	}
}

func TestDefaultFactory_CreateRateLimitKVS(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	factory := NewDefaultFactory("localhost", 4180, logger)

	kvsStore := factory.CreateRateLimitKVS()
	if kvsStore == nil {
		t.Fatal("CreateRateLimitKVS returned nil")
	}
	defer kvsStore.Close()

	// Test basic KVS operations
	ctx := context.Background()
	err := kvsStore.Set(ctx, "rate-key", []byte("1"), 3600*time.Second)
	if err != nil {
		t.Errorf("Failed to set value: %v", err)
	}

	value, err := kvsStore.Get(ctx, "rate-key")
	if err != nil {
		t.Errorf("Failed to get value: %v", err)
	}
	if string(value) != "1" {
		t.Errorf("Expected value '1', got '%s'", string(value))
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
	defer sessionKVS.Close()
	defer tokenKVS.Close()
	defer rateLimitKVS.Close()

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
	defer sessionKVS.Close()

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
	sessionKVS, _, _, err := factory.CreateKVSStores(cfg)
	if err != nil {
		t.Fatalf("CreateKVSStores failed: %v", err)
	}
	defer sessionKVS.Close()

	sessionStore := factory.CreateSessionStore(sessionKVS)

	// Create test upstream
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	cfg.Proxy.Upstream.URL = upstream.URL
	// Create a simple mock proxy handler for testing
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create middleware
	middleware, err := factory.CreateMiddleware(cfg, sessionStore, proxyHandler, logger)
	if err != nil {
		t.Fatalf("CreateMiddleware failed: %v", err)
	}
	if middleware == nil {
		t.Fatal("CreateMiddleware returned nil")
	}
}
