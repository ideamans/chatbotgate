package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ideamans/chatbotgate/pkg/kvs"
)

// KVSStore implements session storage using a KVS backend.
// It wraps any kvs.Store implementation (Memory, LevelDB, Redis).
type KVSStore struct {
	kvs kvs.Store
}

// NewKVSStore creates a new KVS-based session store.
func NewKVSStore(kvsStore kvs.Store) *KVSStore {
	return &KVSStore{
		kvs: kvsStore,
	}
}

// Get retrieves a session by ID.
func (s *KVSStore) Get(id string) (*Session, error) {
	ctx := context.Background()

	data, err := s.kvs.Get(ctx, id)
	if err != nil {
		if errors.Is(err, kvs.ErrNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("session: failed to get from KVS: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("session: failed to unmarshal: %w", err)
	}

	// Check if session is valid
	if !session.IsValid() {
		// Delete expired session asynchronously
		go s.Delete(id)
		return nil, ErrSessionNotFound
	}

	return &session, nil
}

// Set stores a session with the given ID.
func (s *KVSStore) Set(id string, session *Session) error {
	ctx := context.Background()

	// Calculate TTL until expiration
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return errors.New("session: session already expired")
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("session: failed to marshal: %w", err)
	}

	if err := s.kvs.Set(ctx, id, data, ttl); err != nil {
		return fmt.Errorf("session: failed to set in KVS: %w", err)
	}

	return nil
}

// Delete removes a session by ID.
func (s *KVSStore) Delete(id string) error {
	ctx := context.Background()

	if err := s.kvs.Delete(ctx, id); err != nil {
		return fmt.Errorf("session: failed to delete from KVS: %w", err)
	}

	return nil
}

// Close closes the underlying KVS store.
func (s *KVSStore) Close() error {
	return s.kvs.Close()
}

// Count returns the number of active sessions.
// This is a convenience method for monitoring.
func (s *KVSStore) Count() (int, error) {
	ctx := context.Background()
	count, err := s.kvs.Count(ctx, "")
	if err != nil {
		return 0, fmt.Errorf("session: failed to count: %w", err)
	}
	return count, nil
}

// List returns all active sessions.
// This is a convenience method for admin interfaces.
func (s *KVSStore) List() ([]*Session, error) {
	ctx := context.Background()

	keys, err := s.kvs.List(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("session: failed to list keys: %w", err)
	}

	sessions := make([]*Session, 0, len(keys))
	for _, key := range keys {
		session, err := s.Get(key)
		if err != nil {
			// Skip invalid or expired sessions
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}
