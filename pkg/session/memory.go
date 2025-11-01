package session

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrSessionNotFound is returned when a session is not found
	ErrSessionNotFound = errors.New("session not found")
)

// MemoryStore is an in-memory session store
type MemoryStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	// Cleanup interval for expired sessions
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// NewMemoryStore creates a new in-memory session store
func NewMemoryStore(cleanupInterval time.Duration) *MemoryStore {
	store := &MemoryStore{
		sessions:        make(map[string]*Session),
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go store.cleanupExpired()

	return store
}

// Get retrieves a session by ID
func (s *MemoryStore) Get(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[id]
	if !exists {
		return nil, ErrSessionNotFound
	}

	// Check if session is expired
	if !session.IsValid() {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// Set stores a session
func (s *MemoryStore) Set(id string, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[id] = session
	return nil
}

// Delete removes a session
func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
	return nil
}

// cleanupExpired removes expired sessions periodically
func (s *MemoryStore) cleanupExpired() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for id, session := range s.sessions {
				if !session.IsValid() && now.After(session.ExpiresAt) {
					delete(s.sessions, id)
				}
			}
			s.mu.Unlock()
		case <-s.stopCleanup:
			return
		}
	}
}

// Close stops the cleanup goroutine
func (s *MemoryStore) Close() {
	close(s.stopCleanup)
}

// Count returns the number of sessions (for testing)
func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}
