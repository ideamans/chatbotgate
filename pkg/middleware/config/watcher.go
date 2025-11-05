package config

import (
	"fmt"
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a configuration file for changes
type Watcher struct {
	path     string
	watcher  *fsnotify.Watcher
	callback func(*Config) error
	loader   *FileLoader
	mu       sync.Mutex
	stopped  bool
}

// NewWatcher creates a new configuration file watcher
func NewWatcher(path string, callback func(*Config) error) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	return &Watcher{
		path:     path,
		watcher:  fsWatcher,
		callback: callback,
		loader:   NewFileLoader(path),
		stopped:  false,
	}, nil
}

// Start begins watching the configuration file
func (w *Watcher) Start() error {
	err := w.watcher.Add(w.path)
	if err != nil {
		return fmt.Errorf("failed to watch file: %w", err)
	}

	go w.watch()
	return nil
}

// watch monitors file system events
func (w *Watcher) watch() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Handle write and create events
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				w.mu.Lock()
				if w.stopped {
					w.mu.Unlock()
					return
				}
				w.mu.Unlock()

				// Reload configuration
				cfg, err := w.loader.Load()
				if err != nil {
					log.Printf("Failed to reload configuration: %v", err)
					continue
				}

				// Call callback
				if w.callback != nil {
					if err := w.callback(cfg); err != nil {
						log.Printf("Configuration reload callback failed: %v", err)
					} else {
						log.Printf("Configuration reloaded successfully")
					}
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// Stop stops watching the configuration file
func (w *Watcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return nil
	}

	w.stopped = true
	return w.watcher.Close()
}
