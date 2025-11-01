package session

import (
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func setupTestRedis(t *testing.T) (*RedisStore, *miniredis.Miniredis) {
	t.Helper()

	// Create mock Redis server
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	// Create Redis store
	store, err := NewRedisStore(RedisConfig{
		Addr:   mr.Addr(),
		Prefix: "test:",
	})
	if err != nil {
		mr.Close()
		t.Fatalf("Failed to create RedisStore: %v", err)
	}

	return store, mr
}

func TestNewRedisStore(t *testing.T) {
	store, mr := setupTestRedis(t)
	defer mr.Close()
	defer store.Close()

	if store == nil {
		t.Fatal("NewRedisStore() returned nil")
	}

	if store.prefix != "test:" {
		t.Errorf("prefix = %s, want test:", store.prefix)
	}
}

func TestNewRedisStore_DefaultPrefix(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}
	defer mr.Close()

	store, err := NewRedisStore(RedisConfig{
		Addr: mr.Addr(),
	})
	if err != nil {
		t.Fatalf("Failed to create RedisStore: %v", err)
	}
	defer store.Close()

	if store.prefix != "session:" {
		t.Errorf("prefix = %s, want session:", store.prefix)
	}
}

func TestNewRedisStore_ConnectionError(t *testing.T) {
	_, err := NewRedisStore(RedisConfig{
		Addr: "localhost:9999", // Non-existent server
	})
	if err == nil {
		t.Error("NewRedisStore() should return error for invalid address")
	}
}

func TestRedisStore_SetAndGet(t *testing.T) {
	store, mr := setupTestRedis(t)
	defer mr.Close()
	defer store.Close()

	session := &Session{
		ID:            "test-session-id",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		Authenticated: true,
	}

	// Set session
	err := store.Set(session.ID, session)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get session
	got, err := store.Get(session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.ID != session.ID {
		t.Errorf("ID = %s, want %s", got.ID, session.ID)
	}
	if got.Email != session.Email {
		t.Errorf("Email = %s, want %s", got.Email, session.Email)
	}
	if got.Provider != session.Provider {
		t.Errorf("Provider = %s, want %s", got.Provider, session.Provider)
	}
	if !got.Authenticated {
		t.Error("Authenticated should be true")
	}
}

func TestRedisStore_GetNotFound(t *testing.T) {
	store, mr := setupTestRedis(t)
	defer mr.Close()
	defer store.Close()

	_, err := store.Get("non-existent-id")
	if err != ErrRedisSessionNotFound {
		t.Errorf("Get() error = %v, want ErrRedisSessionNotFound", err)
	}
}

func TestRedisStore_Delete(t *testing.T) {
	store, mr := setupTestRedis(t)
	defer mr.Close()
	defer store.Close()

	session := &Session{
		ID:            "test-session-id",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		Authenticated: true,
	}

	// Set session
	store.Set(session.ID, session)

	// Delete session
	err := store.Delete(session.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err = store.Get(session.ID)
	if err != ErrRedisSessionNotFound {
		t.Errorf("Get() after Delete() error = %v, want ErrRedisSessionNotFound", err)
	}
}

func TestRedisStore_ExpiredSession(t *testing.T) {
	store, mr := setupTestRedis(t)
	defer mr.Close()
	defer store.Close()

	session := &Session{
		ID:            "test-session-id",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(1 * time.Second),
		Authenticated: true,
	}

	// Set session
	store.Set(session.ID, session)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Fast-forward time in miniredis
	mr.FastForward(2 * time.Second)

	// Try to get expired session
	_, err := store.Get(session.ID)
	if err != ErrRedisSessionNotFound {
		t.Errorf("Get() error = %v, want ErrRedisSessionNotFound for expired session", err)
	}
}

func TestRedisStore_SetExpiredSession(t *testing.T) {
	store, mr := setupTestRedis(t)
	defer mr.Close()
	defer store.Close()

	session := &Session{
		ID:            "test-session-id",
		Email:         "user@example.com",
		Provider:      "google",
		CreatedAt:     time.Now().Add(-2 * time.Hour),
		ExpiresAt:     time.Now().Add(-1 * time.Hour), // Already expired
		Authenticated: true,
	}

	// Try to set expired session
	err := store.Set(session.ID, session)
	if err == nil {
		t.Error("Set() should return error for expired session")
	}
}

func TestRedisStore_Count(t *testing.T) {
	store, mr := setupTestRedis(t)
	defer mr.Close()
	defer store.Close()

	// Initially no sessions
	count, err := store.Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}

	// Add sessions
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:            fmt.Sprintf("session-%d", i),
			Email:         "user@example.com",
			Provider:      "google",
			CreatedAt:     time.Now(),
			ExpiresAt:     time.Now().Add(1 * time.Hour),
			Authenticated: true,
		}
		store.Set(session.ID, session)
	}

	// Count sessions
	count, err = store.Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}
}

func TestRedisStore_List(t *testing.T) {
	store, mr := setupTestRedis(t)
	defer mr.Close()
	defer store.Close()

	// Add sessions
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:            fmt.Sprintf("session-%d", i),
			Email:         fmt.Sprintf("user%d@example.com", i),
			Provider:      "google",
			CreatedAt:     time.Now(),
			ExpiresAt:     time.Now().Add(1 * time.Hour),
			Authenticated: true,
		}
		store.Set(session.ID, session)
	}

	// List sessions
	sessions, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("List() returned %d sessions, want 3", len(sessions))
	}

	// Verify sessions
	emails := make(map[string]bool)
	for _, s := range sessions {
		emails[s.Email] = true
	}

	expectedEmails := []string{
		"user0@example.com",
		"user1@example.com",
		"user2@example.com",
	}
	for _, email := range expectedEmails {
		if !emails[email] {
			t.Errorf("List() missing email %s", email)
		}
	}
}
