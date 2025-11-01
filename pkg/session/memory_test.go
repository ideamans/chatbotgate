package session

import (
	"testing"
	"time"
)

func TestMemoryStore_SetAndGet(t *testing.T) {
	store := NewMemoryStore(1 * time.Minute)
	defer store.Close()

	now := time.Now()
	session := &Session{
		ID:            "test-session-1",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     now,
		ExpiresAt:     now.Add(1 * time.Hour),
		Authenticated: true,
	}

	// Set session
	err := store.Set("test-session-1", session)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get session
	retrieved, err := store.Get("test-session-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Email != session.Email {
		t.Errorf("retrieved.Email = %s, want %s", retrieved.Email, session.Email)
	}

	if retrieved.Provider != session.Provider {
		t.Errorf("retrieved.Provider = %s, want %s", retrieved.Provider, session.Provider)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := NewMemoryStore(1 * time.Minute)
	defer store.Close()

	_, err := store.Get("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("Get() error = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore(1 * time.Minute)
	defer store.Close()

	now := time.Now()
	session := &Session{
		ID:            "test-session-2",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     now,
		ExpiresAt:     now.Add(1 * time.Hour),
		Authenticated: true,
	}

	// Set session
	store.Set("test-session-2", session)

	// Verify it exists
	_, err := store.Get("test-session-2")
	if err != nil {
		t.Fatalf("Get() before delete error = %v", err)
	}

	// Delete session
	err = store.Delete("test-session-2")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err = store.Get("test-session-2")
	if err != ErrSessionNotFound {
		t.Errorf("Get() after delete error = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStore_ExpiredSession(t *testing.T) {
	store := NewMemoryStore(1 * time.Minute)
	defer store.Close()

	now := time.Now()
	expiredSession := &Session{
		ID:            "expired-session",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     now.Add(-2 * time.Hour),
		ExpiresAt:     now.Add(-1 * time.Hour),
		Authenticated: true,
	}

	// Set expired session
	store.Set("expired-session", expiredSession)

	// Try to get expired session
	_, err := store.Get("expired-session")
	if err != ErrSessionNotFound {
		t.Errorf("Get() for expired session error = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStore_CleanupExpired(t *testing.T) {
	// Use a short cleanup interval for testing
	store := NewMemoryStore(100 * time.Millisecond)
	defer store.Close()

	now := time.Now()

	// Add an expired session
	expiredSession := &Session{
		ID:            "expired",
		Email:         "expired@example.com",
		Provider:      "google",
		CreatedAt:     now.Add(-2 * time.Hour),
		ExpiresAt:     now.Add(-1 * time.Hour),
		Authenticated: true,
	}
	store.Set("expired", expiredSession)

	// Add a valid session
	validSession := &Session{
		ID:            "valid",
		Email:         "valid@example.com",
		Provider:      "google",
		CreatedAt:     now,
		ExpiresAt:     now.Add(1 * time.Hour),
		Authenticated: true,
	}
	store.Set("valid", validSession)

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Check that expired session was removed
	count := store.Count()
	if count != 1 {
		t.Errorf("After cleanup, store has %d sessions, want 1", count)
	}

	// Verify the valid session still exists
	_, err := store.Get("valid")
	if err != nil {
		t.Error("valid session should still exist after cleanup")
	}
}

func TestMemoryStore_Count(t *testing.T) {
	store := NewMemoryStore(1 * time.Minute)
	defer store.Close()

	if count := store.Count(); count != 0 {
		t.Errorf("initial Count() = %d, want 0", count)
	}

	now := time.Now()

	// Add sessions
	for i := 0; i < 5; i++ {
		session := &Session{
			ID:            string(rune(i)),
			Email:         "user@example.com",
			Provider:      "google",
			CreatedAt:     now,
			ExpiresAt:     now.Add(1 * time.Hour),
			Authenticated: true,
		}
		store.Set(string(rune(i)), session)
	}

	if count := store.Count(); count != 5 {
		t.Errorf("after adding 5 sessions, Count() = %d, want 5", count)
	}

	// Delete one
	store.Delete(string(rune(0)))

	if count := store.Count(); count != 4 {
		t.Errorf("after deleting 1 session, Count() = %d, want 4", count)
	}
}
