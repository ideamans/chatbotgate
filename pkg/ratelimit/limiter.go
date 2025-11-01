package ratelimit

import (
	"sync"
	"time"
)

// Limiter implements a simple token bucket rate limiter
type Limiter struct {
	buckets map[string]*bucket
	mu      sync.RWMutex
	rate    int           // tokens per interval
	interval time.Duration
}

// bucket represents a token bucket for a specific key
type bucket struct {
	tokens     int
	lastRefill time.Time
}

// NewLimiter creates a new rate limiter
// rate: number of allowed requests
// interval: time window for the rate
func NewLimiter(rate int, interval time.Duration) *Limiter {
	return &Limiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		interval: interval,
	}
}

// Allow checks if a request is allowed for the given key
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Handle zero rate case
	if l.rate <= 0 {
		return false
	}

	b, exists := l.buckets[key]
	if !exists {
		// First request for this key
		l.buckets[key] = &bucket{
			tokens:     l.rate - 1,
			lastRefill: time.Now(),
		}
		return true
	}

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)

	if elapsed >= l.interval {
		// Full refill
		intervalsElapsed := int(elapsed / l.interval)
		b.tokens = l.rate
		b.lastRefill = b.lastRefill.Add(time.Duration(intervalsElapsed) * l.interval)
	}

	// Check if tokens available
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// Reset clears the rate limit for a specific key
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

// Cleanup removes old buckets that haven't been used recently
func (l *Limiter) Cleanup(maxAge time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for key, b := range l.buckets {
		if now.Sub(b.lastRefill) > maxAge {
			delete(l.buckets, key)
		}
	}
}
