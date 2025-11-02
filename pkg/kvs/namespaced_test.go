package kvs

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNamespacedStore_EmptyPrefix(t *testing.T) {
	base, err := NewMemoryStore("", MemoryConfig{})
	if err != nil {
		t.Fatalf("NewMemoryStore failed: %v", err)
	}
	defer base.Close()

	wrapped := NewNamespacedStore(base, "")

	// Empty prefix should return the base store as-is
	if wrapped != base {
		t.Error("NewNamespacedStore with empty prefix should return base store")
	}
}

func TestNamespacedStore_Operations(t *testing.T) {
	ctx := context.Background()
	base, err := NewMemoryStore("", MemoryConfig{})
	if err != nil {
		t.Fatalf("NewMemoryStore failed: %v", err)
	}
	defer base.Close()

	sessionStore := NewNamespacedStore(base, "session:")
	tokenStore := NewNamespacedStore(base, "token:")

	// Set values in different namespaces
	if err := sessionStore.Set(ctx, "user1", []byte("session-data-1"), 0); err != nil {
		t.Fatalf("sessionStore.Set failed: %v", err)
	}
	if err := tokenStore.Set(ctx, "user1", []byte("token-data-1"), 0); err != nil {
		t.Fatalf("tokenStore.Set failed: %v", err)
	}

	// Get should retrieve correct values (namespace isolation)
	sessionVal, err := sessionStore.Get(ctx, "user1")
	if err != nil {
		t.Fatalf("sessionStore.Get failed: %v", err)
	}
	if string(sessionVal) != "session-data-1" {
		t.Errorf("sessionStore.Get returned %q, want %q", string(sessionVal), "session-data-1")
	}

	tokenVal, err := tokenStore.Get(ctx, "user1")
	if err != nil {
		t.Fatalf("tokenStore.Get failed: %v", err)
	}
	if string(tokenVal) != "token-data-1" {
		t.Errorf("tokenStore.Get returned %q, want %q", string(tokenVal), "token-data-1")
	}

	// Exists should work correctly
	exists, err := sessionStore.Exists(ctx, "user1")
	if err != nil {
		t.Fatalf("sessionStore.Exists failed: %v", err)
	}
	if !exists {
		t.Error("sessionStore.Exists returned false, want true")
	}

	// Non-existent key
	exists, err = sessionStore.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("sessionStore.Exists failed: %v", err)
	}
	if exists {
		t.Error("sessionStore.Exists returned true, want false")
	}
}

func TestNamespacedStore_List(t *testing.T) {
	ctx := context.Background()
	base, err := NewMemoryStore("", MemoryConfig{})
	if err != nil {
		t.Fatalf("NewMemoryStore failed: %v", err)
	}
	defer base.Close()

	sessionStore := NewNamespacedStore(base, "session:")
	tokenStore := NewNamespacedStore(base, "token:")

	// Add keys to both namespaces
	sessionStore.Set(ctx, "user1", []byte("s1"), 0)
	sessionStore.Set(ctx, "user2", []byte("s2"), 0)
	sessionStore.Set(ctx, "admin", []byte("s3"), 0)
	tokenStore.Set(ctx, "token1", []byte("t1"), 0)
	tokenStore.Set(ctx, "token2", []byte("t2"), 0)

	// List session keys - should only return session namespace keys
	sessionKeys, err := sessionStore.List(ctx, "")
	if err != nil {
		t.Fatalf("sessionStore.List failed: %v", err)
	}
	if len(sessionKeys) != 3 {
		t.Errorf("sessionStore.List returned %d keys, want 3", len(sessionKeys))
	}
	// Keys should NOT have "session:" prefix (transparent to caller)
	for _, key := range sessionKeys {
		if strings.HasPrefix(key, "session:") {
			t.Errorf("sessionStore.List returned key with prefix: %s", key)
		}
	}

	// List token keys - should only return token namespace keys
	tokenKeys, err := tokenStore.List(ctx, "")
	if err != nil {
		t.Fatalf("tokenStore.List failed: %v", err)
	}
	if len(tokenKeys) != 2 {
		t.Errorf("tokenStore.List returned %d keys, want 2", len(tokenKeys))
	}

	// List with prefix within namespace
	userKeys, err := sessionStore.List(ctx, "user")
	if err != nil {
		t.Fatalf("sessionStore.List with prefix failed: %v", err)
	}
	if len(userKeys) != 2 {
		t.Errorf("sessionStore.List(\"user\") returned %d keys, want 2", len(userKeys))
	}
}

func TestNamespacedStore_Count(t *testing.T) {
	ctx := context.Background()
	base, err := NewMemoryStore("", MemoryConfig{})
	if err != nil {
		t.Fatalf("NewMemoryStore failed: %v", err)
	}
	defer base.Close()

	sessionStore := NewNamespacedStore(base, "session:")
	tokenStore := NewNamespacedStore(base, "token:")

	// Add keys
	sessionStore.Set(ctx, "user1", []byte("s1"), 0)
	sessionStore.Set(ctx, "user2", []byte("s2"), 0)
	tokenStore.Set(ctx, "token1", []byte("t1"), 0)

	// Count session keys - should only count session namespace
	count, err := sessionStore.Count(ctx, "")
	if err != nil {
		t.Fatalf("sessionStore.Count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("sessionStore.Count returned %d, want 2", count)
	}

	// Count token keys - should only count token namespace
	count, err = tokenStore.Count(ctx, "")
	if err != nil {
		t.Fatalf("tokenStore.Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("tokenStore.Count returned %d, want 1", count)
	}
}

func TestNamespacedStore_Delete(t *testing.T) {
	ctx := context.Background()
	base, err := NewMemoryStore("", MemoryConfig{})
	if err != nil {
		t.Fatalf("NewMemoryStore failed: %v", err)
	}
	defer base.Close()

	sessionStore := NewNamespacedStore(base, "session:")

	// Set and delete
	sessionStore.Set(ctx, "user1", []byte("data"), 0)

	if err := sessionStore.Delete(ctx, "user1"); err != nil {
		t.Fatalf("sessionStore.Delete failed: %v", err)
	}

	// Should not exist anymore
	exists, err := sessionStore.Exists(ctx, "user1")
	if err != nil {
		t.Fatalf("sessionStore.Exists failed: %v", err)
	}
	if exists {
		t.Error("sessionStore.Exists returned true after Delete, want false")
	}
}

func TestNamespacedStore_TTL(t *testing.T) {
	ctx := context.Background()
	base, err := NewMemoryStore("", MemoryConfig{})
	if err != nil {
		t.Fatalf("NewMemoryStore failed: %v", err)
	}
	defer base.Close()

	sessionStore := NewNamespacedStore(base, "session:")

	// Set with short TTL
	if err := sessionStore.Set(ctx, "temp", []byte("data"), 100*time.Millisecond); err != nil {
		t.Fatalf("sessionStore.Set failed: %v", err)
	}

	// Should exist immediately
	exists, err := sessionStore.Exists(ctx, "temp")
	if err != nil {
		t.Fatalf("sessionStore.Exists failed: %v", err)
	}
	if !exists {
		t.Error("sessionStore.Exists returned false, want true")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not exist anymore (auto-deleted by TTL)
	_, err = sessionStore.Get(ctx, "temp")
	if err != ErrNotFound {
		t.Errorf("sessionStore.Get after expiration returned error %v, want ErrNotFound", err)
	}
}

func TestNamespacedStore_NamespaceIsolation(t *testing.T) {
	ctx := context.Background()
	base, err := NewMemoryStore("", MemoryConfig{})
	if err != nil {
		t.Fatalf("NewMemoryStore failed: %v", err)
	}
	defer base.Close()

	// Create three namespaced stores sharing the same base
	sessionStore := NewNamespacedStore(base, "session:")
	tokenStore := NewNamespacedStore(base, "token:")
	rateLimitStore := NewNamespacedStore(base, "ratelimit:")

	// Add data to each namespace
	sessionStore.Set(ctx, "key1", []byte("session1"), 0)
	sessionStore.Set(ctx, "key2", []byte("session2"), 0)
	tokenStore.Set(ctx, "key1", []byte("token1"), 0)
	rateLimitStore.Set(ctx, "ip1", []byte("limit1"), 0)
	rateLimitStore.Set(ctx, "ip2", []byte("limit2"), 0)

	// Verify counts are isolated
	sessionCount, _ := sessionStore.Count(ctx, "")
	tokenCount, _ := tokenStore.Count(ctx, "")
	rateLimitCount, _ := rateLimitStore.Count(ctx, "")

	if sessionCount != 2 {
		t.Errorf("sessionStore.Count returned %d, want 2", sessionCount)
	}
	if tokenCount != 1 {
		t.Errorf("tokenStore.Count returned %d, want 1", tokenCount)
	}
	if rateLimitCount != 2 {
		t.Errorf("rateLimitStore.Count returned %d, want 2", rateLimitCount)
	}

	// Verify Lists are isolated
	sessionKeys, _ := sessionStore.List(ctx, "")
	tokenKeys, _ := tokenStore.List(ctx, "")
	rateLimitKeys, _ := rateLimitStore.List(ctx, "")

	if len(sessionKeys) != 2 {
		t.Errorf("sessionStore.List returned %d keys, want 2", len(sessionKeys))
	}
	if len(tokenKeys) != 1 {
		t.Errorf("tokenStore.List returned %d keys, want 1", len(tokenKeys))
	}
	if len(rateLimitKeys) != 2 {
		t.Errorf("rateLimitStore.List returned %d keys, want 2", len(rateLimitKeys))
	}

	// Verify Get is isolated (same key "key1" in different namespaces)
	sessionVal, _ := sessionStore.Get(ctx, "key1")
	tokenVal, _ := tokenStore.Get(ctx, "key1")

	if string(sessionVal) != "session1" {
		t.Errorf("sessionStore.Get(key1) returned %q, want session1", string(sessionVal))
	}
	if string(tokenVal) != "token1" {
		t.Errorf("tokenStore.Get(key1) returned %q, want token1", string(tokenVal))
	}
}
