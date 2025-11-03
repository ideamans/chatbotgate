package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name        string
		upstreamURL string
		wantErr     bool
	}{
		{
			name:        "valid URL",
			upstreamURL: "http://localhost:8080",
			wantErr:     false,
		},
		{
			name:        "valid URL with path",
			upstreamURL: "http://localhost:8080/api",
			wantErr:     false,
		},
		{
			name:        "invalid URL",
			upstreamURL: "://invalid",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewHandler(tt.upstreamURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && handler == nil {
				t.Error("NewHandler() returned nil handler")
			}
		})
	}
}

func TestHandler_ServeHTTP(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	// Create proxy handler
	handler, err := NewHandler(backend.URL)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(rec, req)

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "backend response" {
		t.Errorf("ServeHTTP() body = %s, want %s", rec.Body.String(), "backend response")
	}
}

func TestAddAuthHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	email := "user@example.com"
	provider := "google"

	AddAuthHeaders(req, email, provider)

	// Check headers
	if got := req.Header.Get("X-ChatbotGate-User"); got != email {
		t.Errorf("X-ChatbotGate-User = %s, want %s", got, email)
	}

	if got := req.Header.Get("X-ChatbotGate-Email"); got != email {
		t.Errorf("X-ChatbotGate-Email = %s, want %s", got, email)
	}

	if got := req.Header.Get("X-Auth-Provider"); got != provider {
		t.Errorf("X-Auth-Provider = %s, want %s", got, provider)
	}
}

func TestHandler_ProxyRequest(t *testing.T) {
	// Create a test backend that echoes headers
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back the auth headers
		w.Header().Set("X-Echo-User", r.Header.Get("X-ChatbotGate-User"))
		w.Header().Set("X-Echo-Provider", r.Header.Get("X-Auth-Provider"))
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	handler, err := NewHandler(backend.URL)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	// Create request with auth headers
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	AddAuthHeaders(req, "user@example.com", "google")

	rec := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(rec, req)

	// Verify headers were forwarded
	if got := rec.Header().Get("X-Echo-User"); got != "user@example.com" {
		t.Errorf("Backend received X-ChatbotGate-User = %s, want user@example.com", got)
	}

	if got := rec.Header().Get("X-Echo-Provider"); got != "google" {
		t.Errorf("Backend received X-Auth-Provider = %s, want google", got)
	}
}

func TestNewHandlerWithHosts(t *testing.T) {
	tests := []struct {
		name            string
		defaultUpstream string
		hosts           map[string]string
		wantErr         bool
	}{
		{
			name:            "valid default and hosts",
			defaultUpstream: "http://default:8080",
			hosts: map[string]string{
				"app1.example.com": "http://backend1:8080",
				"app2.example.com": "http://backend2:8080",
			},
			wantErr: false,
		},
		{
			name:            "empty hosts map",
			defaultUpstream: "http://default:8080",
			hosts:           map[string]string{},
			wantErr:         false,
		},
		{
			name:            "invalid default upstream",
			defaultUpstream: "://invalid",
			hosts: map[string]string{
				"app.example.com": "http://backend:8080",
			},
			wantErr: true,
		},
		{
			name:            "invalid host upstream",
			defaultUpstream: "http://default:8080",
			hosts: map[string]string{
				"app.example.com": "://invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewHandlerWithHosts(tt.defaultUpstream, tt.hosts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHandlerWithHosts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && handler == nil {
				t.Error("NewHandlerWithHosts() returned nil handler")
			}
		})
	}
}

func TestHandler_HostBasedRouting(t *testing.T) {
	// Create two different backend servers
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend1 response"))
	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend2 response"))
	}))
	defer backend2.Close()

	defaultBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("default backend response"))
	}))
	defer defaultBackend.Close()

	// Create proxy handler with host routing
	handler, err := NewHandlerWithHosts(defaultBackend.URL, map[string]string{
		"app1.example.com": backend1.URL,
		"app2.example.com": backend2.URL,
	})
	if err != nil {
		t.Fatalf("NewHandlerWithHosts() error = %v", err)
	}

	tests := []struct {
		name         string
		host         string
		wantResponse string
	}{
		{
			name:         "route to backend1",
			host:         "app1.example.com",
			wantResponse: "backend1 response",
		},
		{
			name:         "route to backend2",
			host:         "app2.example.com",
			wantResponse: "backend2 response",
		},
		{
			name:         "route to default backend",
			host:         "unknown.example.com",
			wantResponse: "default backend response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("ServeHTTP() status = %d, want %d", rec.Code, http.StatusOK)
			}

			if rec.Body.String() != tt.wantResponse {
				t.Errorf("ServeHTTP() body = %s, want %s", rec.Body.String(), tt.wantResponse)
			}
		})
	}
}
