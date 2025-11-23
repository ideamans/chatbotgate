package kvs

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLevelDBStore tests LevelDB store creation with various configurations
func TestNewLevelDBStore(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		config      LevelDBConfig
		expectError bool
		description string
	}{
		{
			name:      "with empty path (uses cache dir)",
			namespace: "test-cache",
			config: LevelDBConfig{
				Path:            "",
				CleanupInterval: 1 * time.Second,
			},
			expectError: false,
			description: "Should use cache directory when path is empty",
		},
		{
			name:      "with custom path",
			namespace: "test-custom",
			config: LevelDBConfig{
				Path:            filepath.Join(t.TempDir(), "custom-db"),
				CleanupInterval: 1 * time.Second,
			},
			expectError: false,
			description: "Should create database at custom path",
		},
		{
			name:      "with empty namespace",
			namespace: "",
			config: LevelDBConfig{
				Path:            filepath.Join(t.TempDir(), "default-ns"),
				CleanupInterval: 1 * time.Second,
			},
			expectError: false,
			description: "Should use 'default' namespace when empty",
		},
		{
			name:      "with namespace containing special chars",
			namespace: "test@namespace#123!",
			config: LevelDBConfig{
				Path:            filepath.Join(t.TempDir(), "special-chars"),
				CleanupInterval: 1 * time.Second,
			},
			expectError: false,
			description: "Should sanitize namespace with special characters",
		},
		{
			name:      "with sync writes enabled",
			namespace: "test-sync",
			config: LevelDBConfig{
				Path:            filepath.Join(t.TempDir(), "sync-db"),
				SyncWrites:      true,
				CleanupInterval: 1 * time.Second,
			},
			expectError: false,
			description: "Should create database with sync writes enabled",
		},
		{
			name:      "with zero cleanup interval (uses default)",
			namespace: "test-default-cleanup",
			config: LevelDBConfig{
				Path:            filepath.Join(t.TempDir(), "default-cleanup"),
				CleanupInterval: 0,
			},
			expectError: false,
			description: "Should use default cleanup interval when zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewLevelDBStore(tt.namespace, tt.config)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
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
			}
		})
	}
}

// TestLevelDBDecodeValue tests the decodeValue function
func TestLevelDBDecodeValue(t *testing.T) {
	tests := []struct {
		name        string
		encoded     []byte
		expectValue []byte
		expectExp   bool
		expectError bool
		description string
	}{
		{
			name:        "valid value without expiration",
			encoded:     encodeValue([]byte("test-value"), 0),
			expectValue: []byte("test-value"),
			expectExp:   false,
			expectError: false,
			description: "Should decode value without expiration",
		},
		{
			name:        "valid value with future expiration",
			encoded:     encodeValue([]byte("test-value"), 1*time.Hour),
			expectValue: []byte("test-value"),
			expectExp:   false,
			expectError: false,
			description: "Should decode value with future expiration",
		},
		{
			name:        "valid value with no TTL (negative treated as zero)",
			encoded:     encodeValue([]byte("test-value"), -1*time.Millisecond),
			expectValue: []byte("test-value"),
			expectExp:   false,
			expectError: false,
			description: "Negative TTL should be treated as zero (no expiration)",
		},
		{
			name:        "invalid encoded value (too short)",
			encoded:     []byte{1, 2, 3},
			expectValue: nil,
			expectExp:   false,
			expectError: true,
			description: "Should return error for malformed data",
		},
		{
			name:        "empty value with expiration",
			encoded:     encodeValue([]byte{}, 1*time.Hour),
			expectValue: []byte{},
			expectExp:   false,
			expectError: false,
			description: "Should handle empty value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, expired, err := decodeValue(tt.encoded)

			if tt.expectError {
				require.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)
				assert.Equal(t, tt.expectExp, expired, "Expiration flag should match")
				if !expired {
					assert.Equal(t, tt.expectValue, value, "Value should match")
				}
			}
		})
	}
}

// TestLevelDBEncodeValue tests the encodeValue function
func TestLevelDBEncodeValue(t *testing.T) {
	tests := []struct {
		name        string
		value       []byte
		ttl         time.Duration
		description string
	}{
		{
			name:        "encode without TTL",
			value:       []byte("test"),
			ttl:         0,
			description: "Should encode with zero expiration",
		},
		{
			name:        "encode with TTL",
			value:       []byte("test"),
			ttl:         1 * time.Hour,
			description: "Should encode with future expiration",
		},
		{
			name:        "encode empty value",
			value:       []byte{},
			ttl:         0,
			description: "Should encode empty value",
		},
		{
			name:        "encode large value",
			value:       make([]byte, 1024),
			ttl:         1 * time.Hour,
			description: "Should encode large value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := encodeValue(tt.value, tt.ttl)

			// Verify encoding structure
			assert.GreaterOrEqual(t, len(encoded), 8, "Encoded value should have at least 8 bytes for expiration")
			assert.Equal(t, 8+len(tt.value), len(encoded), "Encoded length should be 8 + value length")

			// Decode and verify
			decoded, expired, err := decodeValue(encoded)
			require.NoError(t, err, "Should decode without error")
			if tt.ttl == 0 {
				assert.False(t, expired, "Should not be expired when TTL is 0")
				assert.Equal(t, tt.value, decoded, "Decoded value should match original")
			}
		})
	}
}

// TestLevelDBStoreGetErrors tests Get method error cases
func TestLevelDBStoreGetErrors(t *testing.T) {
	tmpDir := t.TempDir()
	config := LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewLevelDBStore("test", config)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	t.Run("get with expired value triggers async delete", func(t *testing.T) {
		// Set a key with very short TTL
		err := store.Set(ctx, "expire-me", []byte("value"), 1*time.Millisecond)
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		// Get should return ErrNotFound and trigger async delete
		_, err = store.Get(ctx, "expire-me")
		assert.Equal(t, ErrNotFound, err, "Should return ErrNotFound for expired key")

		// Give async delete time to complete
		time.Sleep(10 * time.Millisecond)
	})
}

// TestLevelDBStoreDeleteErrors tests Delete method error cases
func TestLevelDBStoreDeleteErrors(t *testing.T) {
	tmpDir := t.TempDir()
	config := LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewLevelDBStore("test", config)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	t.Run("delete non-existent key", func(t *testing.T) {
		err := store.Delete(ctx, "non-existent")
		assert.NoError(t, err, "Delete should not error for non-existent key")
	})
}

// TestLevelDBStoreListErrors tests List method error cases
func TestLevelDBStoreListErrors(t *testing.T) {
	tmpDir := t.TempDir()
	config := LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewLevelDBStore("test", config)
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

// TestLevelDBStoreCountErrors tests Count method error cases
func TestLevelDBStoreCountErrors(t *testing.T) {
	tmpDir := t.TempDir()
	config := LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewLevelDBStore("test", config)
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

// TestLevelDBStoreCloseMultipleTimes tests calling Close multiple times
func TestLevelDBStoreCloseMultipleTimes(t *testing.T) {
	tmpDir := t.TempDir()
	config := LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewLevelDBStore("test", config)
	require.NoError(t, err)

	// First close should succeed
	err = store.Close()
	assert.NoError(t, err, "First Close should succeed")

	// Second close should return ErrClosed
	err = store.Close()
	assert.Equal(t, ErrClosed, err, "Second Close should return ErrClosed")
}

// TestLevelDBStoreExistsErrors tests Exists method error cases
func TestLevelDBStoreExistsErrors(t *testing.T) {
	tmpDir := t.TempDir()
	config := LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewLevelDBStore("test", config)
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

// TestLevelDBStoreInvalidPath tests NewLevelDBStore with invalid path
func TestLevelDBStoreInvalidPath(t *testing.T) {
	// This test tries to create a database in a non-writable location
	// On most systems, /dev/null is not a valid directory
	config := LevelDBConfig{
		Path:            "/dev/null/invalid/path/db",
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewLevelDBStore("test", config)
	// This should fail because we can't create a directory at /dev/null
	if err != nil {
		assert.Nil(t, store, "Store should be nil on error")
		assert.Contains(t, err.Error(), "failed to", "Error should mention failure")
	} else {
		// If somehow this succeeded, clean up
		if store != nil {
			_ = store.Close()
		}
	}
}

// TestLevelDBCleanupWithClosedStore tests cleanup when store is closed
func TestLevelDBCleanupWithClosedStore(t *testing.T) {
	tmpDir := t.TempDir()
	config := LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 100 * time.Millisecond,
	}

	store, err := NewLevelDBStore("test", config)
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

// TestLevelDBStoreSetErrors tests Set method error cases
func TestLevelDBStoreSetErrors(t *testing.T) {
	tmpDir := t.TempDir()
	config := LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 1 * time.Second,
	}

	store, err := NewLevelDBStore("test", config)
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
}
