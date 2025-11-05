package filewatcher

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// mockListener implements ChangeListener for testing
type mockListener struct {
	mu     sync.Mutex
	events []ChangeEvent
}

func (m *mockListener) OnFileChange(event ChangeEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockListener) getEvents() []ChangeEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ChangeEvent{}, m.events...)
}

func TestWatcher_BasicFileChange(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")

	if err := os.WriteFile(tmpFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create watcher with short debounce
	watcher, err := NewWatcher(tmpFile, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer func() { _ = watcher.Close() }()

	// Add listener
	listener := &mockListener{}
	watcher.AddListener(listener)

	// Start watcher
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := watcher.Start(ctx); err != nil && err != context.Canceled {
			t.Errorf("Watcher error: %v", err)
		}
	}()

	// Wait for watcher to be ready
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(tmpFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Wait for debounce + processing
	time.Sleep(200 * time.Millisecond)

	// Check events
	events := listener.getEvents()
	if len(events) == 0 {
		t.Fatal("Expected at least one change event, got none")
	}

	event := events[0]
	if event.Error != nil {
		t.Errorf("Expected no error, got: %v", event.Error)
	}
	if event.Path == "" {
		t.Error("Expected non-empty path")
	}
}

func TestWatcher_Debounce(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")

	if err := os.WriteFile(tmpFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create watcher with 100ms debounce
	watcher, err := NewWatcher(tmpFile, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer func() { _ = watcher.Close() }()

	// Add listener
	listener := &mockListener{}
	watcher.AddListener(listener)

	// Start watcher
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := watcher.Start(ctx); err != nil && err != context.Canceled {
			t.Errorf("Watcher error: %v", err)
		}
	}()

	// Wait for watcher to be ready
	time.Sleep(50 * time.Millisecond)

	// Write multiple times rapidly (simulating editor saves)
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(tmpFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Wait for debounce to settle
	time.Sleep(200 * time.Millisecond)

	// Should have only 1 event due to debouncing
	events := listener.getEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 debounced event, got %d", len(events))
	}
}

func TestWatcher_MultipleListeners(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")

	if err := os.WriteFile(tmpFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create watcher
	watcher, err := NewWatcher(tmpFile, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer func() { _ = watcher.Close() }()

	// Add multiple listeners
	listener1 := &mockListener{}
	listener2 := &mockListener{}
	watcher.AddListener(listener1)
	watcher.AddListener(listener2)

	// Start watcher
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := watcher.Start(ctx); err != nil && err != context.Canceled {
			t.Errorf("Watcher error: %v", err)
		}
	}()

	// Wait for watcher to be ready
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(tmpFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Wait for event processing
	time.Sleep(200 * time.Millisecond)

	// Both listeners should receive the event
	events1 := listener1.getEvents()
	events2 := listener2.getEvents()

	if len(events1) == 0 {
		t.Error("Listener 1 received no events")
	}
	if len(events2) == 0 {
		t.Error("Listener 2 received no events")
	}
}

func TestWatcher_ContextCancellation(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")

	if err := os.WriteFile(tmpFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create watcher
	watcher, err := NewWatcher(tmpFile, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer func() { _ = watcher.Close() }()

	// Start watcher with cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start(ctx)
	}()

	// Wait a bit then cancel
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Should receive context.Canceled error
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Watcher did not stop after context cancellation")
	}
}
