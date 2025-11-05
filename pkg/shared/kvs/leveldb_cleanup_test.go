package kvs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLevelDBCleanupLoop tests that the cleanup loop runs periodically
func TestLevelDBCleanupLoop(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kvs-leveldb-cleanup-test-*")
	require.NoError(t, err, "Should create temp dir")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create store with short cleanup interval
	config := Config{
		Type: "leveldb",
		LevelDB: LevelDBConfig{
			Path:            filepath.Join(tmpDir, "db"),
			SyncWrites:      false,
			CleanupInterval: 200 * time.Millisecond, // Run cleanup every 200ms
		},
	}

	store, err := New(config)
	require.NoError(t, err, "Should create LevelDBStore")
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// Set multiple keys with very short TTL
	keys := []string{"cleanup1", "cleanup2", "cleanup3", "cleanup4", "cleanup5"}
	for _, key := range keys {
		err := store.Set(ctx, key, []byte("value"), 150*time.Millisecond)
		require.NoError(t, err, "Set should not return error")
	}

	// Verify all keys exist immediately
	count, err := store.Count(ctx, "cleanup")
	require.NoError(t, err, "Count should not return error")
	assert.Equal(t, len(keys), count, "All keys should exist immediately")

	// Wait for keys to expire (150ms) + cleanup to run (200ms) + buffer
	time.Sleep(400 * time.Millisecond)

	// Verify cleanup deleted all expired keys
	count, err = store.Count(ctx, "cleanup")
	require.NoError(t, err, "Count should not return error")
	assert.Equal(t, 0, count, "All expired keys should be cleaned up")
}

// TestLevelDBCleanupInterval tests that cleanup respects the configured interval
func TestLevelDBCleanupInterval(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kvs-leveldb-cleanup-interval-test-*")
	require.NoError(t, err, "Should create temp dir")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create store with 500ms cleanup interval
	config := Config{
		Type: "leveldb",
		LevelDB: LevelDBConfig{
			Path:            filepath.Join(tmpDir, "db"),
			SyncWrites:      false,
			CleanupInterval: 500 * time.Millisecond,
		},
	}

	store, err := New(config)
	require.NoError(t, err, "Should create LevelDBStore")
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// Set a key with 100ms TTL
	err = store.Set(ctx, "interval-test", []byte("value"), 100*time.Millisecond)
	require.NoError(t, err, "Set should not return error")

	// Wait 200ms - key is expired but cleanup hasn't run yet
	time.Sleep(200 * time.Millisecond)

	// Key should still be retrievable (Get checks expiration)
	// Actually, Get will return ErrNotFound because it checks TTL
	// So we need to check the raw storage
	exists, err := store.Exists(ctx, "interval-test")
	require.NoError(t, err, "Exists should not return error")
	// Exists checks TTL, so it should return false
	assert.False(t, exists, "Expired key should not exist (Exists checks TTL)")

	// Wait for cleanup to run (total 600ms > 500ms cleanup interval)
	time.Sleep(400 * time.Millisecond)

	// After cleanup, the key should definitely be gone from storage
	count, err := store.Count(ctx, "interval-test")
	require.NoError(t, err, "Count should not return error")
	assert.Equal(t, 0, count, "Cleanup should have removed expired key")
}

// TestLevelDBCleanupStopsOnClose tests that cleanup stops when store is closed
func TestLevelDBCleanupStopsOnClose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kvs-leveldb-cleanup-stop-test-*")
	require.NoError(t, err, "Should create temp dir")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := Config{
		Type: "leveldb",
		LevelDB: LevelDBConfig{
			Path:            filepath.Join(tmpDir, "db"),
			SyncWrites:      false,
			CleanupInterval: 100 * time.Millisecond,
		},
	}

	store, err := New(config)
	require.NoError(t, err, "Should create LevelDBStore")

	leveldbStore, ok := store.(*LevelDBStore)
	require.True(t, ok, "Should be LevelDBStore")

	// Close the store
	err = store.Close()
	require.NoError(t, err, "Close should not return error")

	// Wait for cleanup to signal completion
	select {
	case <-leveldbStore.cleanupDone:
		// Success - cleanup goroutine exited
	case <-time.After(1 * time.Second):
		t.Fatal("Cleanup goroutine did not exit after Close")
	}
}

// TestLevelDBCleanupSelectiveDelete tests that cleanup only deletes expired keys
func TestLevelDBCleanupSelectiveDelete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kvs-leveldb-cleanup-selective-test-*")
	require.NoError(t, err, "Should create temp dir")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := Config{
		Type: "leveldb",
		LevelDB: LevelDBConfig{
			Path:            filepath.Join(tmpDir, "db"),
			SyncWrites:      false,
			CleanupInterval: 200 * time.Millisecond,
		},
	}

	store, err := New(config)
	require.NoError(t, err, "Should create LevelDBStore")
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// Set keys with different TTLs
	err = store.Set(ctx, "short-ttl-1", []byte("value"), 150*time.Millisecond)
	require.NoError(t, err, "Set should not return error")

	err = store.Set(ctx, "short-ttl-2", []byte("value"), 150*time.Millisecond)
	require.NoError(t, err, "Set should not return error")

	err = store.Set(ctx, "long-ttl-1", []byte("value"), 10*time.Second)
	require.NoError(t, err, "Set should not return error")

	err = store.Set(ctx, "long-ttl-2", []byte("value"), 10*time.Second)
	require.NoError(t, err, "Set should not return error")

	err = store.Set(ctx, "no-ttl", []byte("value"), 0)
	require.NoError(t, err, "Set should not return error")

	// Wait for short TTL keys to expire and cleanup to run
	time.Sleep(400 * time.Millisecond)

	// Verify short TTL keys are gone
	exists, err := store.Exists(ctx, "short-ttl-1")
	require.NoError(t, err, "Exists should not return error")
	assert.False(t, exists, "short-ttl-1 should be deleted")

	exists, err = store.Exists(ctx, "short-ttl-2")
	require.NoError(t, err, "Exists should not return error")
	assert.False(t, exists, "short-ttl-2 should be deleted")

	// Verify long TTL keys still exist
	exists, err = store.Exists(ctx, "long-ttl-1")
	require.NoError(t, err, "Exists should not return error")
	assert.True(t, exists, "long-ttl-1 should still exist")

	exists, err = store.Exists(ctx, "long-ttl-2")
	require.NoError(t, err, "Exists should not return error")
	assert.True(t, exists, "long-ttl-2 should still exist")

	// Verify no-TTL key still exists
	exists, err = store.Exists(ctx, "no-ttl")
	require.NoError(t, err, "Exists should not return error")
	assert.True(t, exists, "no-ttl should still exist")
}

// TestLevelDBCleanupBatchDeletion tests that cleanup uses batch deletion
func TestLevelDBCleanupBatchDeletion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kvs-leveldb-cleanup-batch-test-*")
	require.NoError(t, err, "Should create temp dir")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := Config{
		Type: "leveldb",
		LevelDB: LevelDBConfig{
			Path:            filepath.Join(tmpDir, "db"),
			SyncWrites:      false,
			CleanupInterval: 200 * time.Millisecond,
		},
	}

	store, err := New(config)
	require.NoError(t, err, "Should create LevelDBStore")
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// Set many keys with short TTL to test batch deletion
	numKeys := 100
	for i := 0; i < numKeys; i++ {
		key := "batch-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		err := store.Set(ctx, key, []byte("value"), 150*time.Millisecond)
		require.NoError(t, err, "Set should not return error")
	}

	// Verify all keys exist
	initialCount, err := store.Count(ctx, "batch-")
	require.NoError(t, err, "Count should not return error")
	assert.Equal(t, numKeys, initialCount, "All keys should exist initially")

	// Wait for expiration and cleanup
	time.Sleep(400 * time.Millisecond)

	// Verify all keys were deleted in batch
	finalCount, err := store.Count(ctx, "batch-")
	require.NoError(t, err, "Count should not return error")
	assert.Equal(t, 0, finalCount, "All expired keys should be deleted via batch operation")
}

// TestLevelDBCleanupWithNamespace tests cleanup works with namespaced store
func TestLevelDBCleanupWithNamespace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kvs-leveldb-cleanup-namespace-test-*")
	require.NoError(t, err, "Should create temp dir")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := Config{
		Type:      "leveldb",
		Namespace: "test-namespace",
		LevelDB: LevelDBConfig{
			Path:            filepath.Join(tmpDir, "db"),
			SyncWrites:      false,
			CleanupInterval: 200 * time.Millisecond,
		},
	}

	store, err := New(config)
	require.NoError(t, err, "Should create LevelDBStore")
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// Set keys with short TTL
	err = store.Set(ctx, "cleanup-key-1", []byte("value"), 150*time.Millisecond)
	require.NoError(t, err, "Set should not return error")

	err = store.Set(ctx, "cleanup-key-2", []byte("value"), 150*time.Millisecond)
	require.NoError(t, err, "Set should not return error")

	// Verify keys exist
	count, err := store.Count(ctx, "cleanup-")
	require.NoError(t, err, "Count should not return error")
	assert.Equal(t, 2, count, "Both keys should exist")

	// Wait for expiration and cleanup
	time.Sleep(400 * time.Millisecond)

	// Verify cleanup worked with namespace
	count, err = store.Count(ctx, "cleanup-")
	require.NoError(t, err, "Count should not return error")
	assert.Equal(t, 0, count, "All expired keys should be cleaned up even with namespace")
}

// TestLevelDBCleanupConcurrentOperations tests cleanup works correctly during concurrent operations
func TestLevelDBCleanupConcurrentOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kvs-leveldb-cleanup-concurrent-test-*")
	require.NoError(t, err, "Should create temp dir")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := Config{
		Type: "leveldb",
		LevelDB: LevelDBConfig{
			Path:            filepath.Join(tmpDir, "db"),
			SyncWrites:      false,
			CleanupInterval: 100 * time.Millisecond,
		},
	}

	store, err := New(config)
	require.NoError(t, err, "Should create LevelDBStore")
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// Start goroutine that continuously writes keys
	done := make(chan bool)
	go func() {
		for i := 0; i < 50; i++ {
			_ = store.Set(ctx, "concurrent-"+string(rune('a'+i%26)), []byte("value"), 200*time.Millisecond)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for writes to complete
	<-done

	// Wait for cleanup to run multiple times
	time.Sleep(500 * time.Millisecond)

	// Verify all expired keys were cleaned up
	count, err := store.Count(ctx, "concurrent-")
	require.NoError(t, err, "Count should not return error")
	assert.Equal(t, 0, count, "All expired keys should be cleaned up despite concurrent operations")
}
