package manager

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
	"github.com/ideamans/multi-oauth2-proxy/pkg/logging"
	"github.com/ideamans/multi-oauth2-proxy/pkg/proxy"
	"github.com/ideamans/multi-oauth2-proxy/pkg/session"
)

// createTestConfig creates a minimal valid configuration for testing
func createTestConfig(serviceName string, allowedEmails []string) *config.Config {
	return &config.Config{
		Service: config.ServiceConfig{
			Name:        serviceName,
			Description: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Proxy: config.ProxyConfig{
			Upstream: "http://localhost:8080",
		},
		Session: config.SessionConfig{
			CookieName:     "_test_session",
			CookieSecret:   "test-secret-key-with-32-characters",
			CookieExpire:   "1h",
			CookieSecure:   false,
			CookieHTTPOnly: true,
			CookieSameSite: "lax",
		},
		OAuth2: config.OAuth2Config{
			Providers: []config.OAuth2Provider{
				{
					Name:         "test",
					Type:         "google",
					DisplayName:  "Test Provider",
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					Enabled:      true,
				},
			},
		},
		Authorization: config.AuthorizationConfig{
			Allowed: allowedEmails,
		},
	}
}

func TestMiddlewareManager_New(t *testing.T) {
	cfg := createTestConfig("Test Service", []string{"user@example.com"})
	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	manager, err := New(ManagerConfig{
		Config:       cfg,
		Host:         "localhost",
		Port:         4180,
		SessionStore: sessionStore,
		Logger:       logger,
	})

	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Manager is nil")
	}

	// Check that current middleware is set
	if manager.current.Load() == nil {
		t.Error("Current middleware is not set")
	}
}

func TestMiddlewareManager_New_Validation(t *testing.T) {
	cfg := createTestConfig("Test Service", nil)
	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	tests := []struct {
		name      string
		config    ManagerConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: ManagerConfig{
				Config:       cfg,
				Host:         "localhost",
				Port:         4180,
				SessionStore: sessionStore,
				Logger:       logger,
			},
			wantError: false,
		},
		{
			name: "missing config",
			config: ManagerConfig{
				Config:       nil,
				Host:         "localhost",
				Port:         4180,
				SessionStore: sessionStore,
				Logger:       logger,
			},
			wantError: true,
		},
		{
			name: "missing session store",
			config: ManagerConfig{
				Config:       cfg,
				Host:         "localhost",
				Port:         4180,
				SessionStore: nil,
				Logger:       logger,
			},
			wantError: true,
		},
		{
			name: "missing logger",
			config: ManagerConfig{
				Config:       cfg,
				Host:         "localhost",
				Port:         4180,
				SessionStore: sessionStore,
				Logger:       nil,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("New() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestMiddlewareManager_ServeHTTP(t *testing.T) {
	// Create a test backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	cfg := createTestConfig("Test Service", nil)
	cfg.Proxy.Upstream = backend.URL

	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	proxyHandler, err := proxy.NewHandler(backend.URL)
	if err != nil {
		t.Fatalf("Failed to create proxy handler: %v", err)
	}

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	manager, err := New(ManagerConfig{
		Config:       cfg,
		Host:         "localhost",
		Port:         4180,
		SessionStore: sessionStore,
		ProxyHandler: proxyHandler,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test that ServeHTTP delegates to the middleware
	req := httptest.NewRequest(http.MethodGet, "/_auth/login", nil)
	rec := httptest.NewRecorder()

	manager.ServeHTTP(rec, req)

	// Should get login page (status 200)
	if rec.Code != http.StatusOK {
		t.Errorf("ServeHTTP status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMiddlewareManager_Reload(t *testing.T) {
	cfg1 := createTestConfig("Service V1", []string{"user1@example.com"})
	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	manager, err := New(ManagerConfig{
		Config:       cfg1,
		Host:         "localhost",
		Port:         4180,
		SessionStore: sessionStore,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Get initial config
	initialConfig := manager.GetConfig()
	if initialConfig.Service.Name != "Service V1" {
		t.Errorf("Initial service name = %s, want Service V1", initialConfig.Service.Name)
	}

	// Reload with new config
	cfg2 := createTestConfig("Service V2", []string{"user2@example.com"})
	err = manager.Reload(cfg2)
	if err != nil {
		t.Fatalf("Failed to reload: %v", err)
	}

	// Verify new config is active
	newConfig := manager.GetConfig()
	if newConfig.Service.Name != "Service V2" {
		t.Errorf("Reloaded service name = %s, want Service V2", newConfig.Service.Name)
	}

	// Verify allowed list changed
	if len(newConfig.Authorization.Allowed) != 1 || newConfig.Authorization.Allowed[0] != "user2@example.com" {
		t.Errorf("Reloaded allowed list = %v, want [user2@example.com]", newConfig.Authorization.Allowed)
	}
}

func TestMiddlewareManager_Reload_InvalidConfig(t *testing.T) {
	cfg1 := createTestConfig("Service V1", nil)
	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	manager, err := New(ManagerConfig{
		Config:       cfg1,
		Host:         "localhost",
		Port:         4180,
		SessionStore: sessionStore,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Try to reload with invalid config (missing service name)
	invalidCfg := createTestConfig("", nil)
	err = manager.Reload(invalidCfg)
	if err == nil {
		t.Error("Reload with invalid config should fail")
	}

	// Verify old config is still active
	currentConfig := manager.GetConfig()
	if currentConfig.Service.Name != "Service V1" {
		t.Errorf("After failed reload, service name = %s, want Service V1", currentConfig.Service.Name)
	}
}

func TestMiddlewareManager_ConcurrentAccess(t *testing.T) {
	cfg := createTestConfig("Test Service", nil)
	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	manager, err := New(ManagerConfig{
		Config:       cfg,
		Host:         "localhost",
		Port:         4180,
		SessionStore: sessionStore,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Concurrent access test
	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	// Start multiple goroutines that reload and serve HTTP concurrently
	for i := 0; i < 10; i++ {
		// Reloader goroutine
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()
			newCfg := createTestConfig("Service", []string{"user@example.com"})
			if err := manager.Reload(newCfg); err != nil {
				errChan <- err
			}
		}(i)

		// HTTP serving goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/_auth/login", nil)
			rec := httptest.NewRecorder()
			manager.ServeHTTP(rec, req)
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestMiddlewareManager_GetConfig(t *testing.T) {
	cfg := createTestConfig("Test Service", []string{"user@example.com"})
	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	manager, err := New(ManagerConfig{
		Config:       cfg,
		Host:         "localhost",
		Port:         4180,
		SessionStore: sessionStore,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Get config
	retrievedCfg := manager.GetConfig()

	// Verify it's a copy (modifying it shouldn't affect the manager)
	retrievedCfg.Service.Name = "Modified"

	// Get config again
	retrievedCfg2 := manager.GetConfig()
	if retrievedCfg2.Service.Name != "Test Service" {
		t.Errorf("GetConfig should return a copy, got %s", retrievedCfg2.Service.Name)
	}
}
