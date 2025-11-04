package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ideamans/chatbotgate/pkg/kvs"
)

// Get retrieves a session from KVS by ID.
// Returns ErrSessionNotFound if the session doesn't exist or has expired.
func Get(store kvs.Store, id string) (*Session, error) {
	ctx := context.Background()

	data, err := store.Get(ctx, id)
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
		go Delete(store, id)
		return nil, ErrSessionNotFound
	}

	return &session, nil
}

// Set stores a session in KVS with the given ID.
// The session's ExpiresAt field is used to calculate the TTL.
func Set(store kvs.Store, id string, session *Session) error {
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

	if err := store.Set(ctx, id, data, ttl); err != nil {
		return fmt.Errorf("session: failed to set in KVS: %w", err)
	}

	return nil
}

// Delete removes a session from KVS by ID.
func Delete(store kvs.Store, id string) error {
	ctx := context.Background()

	if err := store.Delete(ctx, id); err != nil {
		return fmt.Errorf("session: failed to delete from KVS: %w", err)
	}

	return nil
}

// Count returns the number of active sessions in KVS.
func Count(store kvs.Store) (int, error) {
	ctx := context.Background()
	count, err := store.Count(ctx, "")
	if err != nil {
		return 0, fmt.Errorf("session: failed to count: %w", err)
	}
	return count, nil
}

// List returns all active sessions from KVS.
func List(store kvs.Store) ([]*Session, error) {
	ctx := context.Background()

	keys, err := store.List(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("session: failed to list keys: %w", err)
	}

	sessions := make([]*Session, 0, len(keys))
	for _, key := range keys {
		session, err := Get(store, key)
		if err != nil {
			// Skip invalid or expired sessions
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}
