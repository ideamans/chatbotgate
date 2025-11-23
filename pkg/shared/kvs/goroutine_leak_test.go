package kvs

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// TestMemoryStore_NoGoroutineLeak verifies that MemoryStore doesn't leak goroutines
func TestMemoryStore_NoGoroutineLeak(t *testing.T) {
	defer goleak.VerifyNone(t,
		// Ignore LevelDB goroutines from other tests in the same package
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mpoolDrain"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).compactionError"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).tCompaction"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mCompaction"),
		// Ignore Redis goroutines from failed connection attempts
		goleak.IgnoreTopFunction("github.com/redis/go-redis/v9/maintnotifications.(*CircuitBreakerManager).cleanupLoop"),
	)

	// Create and use a memory store
	store, err := NewMemoryStore("test", MemoryConfig{
		CleanupInterval: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewMemoryStore failed: %v", err)
	}

	ctx := context.Background()

	// Perform some operations
	_ = store.Set(ctx, "key1", []byte("value1"), 1*time.Second)
	_, _ = store.Get(ctx, "key1")
	_ = store.Delete(ctx, "key1")

	// Close the store - this should stop the cleanup goroutine
	err = store.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Give cleanup goroutine time to exit
	time.Sleep(50 * time.Millisecond)

	// goleak.VerifyNone will check at defer time that no goroutines are leaked
}

// TestLevelDBStore_NoGoroutineLeak verifies that LevelDBStore doesn't leak goroutines
func TestLevelDBStore_NoGoroutineLeak(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mpoolDrain"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).compactionError"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).tCompaction"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mCompaction"),
		// Ignore Redis goroutines from failed connection attempts
		goleak.IgnoreTopFunction("github.com/redis/go-redis/v9/maintnotifications.(*CircuitBreakerManager).cleanupLoop"),
	)

	tmpDir := t.TempDir()

	store, err := NewLevelDBStore("test", LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewLevelDBStore failed: %v", err)
	}

	ctx := context.Background()

	// Perform some operations
	_ = store.Set(ctx, "key1", []byte("value1"), 1*time.Second)
	_, _ = store.Get(ctx, "key1")
	_ = store.Delete(ctx, "key1")

	// Close the store - this should stop the cleanup goroutine
	err = store.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Give cleanup goroutine time to exit
	time.Sleep(50 * time.Millisecond)

	// goleak.VerifyNone will check at defer time that no goroutines are leaked
}

// TestMemoryStore_MultipleStoresNoLeak verifies no leaks when creating/closing multiple stores
func TestMemoryStore_MultipleStoresNoLeak(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mpoolDrain"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).compactionError"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).tCompaction"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mCompaction"),
		// Ignore Redis goroutines from failed connection attempts
		goleak.IgnoreTopFunction("github.com/redis/go-redis/v9/maintnotifications.(*CircuitBreakerManager).cleanupLoop"),
	)

	// Create and close multiple stores
	for i := 0; i < 5; i++ {
		store, err := NewMemoryStore("test", MemoryConfig{
			CleanupInterval: 100 * time.Millisecond,
		})
		if err != nil {
			t.Fatalf("NewMemoryStore failed: %v", err)
		}

		ctx := context.Background()
		_ = store.Set(ctx, "key", []byte("value"), 0)

		err = store.Close()
		if err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}

	// Give all cleanup goroutines time to exit
	time.Sleep(100 * time.Millisecond)
}

// TestLevelDBStore_MultipleStoresNoLeak verifies no leaks when creating/closing multiple stores
func TestLevelDBStore_MultipleStoresNoLeak(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mpoolDrain"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).compactionError"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).tCompaction"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mCompaction"),
		// Ignore Redis goroutines from failed connection attempts
		goleak.IgnoreTopFunction("github.com/redis/go-redis/v9/maintnotifications.(*CircuitBreakerManager).cleanupLoop"),
	)

	tmpDir := t.TempDir()

	// Create and close multiple stores
	for i := 0; i < 5; i++ {
		store, err := NewLevelDBStore("test", LevelDBConfig{
			Path:            filepath.Join(tmpDir, "db"),
			CleanupInterval: 100 * time.Millisecond,
		})
		if err != nil {
			t.Fatalf("NewLevelDBStore failed: %v", err)
		}

		ctx := context.Background()
		_ = store.Set(ctx, "key", []byte("value"), 0)

		err = store.Close()
		if err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}

	// Give all cleanup goroutines time to exit
	time.Sleep(100 * time.Millisecond)
}

// TestMemoryStore_AsyncDeleteNoLeak tests that async deletes in Get don't leak goroutines
// This is less critical for MemoryStore but good to verify
func TestMemoryStore_AsyncDeleteNoLeak(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mpoolDrain"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).compactionError"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).tCompaction"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mCompaction"),
		// Ignore Redis goroutines from failed connection attempts
		goleak.IgnoreTopFunction("github.com/redis/go-redis/v9/maintnotifications.(*CircuitBreakerManager).cleanupLoop"),
	)

	store, err := NewMemoryStore("test", MemoryConfig{
		CleanupInterval: 1 * time.Second, // Longer interval to avoid cleanup interference
	})
	if err != nil {
		t.Fatalf("NewMemoryStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// Set keys with very short TTL
	for i := 0; i < 10; i++ {
		_ = store.Set(ctx, "key", []byte("value"), 1*time.Millisecond)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Get expired keys - should trigger async cleanup in some implementations
	for i := 0; i < 10; i++ {
		_, _ = store.Get(ctx, "key")
	}

	// Give time for any async operations to complete
	time.Sleep(50 * time.Millisecond)

	// Close and verify no leaks
	_ = store.Close()
	time.Sleep(50 * time.Millisecond)
}

// TestLevelDBStore_AsyncDeleteNoLeak tests that async deletes in Get don't leak goroutines
func TestLevelDBStore_AsyncDeleteNoLeak(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mpoolDrain"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).compactionError"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).tCompaction"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mCompaction"),
		// Ignore Redis goroutines from failed connection attempts
		goleak.IgnoreTopFunction("github.com/redis/go-redis/v9/maintnotifications.(*CircuitBreakerManager).cleanupLoop"),
	)

	tmpDir := t.TempDir()

	store, err := NewLevelDBStore("test", LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 1 * time.Second, // Longer interval to avoid cleanup interference
	})
	if err != nil {
		t.Fatalf("NewLevelDBStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// Set keys with very short TTL
	for i := 0; i < 10; i++ {
		_ = store.Set(ctx, "key", []byte("value"), 1*time.Millisecond)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Get expired keys - this triggers async delete in LevelDB (line 163)
	for i := 0; i < 10; i++ {
		_, _ = store.Get(ctx, "key")
	}

	// Give time for async goroutines to complete
	time.Sleep(100 * time.Millisecond)

	// Close and verify no leaks
	_ = store.Close()
	time.Sleep(50 * time.Millisecond)
}

// TestMemoryStore_CleanupLoopStops verifies cleanup goroutine properly stops on Close
func TestMemoryStore_CleanupLoopStops(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mpoolDrain"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).compactionError"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).tCompaction"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mCompaction"),
		// Ignore Redis goroutines from failed connection attempts
		goleak.IgnoreTopFunction("github.com/redis/go-redis/v9/maintnotifications.(*CircuitBreakerManager).cleanupLoop"),
	)

	store, err := NewMemoryStore("test", MemoryConfig{
		CleanupInterval: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewMemoryStore failed: %v", err)
	}

	// Let cleanup loop run a few times
	time.Sleep(150 * time.Millisecond)

	// Close should stop the cleanup goroutine
	err = store.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Give cleanup goroutine time to exit
	time.Sleep(100 * time.Millisecond)

	// goleak will verify no goroutines remain
}

// TestLevelDBStore_CleanupLoopStops verifies cleanup goroutine properly stops on Close
func TestLevelDBStore_CleanupLoopStops(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mpoolDrain"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).compactionError"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).tCompaction"),
		goleak.IgnoreTopFunction("github.com/syndtr/goleveldb/leveldb.(*DB).mCompaction"),
		// Ignore Redis goroutines from failed connection attempts
		goleak.IgnoreTopFunction("github.com/redis/go-redis/v9/maintnotifications.(*CircuitBreakerManager).cleanupLoop"),
	)

	tmpDir := t.TempDir()

	store, err := NewLevelDBStore("test", LevelDBConfig{
		Path:            filepath.Join(tmpDir, "db"),
		CleanupInterval: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewLevelDBStore failed: %v", err)
	}

	// Let cleanup loop run a few times
	time.Sleep(150 * time.Millisecond)

	// Close should stop the cleanup goroutine
	err = store.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Give cleanup goroutine time to exit
	time.Sleep(100 * time.Millisecond)

	// goleak will verify no goroutines remain
}
