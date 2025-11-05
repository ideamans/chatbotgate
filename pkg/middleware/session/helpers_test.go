package session

import (
	"errors"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
)

func TestHelpers_SetAndGet(t *testing.T) {
	// Create memory store for testing
	store, err := kvs.NewMemoryStore("test-helpers", kvs.MemoryConfig{})
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	now := time.Now()
	testSession := &Session{
		ID:            "test-session-id",
		Email:         "user@example.com",
		Name:          "Test User",
		Provider:      "google",
		CreatedAt:     now,
		ExpiresAt:     now.Add(1 * time.Hour),
		Authenticated: true,
	}

	// Test Set
	t.Run("Set valid session", func(t *testing.T) {
		err := Set(store, testSession.ID, testSession)
		if err != nil {
			t.Errorf("Set() error = %v, want nil", err)
		}
	})

	// Test Get
	t.Run("Get existing session", func(t *testing.T) {
		got, err := Get(store, testSession.ID)
		if err != nil {
			t.Errorf("Get() error = %v, want nil", err)
		}
		if got == nil {
			t.Fatal("Get() returned nil session")
		}
		if got.ID != testSession.ID {
			t.Errorf("Get() ID = %v, want %v", got.ID, testSession.ID)
		}
		if got.Email != testSession.Email {
			t.Errorf("Get() Email = %v, want %v", got.Email, testSession.Email)
		}
		if got.Name != testSession.Name {
			t.Errorf("Get() Name = %v, want %v", got.Name, testSession.Name)
		}
		if got.Provider != testSession.Provider {
			t.Errorf("Get() Provider = %v, want %v", got.Provider, testSession.Provider)
		}
	})

	// Test Get non-existent session
	t.Run("Get non-existent session", func(t *testing.T) {
		_, err := Get(store, "non-existent-id")
		if !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("Get() error = %v, want ErrSessionNotFound", err)
		}
	})
}

func TestHelpers_SetExpired(t *testing.T) {
	store, err := kvs.NewMemoryStore("test-expired", kvs.MemoryConfig{})
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	now := time.Now()
	expiredSession := &Session{
		ID:            "expired-session",
		Email:         "expired@example.com",
		Provider:      "google",
		CreatedAt:     now.Add(-2 * time.Hour),
		ExpiresAt:     now.Add(-1 * time.Hour), // Already expired
		Authenticated: true,
	}

	// Trying to set an already expired session should fail
	err = Set(store, expiredSession.ID, expiredSession)
	if err == nil {
		t.Error("Set() with expired session should return error")
	}
	if err != nil && err.Error() != "session: session already expired" {
		t.Errorf("Set() error = %v, want 'session already expired'", err)
	}
}

func TestHelpers_GetExpiredSession(t *testing.T) {
	store, err := kvs.NewMemoryStore("test-get-expired", kvs.MemoryConfig{})
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	now := time.Now()

	// Create a session that will expire very soon
	session := &Session{
		ID:            "soon-expired",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     now,
		ExpiresAt:     now.Add(100 * time.Millisecond), // Expires in 100ms
		Authenticated: true,
	}

	// Store the session
	err = Set(store, session.ID, session)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Try to get the expired session
	got, err := Get(store, session.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Get() expired session error = %v, want ErrSessionNotFound", err)
	}
	if got != nil {
		t.Errorf("Get() expired session = %v, want nil", got)
	}
}

func TestHelpers_Delete(t *testing.T) {
	store, err := kvs.NewMemoryStore("test-delete", kvs.MemoryConfig{})
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	now := time.Now()
	session := &Session{
		ID:            "delete-me",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     now,
		ExpiresAt:     now.Add(1 * time.Hour),
		Authenticated: true,
	}

	// Store session
	err = Set(store, session.ID, session)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify it exists
	_, err = Get(store, session.ID)
	if err != nil {
		t.Fatalf("Get() before delete error = %v", err)
	}

	// Delete session
	err = Delete(store, session.ID)
	if err != nil {
		t.Errorf("Delete() error = %v, want nil", err)
	}

	// Verify it's gone
	_, err = Get(store, session.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Get() after delete error = %v, want ErrSessionNotFound", err)
	}

	// Deleting non-existent session should not error
	err = Delete(store, "non-existent")
	if err != nil {
		t.Errorf("Delete() non-existent session error = %v, want nil", err)
	}
}

func TestHelpers_Count(t *testing.T) {
	store, err := kvs.NewMemoryStore("test-count", kvs.MemoryConfig{})
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	now := time.Now()

	// Initially should be 0
	count, err := Count(store)
	if err != nil {
		t.Errorf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() initial = %d, want 0", count)
	}

	// Add 3 sessions
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:            "session-" + string(rune('A'+i)),
			Email:         "user@example.com",
			Provider:      "google",
			CreatedAt:     now,
			ExpiresAt:     now.Add(1 * time.Hour),
			Authenticated: true,
		}
		err = Set(store, session.ID, session)
		if err != nil {
			t.Fatalf("Set() session %d error = %v", i, err)
		}
	}

	// Should now be 3
	count, err = Count(store)
	if err != nil {
		t.Errorf("Count() after adding error = %v", err)
	}
	if count != 3 {
		t.Errorf("Count() after adding = %d, want 3", count)
	}

	// Delete one
	err = Delete(store, "session-A")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Should now be 2
	count, err = Count(store)
	if err != nil {
		t.Errorf("Count() after deleting error = %v", err)
	}
	if count != 2 {
		t.Errorf("Count() after deleting = %d, want 2", count)
	}
}

func TestHelpers_List(t *testing.T) {
	store, err := kvs.NewMemoryStore("test-list", kvs.MemoryConfig{})
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	now := time.Now()

	// Initially should be empty
	sessions, err := List(store)
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("List() initial length = %d, want 0", len(sessions))
	}

	// Add 3 sessions
	expectedIDs := make(map[string]bool)
	for i := 0; i < 3; i++ {
		id := "session-" + string(rune('A'+i))
		expectedIDs[id] = true
		session := &Session{
			ID:            id,
			Email:         "user" + string(rune('A'+i)) + "@example.com",
			Provider:      "google",
			CreatedAt:     now,
			ExpiresAt:     now.Add(1 * time.Hour),
			Authenticated: true,
		}
		err = Set(store, session.ID, session)
		if err != nil {
			t.Fatalf("Set() session %d error = %v", i, err)
		}
	}

	// List all sessions
	sessions, err = List(store)
	if err != nil {
		t.Errorf("List() after adding error = %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("List() after adding length = %d, want 3", len(sessions))
	}

	// Verify all expected sessions are present
	for _, session := range sessions {
		if !expectedIDs[session.ID] {
			t.Errorf("List() unexpected session ID = %v", session.ID)
		}
		delete(expectedIDs, session.ID)
	}
	if len(expectedIDs) > 0 {
		t.Errorf("List() missing sessions: %v", expectedIDs)
	}
}

func TestHelpers_ListWithExpiredSessions(t *testing.T) {
	store, err := kvs.NewMemoryStore("test-list-expired", kvs.MemoryConfig{})
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.Close()

	now := time.Now()

	// Add 2 valid sessions
	for i := 0; i < 2; i++ {
		session := &Session{
			ID:            "valid-" + string(rune('A'+i)),
			Email:         "user@example.com",
			Provider:      "google",
			CreatedAt:     now,
			ExpiresAt:     now.Add(1 * time.Hour),
			Authenticated: true,
		}
		err = Set(store, session.ID, session)
		if err != nil {
			t.Fatalf("Set() valid session %d error = %v", i, err)
		}
	}

	// Add 1 session that expires soon
	expiringSoon := &Session{
		ID:            "expiring-soon",
		Email:         "expiring@example.com",
		Provider:      "google",
		CreatedAt:     now,
		ExpiresAt:     now.Add(100 * time.Millisecond),
		Authenticated: true,
	}
	err = Set(store, expiringSoon.ID, expiringSoon)
	if err != nil {
		t.Fatalf("Set() expiring session error = %v", err)
	}

	// Wait for one to expire
	time.Sleep(200 * time.Millisecond)

	// List should only return valid sessions (expired ones are skipped)
	sessions, err := List(store)
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	// Should only have 2 valid sessions (the expired one is skipped)
	if len(sessions) != 2 {
		t.Errorf("List() with expired length = %d, want 2", len(sessions))
	}

	// Verify none of the returned sessions are expired
	for _, session := range sessions {
		if !session.IsValid() {
			t.Errorf("List() returned expired session: %v", session.ID)
		}
	}
}
