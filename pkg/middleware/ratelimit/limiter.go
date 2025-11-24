// Package ratelimit provides rate limiting functionality for email authentication.
//
// This package implements a token bucket algorithm to prevent abuse of magic link emails
// by limiting how many times a user can request login links within a time window.
// Currently used exclusively for email authentication send rate limiting.
package ratelimit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
)

// Limiter implements a simple token bucket rate limiter backed by KVS
// Currently used to rate limit email authentication magic link sends.
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
	// Use context with timeout to prevent hanging on slow KVS operations
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

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
		// Store the new bucket
		if jsonData, err := json.Marshal(b); err == nil {
			if setErr := l.kvs.Set(ctx, key, jsonData, 0); setErr != nil {
				// KVS write failed, but we still allow the request
				// The bucket won't persist, so next request will be treated as first request
				// This is fail-safe: we prefer to allow traffic over blocking it on KVS errors
			}
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
			if setErr := l.kvs.Set(ctx, key, jsonData, 0); setErr != nil {
				// Same fail-safe behavior as above
			}
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
			if setErr := l.kvs.Set(ctx, key, jsonData, 0); setErr != nil {
				// Write failed. The token was already consumed in memory,
				// so we return true to allow the request.
				// Consequence: if this persists, rate limit may not work properly,
				// but this is better than blocking all traffic on KVS errors.
			}
		}
		return true
	}

	return false
}

// Reset clears the rate limit for a specific key
func (l *Limiter) Reset(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = l.kvs.Delete(ctx, key)
}

// Cleanup removes old buckets that haven't been used recently
func (l *Limiter) Cleanup(maxAge time.Duration) {
	// Use a longer timeout for cleanup as it processes multiple keys
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get all keys
	keys, err := l.kvs.List(ctx, "")
	if err != nil {
		return
	}

	now := time.Now()
	for _, key := range keys {
		// Check if context is cancelled (timeout or shutdown)
		select {
		case <-ctx.Done():
			return
		default:
		}

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
