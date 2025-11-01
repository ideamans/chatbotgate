package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter_Allow(t *testing.T) {
	// Allow 3 requests per second
	limiter := NewLimiter(3, 1*time.Second)

	key := "test-key"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !limiter.Allow(key) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be blocked
	if limiter.Allow(key) {
		t.Error("4th request should be blocked")
	}
}

func TestLimiter_Refill(t *testing.T) {
	// Allow 2 requests per 100ms
	limiter := NewLimiter(2, 100*time.Millisecond)

	key := "test-key"

	// Use up the tokens
	limiter.Allow(key)
	limiter.Allow(key)

	// Should be blocked
	if limiter.Allow(key) {
		t.Error("3rd request should be blocked immediately")
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow(key) {
		t.Error("request should be allowed after refill")
	}
}

func TestLimiter_MultipleKeys(t *testing.T) {
	limiter := NewLimiter(2, 1*time.Second)

	// Different keys should have independent limits
	if !limiter.Allow("key1") {
		t.Error("key1 first request should be allowed")
	}

	if !limiter.Allow("key2") {
		t.Error("key2 first request should be allowed")
	}

	// Each key should have its own bucket
	limiter.Allow("key1") // key1 now has 0 tokens
	limiter.Allow("key2") // key2 now has 0 tokens

	// Both should be blocked
	if limiter.Allow("key1") {
		t.Error("key1 should be blocked")
	}

	if limiter.Allow("key2") {
		t.Error("key2 should be blocked")
	}
}

func TestLimiter_Reset(t *testing.T) {
	limiter := NewLimiter(1, 1*time.Second)

	key := "test-key"

	// Use up the token
	limiter.Allow(key)

	// Should be blocked
	if limiter.Allow(key) {
		t.Error("should be blocked before reset")
	}

	// Reset
	limiter.Reset(key)

	// Should be allowed again
	if !limiter.Allow(key) {
		t.Error("should be allowed after reset")
	}
}

func TestLimiter_Cleanup(t *testing.T) {
	limiter := NewLimiter(5, 1*time.Second)

	// Create some buckets
	limiter.Allow("key1")
	limiter.Allow("key2")
	limiter.Allow("key3")

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Add a new key
	limiter.Allow("key4")

	// Cleanup old buckets (older than 40ms)
	limiter.Cleanup(40 * time.Millisecond)

	// key1, key2, key3 should be cleaned up
	// key4 should still exist

	// We can't directly check the map, but we can verify behavior
	// If cleaned up properly, keys should start fresh
	limiter.mu.RLock()
	count := len(limiter.buckets)
	limiter.mu.RUnlock()

	// Only key4 should remain
	if count != 1 {
		t.Errorf("after cleanup, expected 1 bucket, got %d", count)
	}
}

func TestLimiter_ZeroRate(t *testing.T) {
	limiter := NewLimiter(0, 1*time.Second)

	key := "test-key"

	// With 0 rate, all requests should be blocked
	if limiter.Allow(key) {
		t.Error("with 0 rate, all requests should be blocked")
	}
}

func TestLimiter_HighRate(t *testing.T) {
	// Allow 100 requests per second
	limiter := NewLimiter(100, 1*time.Second)

	key := "test-key"

	// All 100 should be allowed
	for i := 0; i < 100; i++ {
		if !limiter.Allow(key) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 101st should be blocked
	if limiter.Allow(key) {
		t.Error("101st request should be blocked")
	}
}
