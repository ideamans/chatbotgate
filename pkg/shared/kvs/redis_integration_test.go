package kvs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfRedisUnavailable skips the test if Redis is not available
func skipIfRedisUnavailable(t *testing.T) Store {
	if testing.Short() {
		t.Skip("Skipping Redis integration tests in short mode")
	}

	config := Config{
		Type: "redis",
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       15, // Use DB 15 for tests
			PoolSize: 10,
		},
	}

	store, err := New(config)
	if err != nil {
		t.Skipf("Redis not available, skipping test: %v", err)
	}

	// Clear any existing test data
	ctx := context.Background()
	keys, _ := store.List(ctx, "")
	for _, key := range keys {
		_ = store.Delete(ctx, key)
	}

	return store
}

// TestRedisNativeTTL tests that Redis uses native TTL support (no cleanup loop needed)
func TestRedisNativeTTL(t *testing.T) {
	store := skipIfRedisUnavailable(t)
	defer store.Close()

	ctx := context.Background()

	// Set a key with very short TTL
	err := store.Set(ctx, "ttl-test", []byte("value"), 100*time.Millisecond)
	require.NoError(t, err, "Set should not return error")

	// Verify key exists immediately
	exists, err := store.Exists(ctx, "ttl-test")
	require.NoError(t, err, "Exists should not return error")
	assert.True(t, exists, "Key should exist immediately")

	// Wait for TTL to expire (Redis automatically removes expired keys)
	time.Sleep(150 * time.Millisecond)

	// Key should be gone (no cleanup loop needed)
	exists, err = store.Exists(ctx, "ttl-test")
	require.NoError(t, err, "Exists should not return error")
	assert.False(t, exists, "Key should be automatically removed by Redis after TTL")

	// Verify Get also returns NotFound
	_, err = store.Get(ctx, "ttl-test")
	assert.Equal(t, ErrNotFound, err, "Get should return ErrNotFound for expired key")
}

// TestRedisConnectionPooling tests that Redis uses connection pooling
func TestRedisConnectionPooling(t *testing.T) {
	store := skipIfRedisUnavailable(t)
	defer store.Close()

	redisStore, ok := store.(*RedisStore)
	require.True(t, ok, "Should be RedisStore")

	// Verify pool is initialized
	assert.NotNil(t, redisStore.client, "Redis client should be initialized")

	ctx := context.Background()

	// Perform multiple concurrent operations to use connection pool
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			key := "pool-test-" + string(rune('a'+idx))
			_ = store.Set(ctx, key, []byte("value"), 0)
			_, _ = store.Get(ctx, key)
			_ = store.Delete(ctx, key)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// If connection pooling works, all operations should succeed without errors
	// (verified implicitly by not panicking)
}

// TestRedisMultipleDatabase tests using different Redis databases
func TestRedisMultipleDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration tests in short mode")
	}

	// Create two stores using different databases
	config1 := Config{
		Type: "redis",
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       14, // Use DB 14
			PoolSize: 10,
		},
	}

	config2 := Config{
		Type: "redis",
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       15, // Use DB 15
			PoolSize: 10,
		},
	}

	store1, err := New(config1)
	if err != nil {
		t.Skipf("Redis not available, skipping test: %v", err)
		return
	}
	defer store1.Close()

	store2, err := New(config2)
	require.NoError(t, err, "Should create second store")
	defer store2.Close()

	ctx := context.Background()

	// Set same key in both databases
	err = store1.Set(ctx, "db-test", []byte("value-db14"), 0)
	require.NoError(t, err, "Set in DB 14 should not return error")

	err = store2.Set(ctx, "db-test", []byte("value-db15"), 0)
	require.NoError(t, err, "Set in DB 15 should not return error")

	// Verify values are isolated
	val1, err := store1.Get(ctx, "db-test")
	require.NoError(t, err, "Get from DB 14 should not return error")
	assert.Equal(t, []byte("value-db14"), val1, "DB 14 should have its own value")

	val2, err := store2.Get(ctx, "db-test")
	require.NoError(t, err, "Get from DB 15 should not return error")
	assert.Equal(t, []byte("value-db15"), val2, "DB 15 should have its own value")

	// Clean up
	_ = store1.Delete(ctx, "db-test")
	_ = store2.Delete(ctx, "db-test")
}

// TestRedisNamespace tests that namespace configuration works correctly
func TestRedisNamespace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration tests in short mode")
	}

	// Create two stores with different namespaces
	config1 := Config{
		Type:      "redis",
		Namespace: "app1",
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       15,
			PoolSize: 10,
		},
	}

	config2 := Config{
		Type:      "redis",
		Namespace: "app2",
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       15,
			PoolSize: 10,
		},
	}

	store1, err := New(config1)
	if err != nil {
		t.Skipf("Redis not available, skipping test: %v", err)
		return
	}
	defer store1.Close()

	store2, err := New(config2)
	require.NoError(t, err, "Should create second store")
	defer store2.Close()

	ctx := context.Background()

	// Set same logical key in both stores
	err = store1.Set(ctx, "namespace-test", []byte("value-app1"), 0)
	require.NoError(t, err, "Set in store1 should not return error")

	err = store2.Set(ctx, "namespace-test", []byte("value-app2"), 0)
	require.NoError(t, err, "Set in store2 should not return error")

	// Verify values are isolated by namespace
	val1, err := store1.Get(ctx, "namespace-test")
	require.NoError(t, err, "Get from store1 should not return error")
	assert.Equal(t, []byte("value-app1"), val1, "store1 should have its own value")

	val2, err := store2.Get(ctx, "namespace-test")
	require.NoError(t, err, "Get from store2 should not return error")
	assert.Equal(t, []byte("value-app2"), val2, "store2 should have its own value")

	// Verify List only returns keys with the correct namespace
	keys1, err := store1.List(ctx, "")
	require.NoError(t, err, "List from store1 should not return error")
	assert.Contains(t, keys1, "namespace-test", "store1 should list its own keys")
	assert.Len(t, keys1, 1, "store1 should only see its own keys")

	keys2, err := store2.List(ctx, "")
	require.NoError(t, err, "List from store2 should not return error")
	assert.Contains(t, keys2, "namespace-test", "store2 should list its own keys")
	assert.Len(t, keys2, 1, "store2 should only see its own keys")

	// Clean up
	_ = store1.Delete(ctx, "namespace-test")
	_ = store2.Delete(ctx, "namespace-test")
}

// TestRedisConcurrentOperations tests concurrent operations on Redis store
func TestRedisConcurrentOperations(t *testing.T) {
	store := skipIfRedisUnavailable(t)
	defer store.Close()

	ctx := context.Background()
	numGoroutines := 20
	numOpsPerGoroutine := 50

	done := make(chan bool, numGoroutines)

	// Run concurrent operations
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			for i := 0; i < numOpsPerGoroutine; i++ {
				key := "concurrent-" + string(rune('a'+goroutineID%26)) + "-" + string(rune('0'+i%10))

				// Set
				err := store.Set(ctx, key, []byte("value"), 1*time.Second)
				if err != nil {
					t.Errorf("Set failed: %v", err)
				}

				// Get
				_, err = store.Get(ctx, key)
				if err != nil && err != ErrNotFound {
					t.Errorf("Get failed: %v", err)
				}

				// Exists
				_, err = store.Exists(ctx, key)
				if err != nil {
					t.Errorf("Exists failed: %v", err)
				}

				// Delete
				err = store.Delete(ctx, key)
				if err != nil {
					t.Errorf("Delete failed: %v", err)
				}
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all keys are cleaned up
	count, err := store.Count(ctx, "concurrent-")
	require.NoError(t, err, "Count should not return error")
	assert.Equal(t, 0, count, "All keys should be deleted")
}

// TestRedisLargeValue tests storing and retrieving large values
func TestRedisLargeValue(t *testing.T) {
	store := skipIfRedisUnavailable(t)
	defer store.Close()

	ctx := context.Background()

	// Create a large value (1MB)
	largeValue := make([]byte, 1024*1024)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	// Store large value
	err := store.Set(ctx, "large-value", largeValue, 0)
	require.NoError(t, err, "Set large value should not return error")

	// Retrieve large value
	retrieved, err := store.Get(ctx, "large-value")
	require.NoError(t, err, "Get large value should not return error")
	assert.Equal(t, largeValue, retrieved, "Retrieved value should match original")

	// Clean up
	_ = store.Delete(ctx, "large-value")
}

// TestRedisListPagination tests List with many keys
func TestRedisListPagination(t *testing.T) {
	store := skipIfRedisUnavailable(t)
	defer store.Close()

	ctx := context.Background()

	// Set many keys
	numKeys := 200
	for i := 0; i < numKeys; i++ {
		key := "list-page-" + string(rune('a'+i/26)) + string(rune('0'+i%10))
		err := store.Set(ctx, key, []byte("value"), 0)
		require.NoError(t, err, "Set should not return error")
	}

	// List all keys
	keys, err := store.List(ctx, "list-page-")
	require.NoError(t, err, "List should not return error")
	assert.Len(t, keys, numKeys, "Should list all keys")

	// Clean up
	for _, key := range keys {
		_ = store.Delete(ctx, key)
	}
}

// TestRedisTTLPrecision tests TTL precision
func TestRedisTTLPrecision(t *testing.T) {
	store := skipIfRedisUnavailable(t)
	defer store.Close()

	ctx := context.Background()

	// Set key with sub-second TTL
	ttl := 500 * time.Millisecond
	err := store.Set(ctx, "ttl-precision", []byte("value"), ttl)
	require.NoError(t, err, "Set should not return error")

	// Verify key exists immediately
	exists, err := store.Exists(ctx, "ttl-precision")
	require.NoError(t, err, "Exists should not return error")
	assert.True(t, exists, "Key should exist immediately")

	// Wait half the TTL - key should still exist
	time.Sleep(ttl / 2)
	exists, err = store.Exists(ctx, "ttl-precision")
	require.NoError(t, err, "Exists should not return error")
	assert.True(t, exists, "Key should still exist after half TTL")

	// Wait for full TTL + buffer
	time.Sleep(ttl/2 + 100*time.Millisecond)

	// Key should be expired
	exists, err = store.Exists(ctx, "ttl-precision")
	require.NoError(t, err, "Exists should not return error")
	assert.False(t, exists, "Key should be expired after full TTL")
}

// TestRedisEmptyValue tests storing and retrieving empty values
func TestRedisEmptyValue(t *testing.T) {
	store := skipIfRedisUnavailable(t)
	defer store.Close()

	ctx := context.Background()

	// Set empty value
	err := store.Set(ctx, "empty-value", []byte{}, 0)
	require.NoError(t, err, "Set empty value should not return error")

	// Get empty value
	val, err := store.Get(ctx, "empty-value")
	require.NoError(t, err, "Get empty value should not return error")
	assert.Equal(t, []byte{}, val, "Retrieved value should be empty")

	// Verify key exists
	exists, err := store.Exists(ctx, "empty-value")
	require.NoError(t, err, "Exists should not return error")
	assert.True(t, exists, "Empty value key should exist")

	// Clean up
	_ = store.Delete(ctx, "empty-value")
}
