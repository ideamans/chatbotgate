package email

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ideamans/multi-oauth2-proxy/pkg/kvs"
)

var (
	// ErrTokenNotFound is returned when a token is not found
	ErrTokenNotFound = errors.New("token not found")

	// ErrTokenExpired is returned when a token has expired
	ErrTokenExpired = errors.New("token has expired")

	// ErrTokenAlreadyUsed is returned when a token has already been used
	ErrTokenAlreadyUsed = errors.New("token has already been used")
)

// Token represents an email authentication token
type Token struct {
	Value     string
	Email     string
	CreatedAt time.Time
	ExpiresAt time.Time
	Used      bool
}

// IsValid checks if the token is still valid
func (t *Token) IsValid() bool {
	if t.Used {
		return false
	}
	return time.Now().Before(t.ExpiresAt)
}

// TokenStore manages email authentication tokens using a KVS backend
type TokenStore struct {
	kvs    kvs.Store
	secret []byte
}

// NewTokenStore creates a new token store backed by KVS
func NewTokenStore(secret string, kvsStore kvs.Store) *TokenStore {
	return &TokenStore{
		kvs:    kvsStore,
		secret: []byte(secret),
	}
}

// GenerateToken generates a new token for an email address
func (s *TokenStore) GenerateToken(email string, duration time.Duration) (string, error) {
	// Generate random bytes
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create HMAC
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(email))
	h.Write(randomBytes)
	tokenBytes := h.Sum(nil)

	// Encode as base64
	tokenValue := base64.URLEncoding.EncodeToString(tokenBytes)

	// Create token
	token := &Token{
		Value:     tokenValue,
		Email:     email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
		Used:      false,
	}

	// Marshal and store in KVS
	data, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token: %w", err)
	}

	ctx := context.Background()
	if err := s.kvs.Set(ctx, tokenValue, data, duration); err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}

	return tokenValue, nil
}

// VerifyToken verifies a token and returns the associated email
func (s *TokenStore) VerifyToken(tokenValue string) (string, error) {
	ctx := context.Background()

	// Get token from KVS
	data, err := s.kvs.Get(ctx, tokenValue)
	if err != nil {
		if errors.Is(err, kvs.ErrNotFound) {
			return "", ErrTokenNotFound
		}
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return "", fmt.Errorf("failed to unmarshal token: %w", err)
	}

	if token.Used {
		return "", ErrTokenAlreadyUsed
	}

	if time.Now().After(token.ExpiresAt) {
		return "", ErrTokenExpired
	}

	// Mark as used and update in KVS
	token.Used = true
	updatedData, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated token: %w", err)
	}

	ttl := time.Until(token.ExpiresAt)
	if err := s.kvs.Set(ctx, tokenValue, updatedData, ttl); err != nil {
		return "", fmt.Errorf("failed to update token: %w", err)
	}

	return token.Email, nil
}

// DeleteToken removes a token from the store
func (s *TokenStore) DeleteToken(tokenValue string) {
	ctx := context.Background()
	_ = s.kvs.Delete(ctx, tokenValue) // Ignore errors for compatibility
}

// CleanupExpired removes expired tokens (no-op for KVS with TTL support)
// The underlying KVS automatically handles expiration, so this is a no-op for compatibility.
func (s *TokenStore) CleanupExpired() {
	// No-op: KVS implementations with TTL support handle cleanup automatically
}

// Count returns the number of tokens (for testing)
func (s *TokenStore) Count() int {
	ctx := context.Background()
	count, err := s.kvs.Count(ctx, "")
	if err != nil {
		return 0
	}
	return count
}
