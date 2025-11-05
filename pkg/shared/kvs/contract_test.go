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

// ContractTestSuite runs a comprehensive test suite against any Store implementation
// This ensures all backends (Memory, LevelDB, Redis) behave consistently
type ContractTestSuite struct {
	t         *testing.T
	store     Store
	cleanup   func()
	skipRedis bool // Skip tests that require Redis to be running
}

// NewContractTestSuite creates a new contract test suite
func NewContractTestSuite(t *testing.T, store Store, cleanup func()) *ContractTestSuite {
	return &ContractTestSuite{
		t:       t,
		store:   store,
		cleanup: cleanup,
	}
}

// RunAll runs all contract tests
func (s *ContractTestSuite) RunAll() {
	s.t.Run("BasicGetSet", func(t *testing.T) { s.TestBasicGetSet() })
	s.t.Run("GetNonExistent", func(t *testing.T) { s.TestGetNonExistent() })
	s.t.Run("SetWithTTL", func(t *testing.T) { s.TestSetWithTTL() })
	s.t.Run("Delete", func(t *testing.T) { s.TestDelete() })
	s.t.Run("DeleteNonExistent", func(t *testing.T) { s.TestDeleteNonExistent() })
	s.t.Run("Exists", func(t *testing.T) { s.TestExists() })
	s.t.Run("List", func(t *testing.T) { s.TestList() })
	s.t.Run("ListWithPrefix", func(t *testing.T) { s.TestListWithPrefix() })
	s.t.Run("ListEmpty", func(t *testing.T) { s.TestListEmpty() })
	s.t.Run("Count", func(t *testing.T) { s.TestCount() })
	s.t.Run("CountWithPrefix", func(t *testing.T) { s.TestCountWithPrefix() })
	s.t.Run("TTLExpiration", func(t *testing.T) { s.TestTTLExpiration() })
	s.t.Run("OverwriteKey", func(t *testing.T) { s.TestOverwriteKey() })
	s.t.Run("Close", func(t *testing.T) { s.TestClose() })
	s.t.Run("OperationsAfterClose", func(t *testing.T) { s.TestOperationsAfterClose() })

	// Cleanup after all tests
	if s.cleanup != nil {
		s.cleanup()
	}
}

// TestBasicGetSet tests basic Set and Get operations
func (s *ContractTestSuite) TestBasicGetSet() {
	ctx := context.Background()

	// Set a value
	err := s.store.Set(ctx, "test-key", []byte("test-value"), 0)
	require.NoError(s.t, err, "Set should not return error")

	// Get the value back
	val, err := s.store.Get(ctx, "test-key")
	require.NoError(s.t, err, "Get should not return error")
	assert.Equal(s.t, []byte("test-value"), val, "Retrieved value should match")
}

// TestGetNonExistent tests Get on a non-existent key
func (s *ContractTestSuite) TestGetNonExistent() {
	ctx := context.Background()

	val, err := s.store.Get(ctx, "non-existent-key")
	assert.Equal(s.t, ErrNotFound, err, "Should return ErrNotFound")
	assert.Nil(s.t, val, "Value should be nil")
}

// TestSetWithTTL tests Set with TTL (basic check, actual expiration tested separately)
func (s *ContractTestSuite) TestSetWithTTL() {
	ctx := context.Background()

	// Set with TTL
	err := s.store.Set(ctx, "ttl-key", []byte("ttl-value"), 1*time.Hour)
	require.NoError(s.t, err, "Set with TTL should not return error")

	// Should be immediately retrievable
	val, err := s.store.Get(ctx, "ttl-key")
	require.NoError(s.t, err, "Get should not return error")
	assert.Equal(s.t, []byte("ttl-value"), val, "Retrieved value should match")

	// Clean up
	_ = s.store.Delete(ctx, "ttl-key")
}

// TestDelete tests Delete operation
func (s *ContractTestSuite) TestDelete() {
	ctx := context.Background()

	// Set a value
	err := s.store.Set(ctx, "delete-key", []byte("delete-value"), 0)
	require.NoError(s.t, err, "Set should not return error")

	// Verify it exists
	exists, err := s.store.Exists(ctx, "delete-key")
	require.NoError(s.t, err, "Exists should not return error")
	assert.True(s.t, exists, "Key should exist")

	// Delete it
	err = s.store.Delete(ctx, "delete-key")
	require.NoError(s.t, err, "Delete should not return error")

	// Verify it's gone
	exists, err = s.store.Exists(ctx, "delete-key")
	require.NoError(s.t, err, "Exists should not return error")
	assert.False(s.t, exists, "Key should not exist after delete")
}

// TestDeleteNonExistent tests Delete on a non-existent key (should not error)
func (s *ContractTestSuite) TestDeleteNonExistent() {
	ctx := context.Background()

	err := s.store.Delete(ctx, "non-existent-delete-key")
	assert.NoError(s.t, err, "Delete on non-existent key should not return error")
}

// TestExists tests Exists operation
func (s *ContractTestSuite) TestExists() {
	ctx := context.Background()

	// Non-existent key
	exists, err := s.store.Exists(ctx, "exists-test-key")
	require.NoError(s.t, err, "Exists should not return error")
	assert.False(s.t, exists, "Non-existent key should return false")

	// Set a key
	err = s.store.Set(ctx, "exists-test-key", []byte("exists-value"), 0)
	require.NoError(s.t, err, "Set should not return error")

	// Should now exist
	exists, err = s.store.Exists(ctx, "exists-test-key")
	require.NoError(s.t, err, "Exists should not return error")
	assert.True(s.t, exists, "Existing key should return true")

	// Clean up
	_ = s.store.Delete(ctx, "exists-test-key")
}

// TestList tests List operation
func (s *ContractTestSuite) TestList() {
	ctx := context.Background()

	// Set multiple keys
	keys := []string{"list-a", "list-b", "list-c"}
	for _, key := range keys {
		err := s.store.Set(ctx, key, []byte("value"), 0)
		require.NoError(s.t, err, "Set should not return error")
	}

	// List all keys with "list-" prefix
	result, err := s.store.List(ctx, "list-")
	require.NoError(s.t, err, "List should not return error")
	assert.Len(s.t, result, 3, "Should return 3 keys")

	// Verify all keys are present
	for _, key := range keys {
		assert.Contains(s.t, result, key, "Result should contain key")
	}

	// Clean up
	for _, key := range keys {
		_ = s.store.Delete(ctx, key)
	}
}

// TestListWithPrefix tests List with different prefixes
func (s *ContractTestSuite) TestListWithPrefix() {
	ctx := context.Background()

	// Set keys with different prefixes
	_ = s.store.Set(ctx, "prefix1-a", []byte("value"), 0)
	_ = s.store.Set(ctx, "prefix1-b", []byte("value"), 0)
	_ = s.store.Set(ctx, "prefix2-a", []byte("value"), 0)

	// List only prefix1
	result, err := s.store.List(ctx, "prefix1-")
	require.NoError(s.t, err, "List should not return error")
	assert.Len(s.t, result, 2, "Should return 2 keys")
	assert.Contains(s.t, result, "prefix1-a")
	assert.Contains(s.t, result, "prefix1-b")
	assert.NotContains(s.t, result, "prefix2-a")

	// Clean up
	_ = s.store.Delete(ctx, "prefix1-a")
	_ = s.store.Delete(ctx, "prefix1-b")
	_ = s.store.Delete(ctx, "prefix2-a")
}

// TestListEmpty tests List when no keys match
func (s *ContractTestSuite) TestListEmpty() {
	ctx := context.Background()

	result, err := s.store.List(ctx, "non-existent-prefix-")
	require.NoError(s.t, err, "List should not return error")
	assert.Empty(s.t, result, "Should return empty slice")
}

// TestCount tests Count operation
func (s *ContractTestSuite) TestCount() {
	ctx := context.Background()

	// Set multiple keys
	keys := []string{"count-a", "count-b", "count-c"}
	for _, key := range keys {
		err := s.store.Set(ctx, key, []byte("value"), 0)
		require.NoError(s.t, err, "Set should not return error")
	}

	// Count all keys with "count-" prefix
	count, err := s.store.Count(ctx, "count-")
	require.NoError(s.t, err, "Count should not return error")
	assert.Equal(s.t, 3, count, "Should count 3 keys")

	// Clean up
	for _, key := range keys {
		_ = s.store.Delete(ctx, key)
	}
}

// TestCountWithPrefix tests Count with different prefixes
func (s *ContractTestSuite) TestCountWithPrefix() {
	ctx := context.Background()

	// Set keys with different prefixes
	_ = s.store.Set(ctx, "cprefix1-a", []byte("value"), 0)
	_ = s.store.Set(ctx, "cprefix1-b", []byte("value"), 0)
	_ = s.store.Set(ctx, "cprefix2-a", []byte("value"), 0)

	// Count only cprefix1
	count, err := s.store.Count(ctx, "cprefix1-")
	require.NoError(s.t, err, "Count should not return error")
	assert.Equal(s.t, 2, count, "Should count 2 keys")

	// Clean up
	_ = s.store.Delete(ctx, "cprefix1-a")
	_ = s.store.Delete(ctx, "cprefix1-b")
	_ = s.store.Delete(ctx, "cprefix2-a")
}

// TestTTLExpiration tests that keys with TTL actually expire
// Note: This test takes time to run (waits for expiration)
func (s *ContractTestSuite) TestTTLExpiration() {
	ctx := context.Background()

	// Set a key with very short TTL
	ttl := 100 * time.Millisecond
	err := s.store.Set(ctx, "expire-key", []byte("expire-value"), ttl)
	require.NoError(s.t, err, "Set with TTL should not return error")

	// Should exist immediately
	exists, err := s.store.Exists(ctx, "expire-key")
	require.NoError(s.t, err, "Exists should not return error")
	assert.True(s.t, exists, "Key should exist immediately after set")

	// Wait for expiration (with buffer for cleanup interval)
	// For Memory/Redis this should be immediate, for LevelDB we need to wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// For LevelDB, we may need to wait for the cleanup loop to run
	// Try for up to 6 seconds (default cleanup interval is 5s)
	maxWait := 6 * time.Second
	start := time.Now()
	expired := false
	for time.Since(start) < maxWait {
		exists, err = s.store.Exists(ctx, "expire-key")
		require.NoError(s.t, err, "Exists should not return error")
		if !exists {
			expired = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	assert.True(s.t, expired, "Key should have expired after TTL + cleanup interval")
}

// TestOverwriteKey tests overwriting an existing key
func (s *ContractTestSuite) TestOverwriteKey() {
	ctx := context.Background()

	// Set initial value
	err := s.store.Set(ctx, "overwrite-key", []byte("value1"), 0)
	require.NoError(s.t, err, "Set should not return error")

	// Overwrite with new value
	err = s.store.Set(ctx, "overwrite-key", []byte("value2"), 0)
	require.NoError(s.t, err, "Set should not return error")

	// Should get the new value
	val, err := s.store.Get(ctx, "overwrite-key")
	require.NoError(s.t, err, "Get should not return error")
	assert.Equal(s.t, []byte("value2"), val, "Should get overwritten value")

	// Clean up
	_ = s.store.Delete(ctx, "overwrite-key")
}

// TestClose tests that Close works without error
func (s *ContractTestSuite) TestClose() {
	// Note: We don't actually call Close here because it would break subsequent tests
	// The Close behavior is tested in TestOperationsAfterClose with a separate store instance
	// This test is a placeholder to document the expected behavior
	s.t.Log("Close behavior tested in TestOperationsAfterClose")
}

// TestOperationsAfterClose tests that operations fail after Close
func (s *ContractTestSuite) TestOperationsAfterClose() {
	// Note: This test should be run last or with a separate store instance
	// We skip this in the contract suite and test it separately per backend
	s.t.Log("Operations after Close tested separately per backend")
}

// TestMemoryStoreContract runs contract tests for MemoryStore
func TestMemoryStoreContract(t *testing.T) {
	config := Config{
		Type: "memory",
		Memory: MemoryConfig{
			CleanupInterval: 100 * time.Millisecond, // Faster cleanup for tests
		},
	}

	store, err := New(config)
	require.NoError(t, err, "Should create MemoryStore")

	cleanup := func() {
		store.Close()
	}

	suite := NewContractTestSuite(t, store, cleanup)
	suite.RunAll()
}

// TestLevelDBStoreContract runs contract tests for LevelDBStore
func TestLevelDBStoreContract(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "kvs-leveldb-contract-test-*")
	require.NoError(t, err, "Should create temp dir")

	config := Config{
		Type: "leveldb",
		LevelDB: LevelDBConfig{
			Path:            filepath.Join(tmpDir, "db"),
			SyncWrites:      false,
			CleanupInterval: 100 * time.Millisecond, // Faster cleanup for tests
		},
	}

	store, err := New(config)
	require.NoError(t, err, "Should create LevelDBStore")

	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	suite := NewContractTestSuite(t, store, cleanup)
	suite.RunAll()
}

// TestRedisStoreContract runs contract tests for RedisStore
// This test requires a Redis server to be running (skipped if not available)
func TestRedisStoreContract(t *testing.T) {
	// Skip if Redis is not available
	// This can be controlled via build tags or environment variables
	if testing.Short() {
		t.Skip("Skipping Redis contract tests in short mode")
	}

	config := Config{
		Type: "redis",
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       15, // Use DB 15 for tests to avoid conflicts
			PoolSize: 10,
		},
	}

	store, err := New(config)
	if err != nil {
		t.Skipf("Redis not available, skipping contract tests: %v", err)
		return
	}

	cleanup := func() {
		// Clear all test keys before closing
		ctx := context.Background()
		keys, _ := store.List(ctx, "")
		for _, key := range keys {
			_ = store.Delete(ctx, key)
		}
		store.Close()
	}

	suite := NewContractTestSuite(t, store, cleanup)
	suite.RunAll()
}

// TestMemoryStoreCloseOperations tests operations after Close for MemoryStore
func TestMemoryStoreCloseOperations(t *testing.T) {
	config := Config{
		Type: "memory",
		Memory: MemoryConfig{
			CleanupInterval: 1 * time.Second,
		},
	}

	store, err := New(config)
	require.NoError(t, err, "Should create MemoryStore")

	// Close the store
	err = store.Close()
	require.NoError(t, err, "Close should not return error")

	ctx := context.Background()

	// All operations should return ErrClosed
	_, err = store.Get(ctx, "key")
	assert.Equal(t, ErrClosed, err, "Get after Close should return ErrClosed")

	err = store.Set(ctx, "key", []byte("value"), 0)
	assert.Equal(t, ErrClosed, err, "Set after Close should return ErrClosed")

	err = store.Delete(ctx, "key")
	assert.Equal(t, ErrClosed, err, "Delete after Close should return ErrClosed")

	_, err = store.Exists(ctx, "key")
	assert.Equal(t, ErrClosed, err, "Exists after Close should return ErrClosed")

	_, err = store.List(ctx, "")
	assert.Equal(t, ErrClosed, err, "List after Close should return ErrClosed")

	_, err = store.Count(ctx, "")
	assert.Equal(t, ErrClosed, err, "Count after Close should return ErrClosed")
}

// TestLevelDBStoreCloseOperations tests operations after Close for LevelDBStore
func TestLevelDBStoreCloseOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kvs-leveldb-close-test-*")
	require.NoError(t, err, "Should create temp dir")
	defer os.RemoveAll(tmpDir)

	config := Config{
		Type: "leveldb",
		LevelDB: LevelDBConfig{
			Path:            filepath.Join(tmpDir, "db"),
			SyncWrites:      false,
			CleanupInterval: 1 * time.Second,
		},
	}

	store, err := New(config)
	require.NoError(t, err, "Should create LevelDBStore")

	// Close the store
	err = store.Close()
	require.NoError(t, err, "Close should not return error")

	ctx := context.Background()

	// All operations should return ErrClosed
	_, err = store.Get(ctx, "key")
	assert.Equal(t, ErrClosed, err, "Get after Close should return ErrClosed")

	err = store.Set(ctx, "key", []byte("value"), 0)
	assert.Equal(t, ErrClosed, err, "Set after Close should return ErrClosed")

	err = store.Delete(ctx, "key")
	assert.Equal(t, ErrClosed, err, "Delete after Close should return ErrClosed")

	_, err = store.Exists(ctx, "key")
	assert.Equal(t, ErrClosed, err, "Exists after Close should return ErrClosed")

	_, err = store.List(ctx, "")
	assert.Equal(t, ErrClosed, err, "List after Close should return ErrClosed")

	_, err = store.Count(ctx, "")
	assert.Equal(t, ErrClosed, err, "Count after Close should return ErrClosed")
}

// TestRedisStoreCloseOperations tests operations after Close for RedisStore
func TestRedisStoreCloseOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis close tests in short mode")
	}

	config := Config{
		Type: "redis",
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       15,
			PoolSize: 10,
		},
	}

	store, err := New(config)
	if err != nil {
		t.Skipf("Redis not available, skipping close tests: %v", err)
		return
	}

	// Close the store
	err = store.Close()
	require.NoError(t, err, "Close should not return error")

	ctx := context.Background()

	// All operations should return ErrClosed
	_, err = store.Get(ctx, "key")
	assert.Equal(t, ErrClosed, err, "Get after Close should return ErrClosed")

	err = store.Set(ctx, "key", []byte("value"), 0)
	assert.Equal(t, ErrClosed, err, "Set after Close should return ErrClosed")

	err = store.Delete(ctx, "key")
	assert.Equal(t, ErrClosed, err, "Delete after Close should return ErrClosed")

	_, err = store.Exists(ctx, "key")
	assert.Equal(t, ErrClosed, err, "Exists after Close should return ErrClosed")

	_, err = store.List(ctx, "")
	assert.Equal(t, ErrClosed, err, "List after Close should return ErrClosed")

	_, err = store.Count(ctx, "")
	assert.Equal(t, ErrClosed, err, "Count after Close should return ErrClosed")
}
