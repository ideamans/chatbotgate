package factory

import (
	"context"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

func TestNewTestingFactory(t *testing.T) {
	factory := NewTestingFactory("localhost", 4180)

	if factory == nil {
		t.Fatal("NewTestingFactory returned nil")
	}

	if factory.DefaultFactory == nil {
		t.Fatal("Expected DefaultFactory to be embedded")
	}
}

func TestNewTestingFactoryWithLogger(t *testing.T) {
	logger := logging.NewSimpleLogger("custom", logging.LevelDebug, true)
	factory := NewTestingFactoryWithLogger("example.com", 8080, logger)

	if factory == nil {
		t.Fatal("NewTestingFactoryWithLogger returned nil")
	}
}

func TestTestingFactory_CreateKVSStores(t *testing.T) {
	factory := NewTestingFactory("localhost", 4180)

	// Create config with Redis settings (should be ignored in TestingFactory)
	cfg := CreateTestConfig()
	cfg.KVS.Default.Type = "redis"

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

	// Verify all stores are memory-based (they should work without Redis)
	ctx := context.Background()
	err = sessionKVS.Set(ctx, "test", []byte("value"), 3600*time.Second)
	if err != nil {
		t.Errorf("sessionKVS should be memory-based but Set failed: %v", err)
	}

	value, err := sessionKVS.Get(ctx, "test")
	if err != nil {
		t.Errorf("sessionKVS Get failed: %v", err)
	}
	if string(value) != "value" {
		t.Errorf("Expected 'value', got '%s'", string(value))
	}
}

func TestTestingFactory_Integration(t *testing.T) {
	// Test full integration: create factory, create all components, verify they work together
	factory := NewTestingFactory("localhost", 4180)
	cfg := CreateTestConfigWithOAuth2()

	// Create KVS stores
	sessionKVS, tokenKVS, rateLimitKVS, err := factory.CreateKVSStores(cfg)
	if err != nil {
		t.Fatalf("CreateKVSStores failed: %v", err)
	}
	defer func() { _ = sessionKVS.Close() }()
	defer func() { _ = tokenKVS.Close() }()
	defer func() { _ = rateLimitKVS.Close() }()

	// Create session store
	sessionStore := factory.CreateSessionStore(sessionKVS)
	if sessionStore == nil {
		t.Fatal("CreateSessionStore returned nil")
	}

	// Create translator
	translator := factory.CreateTranslator()
	if translator == nil {
		t.Fatal("CreateTranslator returned nil")
	}

	// Create authz checker
	authzChecker := factory.CreateAuthzChecker(cfg.Authorization)
	if authzChecker == nil {
		t.Fatal("CreateAuthzChecker returned nil")
	}

	// Create OAuth2 manager
	oauth2Manager := factory.CreateOAuth2Manager(cfg.OAuth2, cfg.Server, "localhost", 4180)
	if oauth2Manager == nil {
		t.Fatal("CreateOAuth2Manager returned nil")
	}
}
