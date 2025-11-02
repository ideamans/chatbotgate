// Package kvs provides a unified key-value store abstraction
// with implementations for Memory, LevelDB, and Redis.
package kvs

import (
	"context"
	"errors"
	"time"
)

// Store is a key-value store interface that supports TTL and basic operations.
// All implementations must be thread-safe.
type Store interface {
	// Get retrieves a value by key.
	// Returns ErrNotFound if the key does not exist or has expired.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with optional TTL.
	// If ttl is 0, the key will not expire.
	// If ttl is negative, the behavior is implementation-defined (typically treated as 0).
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a key.
	// Does not return an error if the key does not exist.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists and has not expired.
	Exists(ctx context.Context, key string) (bool, error)

	// List returns all keys matching a prefix.
	// Empty prefix returns all keys in the store.
	// The order of keys is not guaranteed.
	List(ctx context.Context, prefix string) ([]string, error)

	// Count returns the number of keys matching a prefix.
	// Empty prefix returns the total count of all keys.
	Count(ctx context.Context, prefix string) (int, error)

	// Close closes the store and releases resources.
	// After Close is called, all operations should return ErrClosed.
	Close() error
}

// Common errors
var (
	// ErrNotFound is returned when a key is not found or has expired.
	ErrNotFound = errors.New("kvs: key not found")

	// ErrClosed is returned when an operation is attempted on a closed store.
	ErrClosed = errors.New("kvs: store is closed")
)

// Config represents the configuration for creating a KVS store.
type Config struct {
	// Type specifies the store type: "memory", "leveldb", or "redis"
	Type string `yaml:"type"`

	// Namespace provides logical isolation within the store.
	// - Memory: uses hierarchical map structure
	// - LevelDB: creates separate directory per namespace
	// - Redis: uses as key prefix
	Namespace string `yaml:"namespace"`

	// Memory-specific config
	Memory MemoryConfig `yaml:"memory"`

	// LevelDB-specific config
	LevelDB LevelDBConfig `yaml:"leveldb"`

	// Redis-specific config
	Redis RedisConfig `yaml:"redis"`
}

// MemoryConfig configures the in-memory store.
type MemoryConfig struct {
	// CleanupInterval is how often to scan for and remove expired keys.
	// Default: 5 minutes
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}

// LevelDBConfig configures the LevelDB store.
type LevelDBConfig struct {
	// Path is the directory path for LevelDB storage.
	// If empty, a temporary directory will be used (OS-dependent).
	Path string `yaml:"path"`

	// SyncWrites enables synchronous writes (slower but safer).
	SyncWrites bool `yaml:"sync_writes"`

	// CleanupInterval is how often to scan for and remove expired keys.
	// Default: 5 minutes
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}

// RedisConfig configures the Redis store.
type RedisConfig struct {
	// Addr is the Redis server address (host:port)
	Addr string `yaml:"addr"`

	// Password is the Redis password (optional)
	Password string `yaml:"password"`

	// DB is the Redis database number (0-15)
	DB int `yaml:"db"`

	// PoolSize is the maximum number of socket connections (0 = default 10 * runtime.NumCPU)
	PoolSize int `yaml:"pool_size"`
}

// New creates a new KVS store based on the provided config.
// The Namespace field provides logical isolation - implementation varies by backend:
// - Memory: separate store instance per namespace
// - LevelDB: separate directory per namespace
// - Redis: key prefix per namespace
func New(cfg Config) (Store, error) {
	switch cfg.Type {
	case "memory", "":
		return NewMemoryStore(cfg.Namespace, cfg.Memory)
	case "leveldb":
		return NewLevelDBStore(cfg.Namespace, cfg.LevelDB)
	case "redis":
		return NewRedisStore(cfg.Namespace, cfg.Redis)
	default:
		return nil, errors.New("kvs: unsupported store type: " + cfg.Type)
	}
}
