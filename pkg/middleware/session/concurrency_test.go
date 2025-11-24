package session

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSession_ConcurrentWrites tests concurrent writes to different sessions
func TestSession_ConcurrentWrites(t *testing.T) {
	store, err := kvs.NewMemoryStore("test-sessions", kvs.MemoryConfig{
		CleanupInterval: 1 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	numGoroutines := 100
	now := time.Now()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Multiple goroutines writing different sessions
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			session := &Session{
				ID:            fmt.Sprintf("session-%d", idx),
				Email:         fmt.Sprintf("user-%d@example.com", idx),
				Name:          fmt.Sprintf("User %d", idx),
				Provider:      "test",
				CreatedAt:     now,
				ExpiresAt:     now.Add(1 * time.Hour),
				Authenticated: true,
			}
			err := Set(store, session.ID, session)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all sessions were written
	for i := 0; i < numGoroutines; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		session, err := Get(store, sessionID)
		require.NoError(t, err)
		assert.Equal(t, sessionID, session.ID)
	}
}

// TestSession_ConcurrentReadWrite tests concurrent reads and writes
func TestSession_ConcurrentReadWrite(t *testing.T) {
	store, err := kvs.NewMemoryStore("test-sessions", kvs.MemoryConfig{
		CleanupInterval: 1 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	now := time.Now()
	numReaders := 50
	numWriters := 50

	// Create initial sessions
	for i := 0; i < 10; i++ {
		session := &Session{
			ID:            fmt.Sprintf("session-%d", i),
			Email:         fmt.Sprintf("user-%d@example.com", i),
			Name:          fmt.Sprintf("User %d", i),
			Provider:      "test",
			CreatedAt:     now,
			ExpiresAt:     now.Add(1 * time.Hour),
			Authenticated: true,
		}
		err := Set(store, session.ID, session)
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	wg.Add(numReaders + numWriters)

	// Readers
	for i := 0; i < numReaders; i++ {
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				sessionID := fmt.Sprintf("session-%d", j%10)
				_, _ = Get(store, sessionID)
			}
		}(i)
	}

	// Writers
	for i := 0; i < numWriters; i++ {
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				sessionID := fmt.Sprintf("session-%d", j%10)
				session := &Session{
					ID:            sessionID,
					Email:         fmt.Sprintf("updated-user-%d@example.com", writerID),
					Name:          fmt.Sprintf("Updated User %d", writerID),
					Provider:      "test",
					CreatedAt:     now,
					ExpiresAt:     now.Add(1 * time.Hour),
					Authenticated: true,
				}
				_ = Set(store, sessionID, session)
			}
		}(i)
	}

	wg.Wait()

	// Verify all sessions are still readable
	for i := 0; i < 10; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		session, err := Get(store, sessionID)
		assert.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, sessionID, session.ID)
	}
}

// TestSession_ConcurrentDelete tests concurrent session deletion
func TestSession_ConcurrentDelete(t *testing.T) {
	store, err := kvs.NewMemoryStore("test-sessions", kvs.MemoryConfig{
		CleanupInterval: 1 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	now := time.Now()
	numGoroutines := 100

	// Create sessions
	for i := 0; i < numGoroutines; i++ {
		session := &Session{
			ID:            fmt.Sprintf("session-%d", i),
			Email:         fmt.Sprintf("user-%d@example.com", i),
			Name:          fmt.Sprintf("User %d", i),
			Provider:      "test",
			CreatedAt:     now,
			ExpiresAt:     now.Add(1 * time.Hour),
			Authenticated: true,
		}
		err := Set(store, session.ID, session)
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Delete sessions concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			sessionID := fmt.Sprintf("session-%d", idx)
			err := Delete(store, sessionID)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all sessions are deleted
	for i := 0; i < numGoroutines; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		session, err := Get(store, sessionID)
		assert.ErrorIs(t, err, ErrSessionNotFound)
		assert.Nil(t, session)
	}
}

// TestSession_ConcurrentUpdateSameSession tests multiple goroutines updating the same session
func TestSession_ConcurrentUpdateSameSession(t *testing.T) {
	store, err := kvs.NewMemoryStore("test-sessions", kvs.MemoryConfig{
		CleanupInterval: 1 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	now := time.Now()
	sessionID := "shared-session"

	// Create initial session
	session := &Session{
		ID:            sessionID,
		Email:         "user@example.com",
		Name:          "User",
		Provider:      "test",
		CreatedAt:     now,
		ExpiresAt:     now.Add(1 * time.Hour),
		Authenticated: true,
	}
	err = Set(store, sessionID, session)
	require.NoError(t, err)

	numGoroutines := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Multiple goroutines updating the same session
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			updatedSession := &Session{
				ID:            sessionID,
				Email:         fmt.Sprintf("user-%d@example.com", idx),
				Name:          fmt.Sprintf("User %d", idx),
				Provider:      "test",
				CreatedAt:     now,
				ExpiresAt:     now.Add(1 * time.Hour),
				Authenticated: true,
			}
			err := Set(store, sessionID, updatedSession)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Session should still be readable and valid
	session, err = Get(store, sessionID)
	require.NoError(t, err)
	assert.Equal(t, sessionID, session.ID)
	assert.True(t, session.IsValid())
}

// TestSession_RaceDetection is designed to be run with -race flag
func TestSession_RaceDetection(t *testing.T) {
	store, err := kvs.NewMemoryStore("test-sessions", kvs.MemoryConfig{
		CleanupInterval: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	now := time.Now()
	sessionID := "race-session"

	// Create initial session
	session := &Session{
		ID:            sessionID,
		Email:         "user@example.com",
		Name:          "User",
		Provider:      "test",
		CreatedAt:     now,
		ExpiresAt:     now.Add(1 * time.Hour),
		Authenticated: true,
	}
	err = Set(store, sessionID, session)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(3)

	// Concurrent operations to trigger race detector
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			updatedSession := &Session{
				ID:            sessionID,
				Email:         fmt.Sprintf("user-%d@example.com", i),
				Name:          fmt.Sprintf("User %d", i),
				Provider:      "test",
				CreatedAt:     now,
				ExpiresAt:     now.Add(1 * time.Hour),
				Authenticated: true,
			}
			_ = Set(store, sessionID, updatedSession)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			_, _ = Get(store, sessionID)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			session, err := Get(store, sessionID)
			if err == nil {
				_ = session.IsValid()
			}
		}
	}()

	wg.Wait()
}
