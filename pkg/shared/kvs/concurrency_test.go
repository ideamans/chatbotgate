package kvs

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryStore_ConcurrentWrites tests concurrent writes to the same key
func TestMemoryStore_ConcurrentWrites(t *testing.T) {
	store, err := NewMemoryStore("test", MemoryConfig{
		CleanupInterval: 1 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	numGoroutines := 100
	key := "concurrent-key"

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Multiple goroutines writing to the same key
	for i := 0; i < numGoroutines; i++ {
		go func(value int) {
			defer wg.Done()
			err := store.Set(ctx, key, []byte(fmt.Sprintf("value-%d", value)), 0)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Should be able to read the key (value may be any of the concurrent writes)
	val, err := store.Get(ctx, key)
	require.NoError(t, err)
	assert.NotNil(t, val)
}

// TestMemoryStore_ConcurrentReadWrite tests concurrent reads and writes
func TestMemoryStore_ConcurrentReadWrite(t *testing.T) {
	store, err := NewMemoryStore("test", MemoryConfig{
		CleanupInterval: 1 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	numReaders := 50
	numWriters := 50

	// Initialize some keys
	for i := 0; i < 10; i++ {
		err := store.Set(ctx, fmt.Sprintf("key-%d", i), []byte(fmt.Sprintf("initial-%d", i)), 0)
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	wg.Add(numReaders + numWriters)

	// Readers
	for i := 0; i < numReaders; i++ {
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key-%d", j%10)
				_, _ = store.Get(ctx, key)
			}
		}(i)
	}

	// Writers
	for i := 0; i < numWriters; i++ {
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key-%d", j%10)
				value := fmt.Sprintf("writer-%d-iteration-%d", writerID, j)
				_ = store.Set(ctx, key, []byte(value), 0)
			}
		}(i)
	}

	wg.Wait()

	// Verify store is still functional
	err = store.Set(ctx, "final-key", []byte("final-value"), 0)
	assert.NoError(t, err)
}

// TestMemoryStore_ConcurrentListAndModify tests concurrent list and modify operations
func TestMemoryStore_ConcurrentListAndModify(t *testing.T) {
	store, err := NewMemoryStore("test", MemoryConfig{
		CleanupInterval: 1 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	prefix := "list-test-"

	var wg sync.WaitGroup
	wg.Add(3)

	// Goroutine 1: Add keys
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("%s%d", prefix, i)
			_ = store.Set(ctx, key, []byte(fmt.Sprintf("value-%d", i)), 0)
		}
	}()

	// Goroutine 2: List keys
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_, _ = store.List(ctx, prefix)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// Goroutine 3: Delete keys
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Let some keys be created first
		for i := 0; i < 50; i++ {
			key := fmt.Sprintf("%s%d", prefix, i)
			_ = store.Delete(ctx, key)
		}
	}()

	wg.Wait()

	// Store should still be functional
	count, err := store.Count(ctx, prefix)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, 0)
}

// TestLevelDBStore_ConcurrentWrites tests concurrent writes to LevelDB
func TestLevelDBStore_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewLevelDBStore("test", LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 1 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	numGoroutines := 100
	key := "concurrent-key"

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Multiple goroutines writing to the same key
	for i := 0; i < numGoroutines; i++ {
		go func(value int) {
			defer wg.Done()
			err := store.Set(ctx, key, []byte(fmt.Sprintf("value-%d", value)), 0)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Should be able to read the key
	val, err := store.Get(ctx, key)
	require.NoError(t, err)
	assert.NotNil(t, val)
}

// TestLevelDBStore_ConcurrentReadWrite tests concurrent reads and writes to LevelDB
func TestLevelDBStore_ConcurrentReadWrite(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewLevelDBStore("test", LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 1 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	numReaders := 50
	numWriters := 50

	// Initialize some keys
	for i := 0; i < 10; i++ {
		err := store.Set(ctx, fmt.Sprintf("key-%d", i), []byte(fmt.Sprintf("initial-%d", i)), 0)
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	wg.Add(numReaders + numWriters)

	// Readers
	for i := 0; i < numReaders; i++ {
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key-%d", j%10)
				_, _ = store.Get(ctx, key)
			}
		}(i)
	}

	// Writers
	for i := 0; i < numWriters; i++ {
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key-%d", j%10)
				value := fmt.Sprintf("writer-%d-iteration-%d", writerID, j)
				_ = store.Set(ctx, key, []byte(value), 0)
			}
		}(i)
	}

	wg.Wait()

	// Verify store is still functional
	err = store.Set(ctx, "final-key", []byte("final-value"), 0)
	assert.NoError(t, err)
}

// TestMemoryStore_ConcurrentExpiration tests concurrent operations during key expiration
func TestMemoryStore_ConcurrentExpiration(t *testing.T) {
	store, err := NewMemoryStore("test", MemoryConfig{
		CleanupInterval: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	numGoroutines := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Multiple goroutines setting keys with short TTL
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				key := fmt.Sprintf("expiring-key-%d-%d", id, j)
				// Set with very short TTL
				_ = store.Set(ctx, key, []byte(fmt.Sprintf("value-%d", j)), 10*time.Millisecond)
				// Immediately try to read
				_, _ = store.Get(ctx, key)
			}
		}(i)
	}

	wg.Wait()

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Store should still be functional
	err = store.Set(ctx, "test-key", []byte("test-value"), 0)
	assert.NoError(t, err)
}

// TestLevelDBStore_ConcurrentCleanup tests concurrent operations during cleanup
func TestLevelDBStore_ConcurrentCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewLevelDBStore("test", LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	numGoroutines := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Multiple goroutines setting keys with short TTL
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				key := fmt.Sprintf("expiring-key-%d-%d", id, j)
				// Set with very short TTL
				_ = store.Set(ctx, key, []byte(fmt.Sprintf("value-%d", j)), 10*time.Millisecond)
				// Immediately try to read
				_, _ = store.Get(ctx, key)
			}
		}(i)
	}

	wg.Wait()

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Store should still be functional
	err = store.Set(ctx, "test-key", []byte("test-value"), 0)
	assert.NoError(t, err)
}

// TestMemoryStore_RaceDetection is designed to be run with -race flag
func TestMemoryStore_RaceDetection(t *testing.T) {
	store, err := NewMemoryStore("test", MemoryConfig{
		CleanupInterval: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(3)

	// Concurrent operations to trigger race detector
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_ = store.Set(ctx, "race-key", []byte(fmt.Sprintf("value-%d", i)), 0)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_, _ = store.Get(ctx, "race-key")
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_, _ = store.Exists(ctx, "race-key")
		}
	}()

	wg.Wait()
}
