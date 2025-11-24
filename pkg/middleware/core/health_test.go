package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

func TestHealthCheck_Liveness(t *testing.T) {
	// Create minimal middleware for testing
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:   "test_session",
				Secret: "test-secret-key-32-bytes-long!",
			},
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
	}

	logger := logging.NewSimpleLogger("test", logging.LevelError, false)
	mw, err := New(cfg, nil, nil, nil, nil, nil, nil, nil, nil, logger)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/_auth/health?probe=live", nil)
	rec := httptest.NewRecorder()

	// Handle request
	mw.handleHealth(rec, req)

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Parse JSON response
	var response HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify response fields
	if response.Status != "live" {
		t.Errorf("expected status 'live', got '%s'", response.Status)
	}
	if !response.Live {
		t.Error("expected live to be true")
	}
	if response.Detail != "ok" {
		t.Errorf("expected detail 'ok', got '%s'", response.Detail)
	}
	if response.RetryAfter != nil {
		t.Errorf("expected retry_after to be nil, got %v", *response.RetryAfter)
	}
}

func TestHealthCheck_Readiness_NotReady(t *testing.T) {
	// Create minimal middleware for testing
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:   "test_session",
				Secret: "test-secret-key-32-bytes-long!",
			},
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
	}

	logger := logging.NewSimpleLogger("test", logging.LevelError, false)
	mw, err := New(cfg, nil, nil, nil, nil, nil, nil, nil, nil, logger)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// DON'T call SetReady() - middleware should be in "starting" state

	// Create test request
	req := httptest.NewRequest("GET", "/_auth/health", nil)
	rec := httptest.NewRecorder()

	// Handle request
	mw.handleHealth(rec, req)

	// Check response - should be 503 (not ready)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}

	// Check Retry-After header
	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter != "5" {
		t.Errorf("expected Retry-After header '5', got '%s'", retryAfter)
	}

	// Parse JSON response
	var response HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify response fields
	if response.Status != "starting" {
		t.Errorf("expected status 'starting', got '%s'", response.Status)
	}
	if !response.Live {
		t.Error("expected live to be true")
	}
	if response.Ready {
		t.Error("expected ready to be false")
	}
	if response.Detail != "warming up" {
		t.Errorf("expected detail 'warming up', got '%s'", response.Detail)
	}
	if response.RetryAfter == nil {
		t.Error("expected retry_after to be set")
	} else if *response.RetryAfter != 5 {
		t.Errorf("expected retry_after to be 5, got %d", *response.RetryAfter)
	}
}

func TestHealthCheck_Readiness_Ready(t *testing.T) {
	// Create minimal middleware for testing
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:   "test_session",
				Secret: "test-secret-key-32-bytes-long!",
			},
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
	}

	logger := logging.NewSimpleLogger("test", logging.LevelError, false)
	mw, err := New(cfg, nil, nil, nil, nil, nil, nil, nil, nil, logger)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Mark middleware as ready
	mw.SetReady()

	// Create test request
	req := httptest.NewRequest("GET", "/_auth/health", nil)
	rec := httptest.NewRecorder()

	// Handle request
	mw.handleHealth(rec, req)

	// Check response - should be 200 (ready)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Parse JSON response
	var response HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify response fields
	if response.Status != "ready" {
		t.Errorf("expected status 'ready', got '%s'", response.Status)
	}
	if !response.Live {
		t.Error("expected live to be true")
	}
	if !response.Ready {
		t.Error("expected ready to be true")
	}
	if response.Detail != "ok" {
		t.Errorf("expected detail 'ok', got '%s'", response.Detail)
	}
	if response.RetryAfter != nil {
		t.Errorf("expected retry_after to be nil, got %v", *response.RetryAfter)
	}
}

func TestHealthCheck_Draining(t *testing.T) {
	// Create minimal middleware for testing
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:   "test_session",
				Secret: "test-secret-key-32-bytes-long!",
			},
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
	}

	logger := logging.NewSimpleLogger("test", logging.LevelError, false)
	mw, err := New(cfg, nil, nil, nil, nil, nil, nil, nil, nil, logger)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Mark middleware as ready first
	mw.SetReady()

	// Then mark as draining
	mw.SetDraining()

	// Create test request
	req := httptest.NewRequest("GET", "/_auth/health", nil)
	rec := httptest.NewRecorder()

	// Handle request
	mw.handleHealth(rec, req)

	// Check response - should be 503 (draining)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}

	// Parse JSON response
	var response HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify response fields
	if response.Status != "draining" {
		t.Errorf("expected status 'draining', got '%s'", response.Status)
	}
	if !response.Live {
		t.Error("expected live to be true")
	}
	if response.Ready {
		t.Error("expected ready to be false")
	}
	if response.RetryAfter == nil {
		t.Error("expected retry_after to be set")
	}
}

func TestHealthCheck_SinceTimestamp(t *testing.T) {
	// Create minimal middleware for testing
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:   "test_session",
				Secret: "test-secret-key-32-bytes-long!",
			},
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
	}

	logger := logging.NewSimpleLogger("test", logging.LevelError, false)
	beforeCreate := time.Now().UTC()
	mw, _ := New(cfg, nil, nil, nil, nil, nil, nil, nil, nil, logger)
	afterCreate := time.Now().UTC().Add(1 * time.Second) // Add 1 second buffer

	// Create test request
	req := httptest.NewRequest("GET", "/_auth/health?probe=live", nil)
	rec := httptest.NewRecorder()

	// Handle request
	mw.handleHealth(rec, req)

	// Parse JSON response
	var response HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Parse the timestamp
	since, err := time.Parse(time.RFC3339, response.Since)
	if err != nil {
		t.Fatalf("failed to parse since timestamp: %v", err)
	}

	// Verify timestamp is within expected range (with some tolerance)
	if since.Before(beforeCreate.Add(-1*time.Second)) || since.After(afterCreate) {
		t.Errorf("since timestamp %v is not within expected range %v - %v", since, beforeCreate, afterCreate)
	}

	// Verify the timestamp format is RFC3339
	if response.Since == "" {
		t.Error("since timestamp should not be empty")
	}
}
