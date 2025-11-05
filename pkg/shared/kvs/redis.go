package kvs

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a Redis-based implementation of Store.
// It provides distributed, persistent storage backed by Redis.
// Namespace isolation is implemented using key prefixes (namespace:key format).
type RedisStore struct {
	namespace string // Stored as "namespace:" prefix for Redis keys
	client    *redis.Client
	closed    bool
	mu        sync.RWMutex
}

// NewRedisStore creates a new Redis KVS store for the given namespace.
// Namespace isolation is achieved using key prefixes.
func NewRedisStore(namespace string, cfg RedisConfig) (*RedisStore, error) {
	opts := &redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}

	client := redis.NewClient(opts)

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("kvs/redis: failed to connect to %s: %w", cfg.Addr, err)
	}

	// Convert namespace to prefix format (namespace:)
	prefix := ""
	if namespace != "" {
		prefix = namespace + ":"
	}

	return &RedisStore{
		namespace: prefix,
		client:    client,
	}, nil
}

// prefixedKey returns the key with namespace prefix prepended.
func (r *RedisStore) prefixedKey(key string) string {
	if r.namespace == "" {
		return key
	}
	return r.namespace + key
}

// Get retrieves a value by key.
func (r *RedisStore) Get(ctx context.Context, key string) ([]byte, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, ErrClosed
	}
	r.mu.RUnlock()

	result, err := r.client.Get(ctx, r.prefixedKey(key)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("kvs/redis: get failed: %w", err)
	}

	return result, nil
}

// Set stores a value with optional TTL.
func (r *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return ErrClosed
	}
	r.mu.RUnlock()

	err := r.client.Set(ctx, r.prefixedKey(key), value, ttl).Err()
	if err != nil {
		return fmt.Errorf("kvs/redis: set failed: %w", err)
	}

	return nil
}

// Delete removes a key.
func (r *RedisStore) Delete(ctx context.Context, key string) error {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return ErrClosed
	}
	r.mu.RUnlock()

	err := r.client.Del(ctx, r.prefixedKey(key)).Err()
	if err != nil {
		return fmt.Errorf("kvs/redis: delete failed: %w", err)
	}

	return nil
}

// Exists checks if a key exists and has not expired.
func (r *RedisStore) Exists(ctx context.Context, key string) (bool, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return false, ErrClosed
	}
	r.mu.RUnlock()

	count, err := r.client.Exists(ctx, r.prefixedKey(key)).Result()
	if err != nil {
		return false, fmt.Errorf("kvs/redis: exists check failed: %w", err)
	}

	return count > 0, nil
}

// List returns all keys matching a prefix.
func (r *RedisStore) List(ctx context.Context, keyPrefix string) ([]string, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, ErrClosed
	}
	r.mu.RUnlock()

	fullPrefix := r.prefixedKey(keyPrefix)
	pattern := fullPrefix + "*"

	// Use SCAN instead of KEYS for better performance on large datasets
	var keys []string
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()

		// Remove namespace prefix to return clean key
		cleanKey := key
		if r.namespace != "" && strings.HasPrefix(key, r.namespace) {
			cleanKey = strings.TrimPrefix(key, r.namespace)
		}

		keys = append(keys, cleanKey)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("kvs/redis: list failed: %w", err)
	}

	return keys, nil
}

// Count returns the number of keys matching a prefix.
func (r *RedisStore) Count(ctx context.Context, prefix string) (int, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return 0, ErrClosed
	}
	r.mu.RUnlock()

	fullPrefix := r.prefixedKey(prefix)
	pattern := fullPrefix + "*"

	// Use SCAN to count keys
	count := 0
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		count++
	}

	if err := iter.Err(); err != nil {
		return 0, fmt.Errorf("kvs/redis: count failed: %w", err)
	}

	return count, nil
}

// Close closes the Redis connection.
func (r *RedisStore) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return ErrClosed
	}
	r.closed = true
	r.mu.Unlock()

	err := r.client.Close()
	if err != nil {
		return fmt.Errorf("kvs/redis: close failed: %w", err)
	}

	return nil
}
