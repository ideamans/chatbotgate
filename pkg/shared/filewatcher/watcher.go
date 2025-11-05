package filewatcher

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ChangeEvent represents a file change event
type ChangeEvent struct {
	Path      string    // Path to the changed file
	Timestamp time.Time // Time of the change
	Error     error     // Error if any occurred during processing
}

// ChangeListener is an interface for receiving file change notifications
type ChangeListener interface {
	OnFileChange(event ChangeEvent)
}

// Watcher monitors file changes and notifies listeners with debounce support
type Watcher struct {
	watcher       *fsnotify.Watcher
	listeners     []ChangeListener
	filePath      string
	debounceDelay time.Duration
	mu            sync.RWMutex
}

// NewWatcher creates a new file watcher with the specified debounce delay
func NewWatcher(filePath string, debounceDelay time.Duration) (*Watcher, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	// Add the file to watch
	if err := fsWatcher.Add(absPath); err != nil {
		_ = fsWatcher.Close()
		return nil, fmt.Errorf("failed to add file to watcher: %w", err)
	}

	return &Watcher{
		watcher:       fsWatcher,
		listeners:     make([]ChangeListener, 0),
		filePath:      absPath,
		debounceDelay: debounceDelay,
	}, nil
}

// AddListener adds a listener to receive file change notifications
func (w *Watcher) AddListener(listener ChangeListener) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.listeners = append(w.listeners, listener)
}

// Start begins watching for file changes
// This is a blocking call and should typically be run in a goroutine
func (w *Watcher) Start(ctx context.Context) error {
	// Channel for debounced events
	debouncedEvents := make(chan fsnotify.Event, 1)

	// Start debounce goroutine
	go w.debounce(ctx, debouncedEvents)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-w.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher events channel closed")
			}

			// Filter events for our target file
			// Some editors create temp files, so we need to check the path
			eventPath, err := filepath.Abs(event.Name)
			if err != nil {
				continue
			}

			if eventPath != w.filePath {
				continue
			}

			// Only process Write and Create events
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create {
				// Send to debounce channel (non-blocking)
				select {
				case debouncedEvents <- event:
				default:
					// Already have a pending event, skip
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher errors channel closed")
			}

			// Notify listeners about the error
			w.notifyListeners(ChangeEvent{
				Path:      w.filePath,
				Timestamp: time.Now(),
				Error:     err,
			})
		}
	}
}

// Close stops the watcher and releases resources
func (w *Watcher) Close() error {
	return w.watcher.Close()
}

// debounce processes events with a delay to avoid rapid-fire notifications
// This is useful when editors save files multiple times in quick succession
func (w *Watcher) debounce(ctx context.Context, events <-chan fsnotify.Event) {
	var timer *time.Timer

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return

		case event := <-events:
			// Reset or create timer
			if timer != nil {
				timer.Stop()
			}

			timer = time.AfterFunc(w.debounceDelay, func() {
				// Notify listeners after debounce delay
				w.notifyListeners(ChangeEvent{
					Path:      event.Name,
					Timestamp: time.Now(),
					Error:     nil,
				})
			})
		}
	}
}

// notifyListeners sends the event to all registered listeners
func (w *Watcher) notifyListeners(event ChangeEvent) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, listener := range w.listeners {
		// Call listener in a separate goroutine to avoid blocking
		go listener.OnFileChange(event)
	}
}
