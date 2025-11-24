package kvs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMemoryStore tests MemoryStore creation with various configurations
func TestNewMemoryStore(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		config      MemoryConfig
		description string
	}{
		{
			name:      "with custom cleanup interval",
			namespace: "test",
			config: MemoryConfig{
				CleanupInterval: 100 * time.Millisecond,
			},
			description: "Should create store with custom cleanup interval",
		},
		{
			name:      "with zero cleanup interval (uses default)",
			namespace: "test",
			config: MemoryConfig{
				CleanupInterval: 0,
			},
			description: "Should use default cleanup interval",
		},
		{
			name:      "with empty namespace",
			namespace: "",
			config: MemoryConfig{
				CleanupInterval: 1 * time.Second,
			},
			description: "Should create store with empty namespace",
		},
		{
			name:      "with special characters in namespace",
			namespace: "test@namespace#123",
			config: MemoryConfig{
				CleanupInterval: 1 * time.Second,
			},
			description: "Should handle special characters in namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewMemoryStore(tt.namespace, tt.config)
			require.NoError(t, err, tt.description)
			require.NotNil(t, store, "Store should not be nil")
			assert.Equal(t, tt.namespace, store.namespace, "Namespace should match")
			defer func() { _ = store.Close() }()

			// Verify cleanup interval
			if tt.config.CleanupInterval == 0 {
				assert.Equal(t, 5*time.Minute, store.cleanupInterval, "Should use default 5 minute cleanup interval")
			} else {
				assert.Equal(t, tt.config.CleanupInterval, store.cleanupInterval, "Cleanup interval should match config")
			}
		})
	}
}

// TestMemoryStoreCloseErrors tests Close method edge cases
func TestMemoryStoreCloseErrors(t *testing.T) {
	config := MemoryConfig{
		CleanupInterval: 1 * time.Second,
	}

	t.Run("close multiple times", func(t *testing.T) {
		store, err := NewMemoryStore("test", config)
		require.NoError(t, err)

		// First close should succeed
		err = store.Close()
		assert.NoError(t, err, "First Close should succeed")

		// Second close should return ErrClosed
		err = store.Close()
		assert.Equal(t, ErrClosed, err, "Second Close should return ErrClosed")
	})

	t.Run("cleanup loop stops after close", func(t *testing.T) {
		store, err := NewMemoryStore("test", MemoryConfig{
			CleanupInterval: 100 * time.Millisecond,
		})
		require.NoError(t, err)

		ctx := context.Background()

		// Set a key with short TTL
		err = store.Set(ctx, "cleanup-test", []byte("value"), 1*time.Millisecond)
		require.NoError(t, err)

		// Close the store
		err = store.Close()
		require.NoError(t, err)

		// Wait to ensure cleanup would have run if store wasn't closed
		time.Sleep(200 * time.Millisecond)

		// Cleanup should have stopped and not panic
	})
}

// TestMemoryStoreCleanupWithClosedStore tests cleanup when store is closed
func TestMemoryStoreCleanupWithClosedStore(t *testing.T) {
	config := MemoryConfig{
		CleanupInterval: 100 * time.Millisecond,
	}

	store, err := NewMemoryStore("test", config)
	require.NoError(t, err)

	ctx := context.Background()

	// Set a key with short TTL
	err = store.Set(ctx, "cleanup-test", []byte("value"), 1*time.Millisecond)
	require.NoError(t, err)

	// Close the store before cleanup runs
	err = store.Close()
	require.NoError(t, err)

	// Wait to ensure cleanup would have run if store wasn't closed
	time.Sleep(200 * time.Millisecond)

	// Try to call cleanup directly on closed store (should return early)
	store.cleanup()
}

// TestMemoryStoreGetErrors tests Get method error cases
func TestMemoryStoreGetErrors(t *testing.T) {
	config := MemoryConfig{
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewMemoryStore("test", config)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	t.Run("get with expired value", func(t *testing.T) {
		// Set a key with very short TTL
		err := store.Set(ctx, "expire-me", []byte("value"), 1*time.Millisecond)
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		// Get should return ErrNotFound
		_, err = store.Get(ctx, "expire-me")
		assert.Equal(t, ErrNotFound, err, "Should return ErrNotFound for expired key")
	})

	t.Run("get non-existent key", func(t *testing.T) {
		_, err := store.Get(ctx, "non-existent")
		assert.Equal(t, ErrNotFound, err, "Should return ErrNotFound")
	})
}

// TestMemoryStoreSetErrors tests Set method error cases
func TestMemoryStoreSetErrors(t *testing.T) {
	config := MemoryConfig{
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewMemoryStore("test", config)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	t.Run("set empty value", func(t *testing.T) {
		err := store.Set(ctx, "empty-key", []byte{}, 0)
		assert.NoError(t, err, "Should allow setting empty value")

		val, err := store.Get(ctx, "empty-key")
		require.NoError(t, err)
		assert.Equal(t, []byte{}, val, "Should retrieve empty value")
	})

	t.Run("set with negative TTL", func(t *testing.T) {
		err := store.Set(ctx, "negative-ttl", []byte("value"), -1*time.Hour)
		assert.NoError(t, err, "Should allow negative TTL (treated as 0)")

		val, err := store.Get(ctx, "negative-ttl")
		require.NoError(t, err)
		assert.Equal(t, []byte("value"), val, "Should retrieve value")
	})

	t.Run("overwrite existing key", func(t *testing.T) {
		err := store.Set(ctx, "overwrite", []byte("value1"), 0)
		require.NoError(t, err)

		err = store.Set(ctx, "overwrite", []byte("value2"), 0)
		require.NoError(t, err)

		val, err := store.Get(ctx, "overwrite")
		require.NoError(t, err)
		assert.Equal(t, []byte("value2"), val, "Should get new value")
	})
}

// TestMemoryStoreDeleteErrors tests Delete method error cases
func TestMemoryStoreDeleteErrors(t *testing.T) {
	config := MemoryConfig{
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewMemoryStore("test", config)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	t.Run("delete non-existent key", func(t *testing.T) {
		err := store.Delete(ctx, "non-existent")
		assert.NoError(t, err, "Delete should not error for non-existent key")
	})
}

// TestMemoryStoreExistsErrors tests Exists method error cases
func TestMemoryStoreExistsErrors(t *testing.T) {
	config := MemoryConfig{
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewMemoryStore("test", config)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	t.Run("exists with expired value", func(t *testing.T) {
		// Set a key with very short TTL
		err := store.Set(ctx, "exists-expired", []byte("value"), 1*time.Millisecond)
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		// Exists should return false for expired key
		exists, err := store.Exists(ctx, "exists-expired")
		require.NoError(t, err)
		assert.False(t, exists, "Should return false for expired key")
	})
}

// TestMemoryStoreListErrors tests List method error cases
func TestMemoryStoreListErrors(t *testing.T) {
	config := MemoryConfig{
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewMemoryStore("test", config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("list with expired values", func(t *testing.T) {
		// Set keys with different expiration
		_ = store.Set(ctx, "list-valid", []byte("value"), 0)
		_ = store.Set(ctx, "list-expired", []byte("value"), 1*time.Millisecond)

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		// List should only return non-expired keys
		keys, err := store.List(ctx, "list-")
		require.NoError(t, err)
		assert.Contains(t, keys, "list-valid", "Should contain valid key")
		assert.NotContains(t, keys, "list-expired", "Should not contain expired key")

		// Cleanup
		_ = store.Delete(ctx, "list-valid")
	})

	_ = store.Close()
}

// TestMemoryStoreCountErrors tests Count method error cases
func TestMemoryStoreCountErrors(t *testing.T) {
	config := MemoryConfig{
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewMemoryStore("test", config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("count with expired values", func(t *testing.T) {
		// Set keys with different expiration
		_ = store.Set(ctx, "count-valid", []byte("value"), 0)
		_ = store.Set(ctx, "count-expired", []byte("value"), 1*time.Millisecond)

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		// Count should only count non-expired keys
		count, err := store.Count(ctx, "count-")
		require.NoError(t, err)
		assert.Equal(t, 1, count, "Should only count valid key")

		// Cleanup
		_ = store.Delete(ctx, "count-valid")
	})

	_ = store.Close()
}

// TestMemoryStoreCleanupExpiredKeys tests the cleanup process
func TestMemoryStoreCleanupExpiredKeys(t *testing.T) {
	config := MemoryConfig{
		CleanupInterval: 50 * time.Millisecond,
	}

	store, err := NewMemoryStore("test", config)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// Set keys with very short TTL
	for i := 0; i < 10; i++ {
		key := "cleanup-key-" + string(rune('0'+i))
		err := store.Set(ctx, key, []byte("value"), 1*time.Millisecond)
		require.NoError(t, err)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Wait for cleanup to run
	time.Sleep(100 * time.Millisecond)

	// All keys should be gone
	count, err := store.Count(ctx, "cleanup-key-")
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Cleanup should have removed all expired keys")
}

// TestMemoryStoreConcurrentAccess tests concurrent operations
func TestMemoryStoreConcurrentAccess(t *testing.T) {
	config := MemoryConfig{
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewMemoryStore("test", config)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	numGoroutines := 10
	numOpsPerGoroutine := 20

	done := make(chan bool, numGoroutines)

	// Run concurrent operations
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			for i := 0; i < numOpsPerGoroutine; i++ {
				key := "concurrent-" + string(rune('a'+goroutineID%26)) + "-" + string(rune('0'+i%10))

				// Set
				_ = store.Set(ctx, key, []byte("value"), 0)

				// Get
				_, _ = store.Get(ctx, key)

				// Exists
				_, _ = store.Exists(ctx, key)

				// Delete
				_ = store.Delete(ctx, key)
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
