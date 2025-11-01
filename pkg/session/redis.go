package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrRedisSessionNotFound is returned when a session is not found in Redis
	ErrRedisSessionNotFound = errors.New("session not found in Redis")
)

// RedisStore implements session storage using Redis
type RedisStore struct {
	client *redis.Client
	prefix string
	ctx    context.Context
}

// RedisConfig contains Redis connection configuration
type RedisConfig struct {
	Addr     string // Redis server address (host:port)
	Password string // Redis password (optional)
	DB       int    // Redis database number
	Prefix   string // Key prefix for sessions (default: "session:")
}

// NewRedisStore creates a new Redis-based session store
func NewRedisStore(config RedisConfig) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	prefix := config.Prefix
	if prefix == "" {
		prefix = "session:"
	}

	return &RedisStore{
		client: client,
		prefix: prefix,
		ctx:    ctx,
	}, nil
}

// Get retrieves a session from Redis
func (s *RedisStore) Get(id string) (*Session, error) {
	key := s.prefix + id

	data, err := s.client.Get(s.ctx, key).Result()
	if err == redis.Nil {
		return nil, ErrRedisSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session from Redis: %w", err)
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Check if session is expired
	if !session.IsValid() {
		// Delete expired session
		s.Delete(id)
		return nil, ErrRedisSessionNotFound
	}

	return &session, nil
}

// Set stores a session in Redis
func (s *RedisStore) Set(id string, session *Session) error {
	key := s.prefix + id

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Calculate TTL
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return errors.New("session already expired")
	}

	if err := s.client.Set(s.ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set session in Redis: %w", err)
	}

	return nil
}

// Delete removes a session from Redis
func (s *RedisStore) Delete(id string) error {
	key := s.prefix + id

	if err := s.client.Del(s.ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session from Redis: %w", err)
	}

	return nil
}

// Close closes the Redis connection
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Count returns the number of active sessions
func (s *RedisStore) Count() (int, error) {
	pattern := s.prefix + "*"
	keys, err := s.client.Keys(s.ctx, pattern).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}
	return len(keys), nil
}

// List returns all active sessions
func (s *RedisStore) List() ([]*Session, error) {
	pattern := s.prefix + "*"
	keys, err := s.client.Keys(s.ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list session keys: %w", err)
	}

	sessions := make([]*Session, 0, len(keys))
	for _, key := range keys {
		data, err := s.client.Get(s.ctx, key).Result()
		if err != nil {
			continue // Skip on error
		}

		var session Session
		if err := json.Unmarshal([]byte(data), &session); err != nil {
			continue // Skip on error
		}

		if session.IsValid() {
			sessions = append(sessions, &session)
		}
	}

	return sessions, nil
}
