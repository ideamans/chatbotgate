package server

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

func TestNewDummyUpstream(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelDebug, false)

	// Create dummy upstream
	dummy := NewDummyUpstream(logger)
	if dummy == nil {
		t.Fatal("Expected dummy upstream to be created")
	}
	defer dummy.Stop()

	// Verify URL is valid
	url := dummy.URL()
	if !strings.HasPrefix(url, "http://127.0.0.1:") {
		t.Errorf("Expected URL to start with http://127.0.0.1:, got %s", url)
	}

	// Test HTTP request
	resp, err := http.Get(url + "/test/path")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Dummy Upstream Server") {
		t.Errorf("Expected body to contain 'Dummy Upstream Server', got %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "/test/path") {
		t.Errorf("Expected body to contain request path '/test/path', got %s", bodyStr)
	}
}

func TestNewDummyUpstreamWithNilLogger(t *testing.T) {
	// Should not panic with nil logger
	dummy := NewDummyUpstream(nil)
	if dummy == nil {
		t.Fatal("Expected dummy upstream to be created even with nil logger")
	}
	defer dummy.Stop()

	// Verify it works
	resp, err := http.Get(dummy.URL())
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestDummyUpstreamStop(t *testing.T) {
	logger := logging.NewSimpleLogger("test", logging.LevelDebug, false)

	dummy := NewDummyUpstream(logger)
	if dummy == nil {
		t.Fatal("Expected dummy upstream to be created")
	}

	url := dummy.URL()

	// Verify it's running
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to make request before stop: %v", err)
	}
	_ = resp.Body.Close()

	// Stop the server
	dummy.Stop()

	// Give it a moment to stop
	time.Sleep(100 * time.Millisecond)

	// Verify it's stopped (connection should fail)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	_, err = client.Get(url)
	if err == nil {
		t.Error("Expected connection error after stop, but request succeeded")
	}
}

func TestDummyUpstreamStopNil(t *testing.T) {
	// Should not panic when stopping nil
	var dummy *DummyUpstream
	dummy.Stop() // Should not panic
}
