package kvs

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// LevelDBStore is a LevelDB-based implementation of Store.
// It provides persistent storage on the filesystem with background cleanup of expired keys.
type LevelDBStore struct {
	prefix          string
	db              *leveldb.DB
	closed          bool
	mu              sync.RWMutex
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	cleanupDone     chan struct{}
}

// NewLevelDBStore creates a new LevelDB KVS store.
func NewLevelDBStore(prefix string, cfg LevelDBConfig) (*LevelDBStore, error) {
	// Resolve path
	dbPath := cfg.Path
	if dbPath == "" {
		// Use OS cache directory if no path specified
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			// Fallback to temp directory
			cacheDir = os.TempDir()
		}

		// Create a unique directory name based on prefix
		dirName := "multi-oauth2-proxy"
		if prefix != "" {
			// Sanitize prefix for use in directory name
			sanitized := strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
					return r
				}
				return '-'
			}, prefix)
			dirName = fmt.Sprintf("%s-%s", dirName, sanitized)
		}

		dbPath = filepath.Join(cacheDir, dirName)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("kvs/leveldb: failed to create directory: %w", err)
	}

	// Open LevelDB
	opts := &opt.Options{
		Strict:      opt.DefaultStrict,
		Compression: opt.SnappyCompression,
	}
	if cfg.SyncWrites {
		opts.WriteBuffer = 4 * 1024 * 1024 // 4MB
		opts.NoSync = false
	}

	db, err := leveldb.OpenFile(dbPath, opts)
	if err != nil {
		// Try to recover if database is corrupted
		if _, ok := err.(*errors.ErrCorrupted); ok {
			db, err = leveldb.RecoverFile(dbPath, nil)
		}
		if err != nil {
			return nil, fmt.Errorf("kvs/leveldb: failed to open database at %s: %w", dbPath, err)
		}
	}

	cleanupInterval := cfg.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 5 * time.Minute // Default cleanup every 5 minutes
	}

	store := &LevelDBStore{
		prefix:          prefix,
		db:              db,
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
		cleanupDone:     make(chan struct{}),
	}

	// Start background cleanup goroutine
	go store.cleanupLoop()

	return store, nil
}

// prefixedKey returns the key with prefix prepended.
func (l *LevelDBStore) prefixedKey(key string) string {
	if l.prefix == "" {
		return key
	}
	return l.prefix + key
}

// encodeValue encodes a value with optional expiration time.
// Format: [8 bytes: expiration unix nano (0 = no expiration)][value bytes]
func encodeValue(value []byte, ttl time.Duration) []byte {
	expiresAt := int64(0)
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl).UnixNano()
	}

	encoded := make([]byte, 8+len(value))
	binary.BigEndian.PutUint64(encoded[0:8], uint64(expiresAt))
	copy(encoded[8:], value)
	return encoded
}

// decodeValue decodes a value and checks expiration.
// Returns (value, expired, error)
func decodeValue(encoded []byte) ([]byte, bool, error) {
	if len(encoded) < 8 {
		return nil, false, fmt.Errorf("kvs/leveldb: invalid encoded value (too short)")
	}

	expiresAt := int64(binary.BigEndian.Uint64(encoded[0:8]))
	value := encoded[8:]

	// Check expiration
	if expiresAt > 0 && time.Now().UnixNano() > expiresAt {
		return nil, true, nil
	}

	return value, false, nil
}

// Get retrieves a value by key.
func (l *LevelDBStore) Get(ctx context.Context, key string) ([]byte, error) {
	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return nil, ErrClosed
	}
	l.mu.RUnlock()

	encoded, err := l.db.Get([]byte(l.prefixedKey(key)), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("kvs/leveldb: get failed: %w", err)
	}

	value, expired, err := decodeValue(encoded)
	if err != nil {
		return nil, err
	}
	if expired {
		// Delete expired key asynchronously
		go l.Delete(context.Background(), key)
		return nil, ErrNotFound
	}

	return value, nil
}

// Set stores a value with optional TTL.
func (l *LevelDBStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return ErrClosed
	}
	l.mu.RUnlock()

	encoded := encodeValue(value, ttl)
	err := l.db.Put([]byte(l.prefixedKey(key)), encoded, nil)
	if err != nil {
		return fmt.Errorf("kvs/leveldb: set failed: %w", err)
	}

	return nil
}

// Delete removes a key.
func (l *LevelDBStore) Delete(ctx context.Context, key string) error {
	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return ErrClosed
	}
	l.mu.RUnlock()

	err := l.db.Delete([]byte(l.prefixedKey(key)), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return fmt.Errorf("kvs/leveldb: delete failed: %w", err)
	}

	return nil
}

// Exists checks if a key exists and has not expired.
func (l *LevelDBStore) Exists(ctx context.Context, key string) (bool, error) {
	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return false, ErrClosed
	}
	l.mu.RUnlock()

	encoded, err := l.db.Get([]byte(l.prefixedKey(key)), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return false, nil
		}
		return false, fmt.Errorf("kvs/leveldb: exists check failed: %w", err)
	}

	_, expired, err := decodeValue(encoded)
	if err != nil {
		return false, err
	}

	return !expired, nil
}

// List returns all keys matching a prefix.
func (l *LevelDBStore) List(ctx context.Context, keyPrefix string) ([]string, error) {
	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return nil, ErrClosed
	}
	l.mu.RUnlock()

	fullPrefix := l.prefixedKey(keyPrefix)
	prefixBytes := []byte(fullPrefix)

	iter := l.db.NewIterator(util.BytesPrefix(prefixBytes), nil)
	defer iter.Release()

	var keys []string
	for iter.Next() {
		key := string(iter.Key())
		value := iter.Value()

		// Check expiration
		_, expired, err := decodeValue(value)
		if err != nil {
			continue // Skip malformed entries
		}
		if expired {
			continue // Skip expired entries
		}

		// Remove store prefix to return clean key
		cleanKey := key
		if l.prefix != "" && strings.HasPrefix(key, l.prefix) {
			cleanKey = strings.TrimPrefix(key, l.prefix)
		}

		keys = append(keys, cleanKey)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("kvs/leveldb: iteration failed: %w", err)
	}

	return keys, nil
}

// Count returns the number of keys matching a prefix.
func (l *LevelDBStore) Count(ctx context.Context, prefix string) (int, error) {
	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return 0, ErrClosed
	}
	l.mu.RUnlock()

	fullPrefix := l.prefixedKey(prefix)
	prefixBytes := []byte(fullPrefix)

	iter := l.db.NewIterator(util.BytesPrefix(prefixBytes), nil)
	defer iter.Release()

	count := 0
	for iter.Next() {
		value := iter.Value()

		// Check expiration
		_, expired, err := decodeValue(value)
		if err != nil {
			continue // Skip malformed entries
		}
		if !expired {
			count++
		}
	}

	if err := iter.Error(); err != nil {
		return 0, fmt.Errorf("kvs/leveldb: count failed: %w", err)
	}

	return count, nil
}

// Close closes the LevelDB database and stops the cleanup goroutine.
func (l *LevelDBStore) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return ErrClosed
	}
	l.closed = true
	l.mu.Unlock()

	// Stop cleanup goroutine
	close(l.stopCleanup)
	<-l.cleanupDone

	// Close database
	err := l.db.Close()
	if err != nil {
		return fmt.Errorf("kvs/leveldb: close failed: %w", err)
	}

	return nil
}

// cleanupLoop runs periodically to remove expired keys.
func (l *LevelDBStore) cleanupLoop() {
	defer close(l.cleanupDone)

	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.stopCleanup:
			return
		}
	}
}

// cleanup scans all keys with our prefix and deletes expired ones.
func (l *LevelDBStore) cleanup() {
	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return
	}
	l.mu.RUnlock()

	// Scan all keys with our prefix
	prefixBytes := []byte(l.prefix)
	iter := l.db.NewIterator(util.BytesPrefix(prefixBytes), nil)
	defer iter.Release()

	now := time.Now().UnixNano()
	var keysToDelete [][]byte

	for iter.Next() {
		value := iter.Value()

		// Decode expiration time
		if len(value) < 8 {
			continue
		}
		expiresAt := int64(binary.BigEndian.Uint64(value[0:8]))

		// If expired, mark for deletion
		if expiresAt > 0 && now > expiresAt {
			// Make a copy of the key
			keyCopy := make([]byte, len(iter.Key()))
			copy(keyCopy, iter.Key())
			keysToDelete = append(keysToDelete, keyCopy)
		}
	}

	// Delete expired keys in batch
	if len(keysToDelete) > 0 {
		batch := new(leveldb.Batch)
		for _, key := range keysToDelete {
			batch.Delete(key)
		}
		_ = l.db.Write(batch, nil) // Ignore errors during cleanup
	}
}
