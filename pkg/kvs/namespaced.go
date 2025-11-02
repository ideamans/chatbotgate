package kvs

import (
	"context"
	"strings"
	"time"
)

// NamespacedStore wraps a Store and prepends a prefix to all keys.
// This allows multiple logical stores to share a single physical KVS backend
// while maintaining isolation.
//
// Example:
//
//	baseKVS, _ := kvs.New(kvs.Config{Type: "redis", ...})
//	sessionStore := kvs.NewNamespacedStore(baseKVS, "session:")
//	tokenStore := kvs.NewNamespacedStore(baseKVS, "token:")
//	rateLimitStore := kvs.NewNamespacedStore(baseKVS, "ratelimit:")
//
// When rateLimitStore.List("") is called, it only scans keys with "ratelimit:" prefix,
// not the entire database. This solves the cleanup scanning issue.
type NamespacedStore struct {
	store  Store
	prefix string
}

// NewNamespacedStore creates a new namespaced store wrapper.
// If prefix is empty, it returns the underlying store as-is for efficiency.
func NewNamespacedStore(store Store, prefix string) Store {
	if prefix == "" {
		return store
	}
	return &NamespacedStore{
		store:  store,
		prefix: prefix,
	}
}

// prefixKey prepends the namespace prefix to a key.
func (n *NamespacedStore) prefixKey(key string) string {
	return n.prefix + key
}

// unprefixKey removes the namespace prefix from a key.
func (n *NamespacedStore) unprefixKey(key string) string {
	return strings.TrimPrefix(key, n.prefix)
}

// Get retrieves a value by key (with prefix prepended).
func (n *NamespacedStore) Get(ctx context.Context, key string) ([]byte, error) {
	return n.store.Get(ctx, n.prefixKey(key))
}

// Set stores a value with optional TTL (with prefix prepended).
func (n *NamespacedStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return n.store.Set(ctx, n.prefixKey(key), value, ttl)
}

// Delete removes a key (with prefix prepended).
func (n *NamespacedStore) Delete(ctx context.Context, key string) error {
	return n.store.Delete(ctx, n.prefixKey(key))
}

// Exists checks if a key exists (with prefix prepended).
func (n *NamespacedStore) Exists(ctx context.Context, key string) (bool, error) {
	return n.store.Exists(ctx, n.prefixKey(key))
}

// List returns all keys matching a prefix (with namespace prefix prepended).
// The returned keys have the namespace prefix removed for transparency.
//
// Example:
//
//	// Store has keys: "session:user1", "session:user2", "token:abc"
//	sessionStore := NewNamespacedStore(store, "session:")
//	keys, _ := sessionStore.List("")  // Returns: ["user1", "user2"]
func (n *NamespacedStore) List(ctx context.Context, keyPrefix string) ([]string, error) {
	fullPrefix := n.prefixKey(keyPrefix)
	keys, err := n.store.List(ctx, fullPrefix)
	if err != nil {
		return nil, err
	}

	// Remove namespace prefix from all keys
	result := make([]string, len(keys))
	for i, key := range keys {
		result[i] = n.unprefixKey(key)
	}
	return result, nil
}

// Count returns the number of keys matching a prefix (with namespace prefix prepended).
func (n *NamespacedStore) Count(ctx context.Context, prefix string) (int, error) {
	fullPrefix := n.prefixKey(prefix)
	return n.store.Count(ctx, fullPrefix)
}

// Close closes the underlying store.
//
// IMPORTANT: If multiple NamespacedStore instances share the same underlying store,
// closing one will close the store for all. The caller is responsible for managing
// the lifecycle of the shared underlying store. Typically, you should only call Close()
// on the base store itself, not on the namespaced wrappers.
func (n *NamespacedStore) Close() error {
	return n.store.Close()
}
