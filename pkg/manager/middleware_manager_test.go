package manager

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/factory"
	"github.com/ideamans/chatbotgate/pkg/logging"
	"github.com/ideamans/chatbotgate/pkg/proxy"
	"github.com/ideamans/chatbotgate/pkg/session"
)

// createTestFactory creates a test factory for the given host and port
func createTestFactory(host string, port int, logger logging.Logger) factory.Factory {
	return factory.NewDefaultFactory(host, port, logger)
}

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
			Upstream: config.UpstreamConfig{
				URL: "http://localhost:8080",
			},
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
				},
			},
		},
		Authorization: config.AuthorizationConfig{
			Allowed: allowedEmails,
		},
	}
}

func TestSingleDomainManager_New(t *testing.T) {
	cfg := createTestConfig("Test Service", []string{"user@example.com"})
	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	f := createTestFactory("localhost", 4180, logger)

	manager, err := New(ManagerConfig{
		Config:       cfg,
		Factory:      f,
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

func TestSingleDomainManager_New_Validation(t *testing.T) {
	cfg := createTestConfig("Test Service", nil)
	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()
	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)
	f := createTestFactory("localhost", 4180, logger)

	tests := []struct {
		name      string
		config    ManagerConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: ManagerConfig{
				Config:       cfg,
				Factory:      f,
				SessionStore: sessionStore,
				Logger:       logger,
			},
			wantError: false,
		},
		{
			name: "missing config",
			config: ManagerConfig{
				Config:       nil,
				Factory:      f,
				SessionStore: sessionStore,
				Logger:       logger,
			},
			wantError: true,
		},
		{
			name: "missing session store",
			config: ManagerConfig{
				Config:       cfg,
				Factory:      f,
				SessionStore: nil,
				Logger:       logger,
			},
			wantError: true,
		},
		{
			name: "missing logger",
			config: ManagerConfig{
				Config:       cfg,
				Factory:      f,
				SessionStore: sessionStore,
				Logger:       nil,
			},
			wantError: true,
		},
		{
			name: "missing factory",
			config: ManagerConfig{
				Config:       cfg,
				Factory:      nil,
				SessionStore: sessionStore,
				Logger:       logger,
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

func TestSingleDomainManager_ServeHTTP(t *testing.T) {
	// Create a test backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	cfg := createTestConfig("Test Service", nil)
	cfg.Proxy.Upstream.URL = backend.URL

	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	proxyHandler, err := proxy.NewHandler(backend.URL)
	if err != nil {
		t.Fatalf("Failed to create proxy handler: %v", err)
	}

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	f := createTestFactory("localhost", 4180, logger)

	manager, err := New(ManagerConfig{
		Config:       cfg,
		Factory:      f,
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

func TestSingleDomainManager_Reload(t *testing.T) {
	cfg1 := createTestConfig("Service V1", []string{"user1@example.com"})
	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	f := createTestFactory("localhost", 4180, logger)

	manager, err := New(ManagerConfig{
		Config:       cfg1,
		Factory:      f,
		
		
		SessionStore: sessionStore,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Reload with new config
	cfg2 := createTestConfig("Service V2", []string{"user2@example.com"})
	err = manager.Reload(cfg2)
	if err != nil {
		t.Fatalf("Failed to reload: %v", err)
	}

	// Verify manager is still functional by making a request
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	manager.ServeHTTP(rec, req)

	// Should still work (redirect to login)
	if rec.Code != http.StatusFound && rec.Code != http.StatusOK {
		t.Errorf("ServeHTTP after reload status = %d, want redirect or OK", rec.Code)
	}
}

func TestSingleDomainManager_Reload_InvalidConfig(t *testing.T) {
	cfg1 := createTestConfig("Service V1", nil)
	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	f := createTestFactory("localhost", 4180, logger)

	manager, err := New(ManagerConfig{
		Config:       cfg1,
		Factory:      f,
		
		
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

	// Verify old config is still active by making a request
	// If old middleware is still working, it should redirect to login
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	manager.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound && rec.Code != http.StatusOK {
		t.Errorf("After failed reload, ServeHTTP status = %d, want redirect or OK", rec.Code)
	}
}

func TestSingleDomainManager_ConcurrentAccess(t *testing.T) {
	cfg := createTestConfig("Test Service", nil)
	sessionStore := session.NewMemoryStore(1 * time.Minute)
	defer sessionStore.Close()

	logger := logging.NewSimpleLogger("test", logging.LevelInfo, false)

	f := createTestFactory("localhost", 4180, logger)

	manager, err := New(ManagerConfig{
		Config:       cfg,
		Factory:      f,
		
		
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

