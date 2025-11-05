package ratelimit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
)

// Limiter implements a simple token bucket rate limiter backed by KVS
type Limiter struct {
	kvs      kvs.Store
	rate     int // tokens per interval
	interval time.Duration
}

// bucket represents a token bucket for a specific key
type bucket struct {
	Tokens     int       `json:"tokens"`
	LastRefill time.Time `json:"last_refill"`
}

// NewLimiter creates a new rate limiter backed by KVS
// rate: number of allowed requests
// interval: time window for the rate
func NewLimiter(rate int, interval time.Duration, kvsStore kvs.Store) *Limiter {
	return &Limiter{
		kvs:      kvsStore,
		rate:     rate,
		interval: interval,
	}
}

// Allow checks if a request is allowed for the given key
func (l *Limiter) Allow(key string) bool {
	ctx := context.Background()

	// Handle zero rate case
	if l.rate <= 0 {
		return false
	}

	// Try to get existing bucket
	data, err := l.kvs.Get(ctx, key)
	var b bucket

	if err != nil {
		// First request for this key
		b = bucket{
			Tokens:     l.rate - 1,
			LastRefill: time.Now(),
		}
		// Store and return true
		if jsonData, err := json.Marshal(b); err == nil {
			_ = l.kvs.Set(ctx, key, jsonData, 0) // No TTL
		}
		return true
	}

	// Unmarshal existing bucket
	if err := json.Unmarshal(data, &b); err != nil {
		// Corrupted data, treat as new bucket
		b = bucket{
			Tokens:     l.rate - 1,
			LastRefill: time.Now(),
		}
		if jsonData, err := json.Marshal(b); err == nil {
			_ = l.kvs.Set(ctx, key, jsonData, 0)
		}
		return true
	}

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(b.LastRefill)

	if elapsed >= l.interval {
		// Full refill
		intervalsElapsed := int(elapsed / l.interval)
		b.Tokens = l.rate
		b.LastRefill = b.LastRefill.Add(time.Duration(intervalsElapsed) * l.interval)
	}

	// Check if tokens available
	if b.Tokens > 0 {
		b.Tokens--
		// Update bucket
		if jsonData, err := json.Marshal(b); err == nil {
			_ = l.kvs.Set(ctx, key, jsonData, 0) // No TTL
		}
		return true
	}

	return false
}

// Reset clears the rate limit for a specific key
func (l *Limiter) Reset(key string) {
	ctx := context.Background()
	_ = l.kvs.Delete(ctx, key)
}

// Cleanup removes old buckets that haven't been used recently
func (l *Limiter) Cleanup(maxAge time.Duration) {
	ctx := context.Background()

	// Get all keys
	keys, err := l.kvs.List(ctx, "")
	if err != nil {
		return
	}

	now := time.Now()
	for _, key := range keys {
		// Get bucket data
		data, err := l.kvs.Get(ctx, key)
		if err != nil {
			continue
		}

		var b bucket
		if err := json.Unmarshal(data, &b); err != nil {
			// Delete corrupted data
			_ = l.kvs.Delete(ctx, key)
			continue
		}

		// Delete if too old
		if now.Sub(b.LastRefill) > maxAge {
			_ = l.kvs.Delete(ctx, key)
		}
	}
}
