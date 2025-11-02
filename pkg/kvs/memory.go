package kvs

import (
	"context"
	"strings"
	"sync"
	"time"
)

// memoryItem represents a stored item with expiration.
type memoryItem struct {
	value     []byte
	expiresAt time.Time // Zero value means no expiration
}

// MemoryStore is an in-memory implementation of Store.
// It stores data in a map and runs a background goroutine to clean up expired items.
// Data is volatile and will be lost when the process restarts.
type MemoryStore struct {
	prefix          string
	items           map[string]*memoryItem
	mu              sync.RWMutex
	closed          bool
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	cleanupDone     chan struct{}
}

// NewMemoryStore creates a new in-memory KVS store.
func NewMemoryStore(prefix string, cfg MemoryConfig) (*MemoryStore, error) {
	cleanupInterval := cfg.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 5 * time.Minute
	}

	store := &MemoryStore{
		prefix:          prefix,
		items:           make(map[string]*memoryItem),
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
		cleanupDone:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go store.cleanupLoop()

	return store, nil
}

// prefixedKey returns the key with prefix prepended.
func (m *MemoryStore) prefixedKey(key string) string {
	if m.prefix == "" {
		return key
	}
	return m.prefix + key
}

// Get retrieves a value by key.
func (m *MemoryStore) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrClosed
	}

	item, exists := m.items[m.prefixedKey(key)]
	if !exists {
		return nil, ErrNotFound
	}

	// Check expiration
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		return nil, ErrNotFound
	}

	// Return a copy to prevent external modifications
	value := make([]byte, len(item.value))
	copy(value, item.value)
	return value, nil
}

// Set stores a value with optional TTL.
func (m *MemoryStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrClosed
	}

	// Make a copy to prevent external modifications
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	item := &memoryItem{
		value: valueCopy,
	}

	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}

	m.items[m.prefixedKey(key)] = item
	return nil
}

// Delete removes a key.
func (m *MemoryStore) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrClosed
	}

	delete(m.items, m.prefixedKey(key))
	return nil
}

// Exists checks if a key exists and has not expired.
func (m *MemoryStore) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return false, ErrClosed
	}

	item, exists := m.items[m.prefixedKey(key)]
	if !exists {
		return false, nil
	}

	// Check expiration
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		return false, nil
	}

	return true, nil
}

// List returns all keys matching a prefix.
func (m *MemoryStore) List(ctx context.Context, keyPrefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrClosed
	}

	fullPrefix := m.prefixedKey(keyPrefix)
	var keys []string
	now := time.Now()

	for key, item := range m.items {
		// Check if key matches prefix
		if !strings.HasPrefix(key, fullPrefix) {
			continue
		}

		// Skip expired items
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			continue
		}

		// Remove store prefix to return clean key
		cleanKey := key
		if m.prefix != "" && strings.HasPrefix(key, m.prefix) {
			cleanKey = strings.TrimPrefix(key, m.prefix)
		}

		keys = append(keys, cleanKey)
	}

	return keys, nil
}

// Count returns the number of keys matching a prefix.
func (m *MemoryStore) Count(ctx context.Context, prefix string) (int, error) {
	keys, err := m.List(ctx, prefix)
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}

// Close closes the store and stops the cleanup goroutine.
func (m *MemoryStore) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return ErrClosed
	}
	m.closed = true
	m.mu.Unlock()

	// Stop cleanup goroutine
	close(m.stopCleanup)
	<-m.cleanupDone

	// Clear all items
	m.mu.Lock()
	m.items = nil
	m.mu.Unlock()

	return nil
}

// cleanupLoop runs periodically to remove expired items.
func (m *MemoryStore) cleanupLoop() {
	defer close(m.cleanupDone)

	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.stopCleanup:
			return
		}
	}
}

// cleanup removes expired items from the store.
func (m *MemoryStore) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return
	}

	now := time.Now()
	for key, item := range m.items {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			delete(m.items, key)
		}
	}
}
