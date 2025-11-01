package email

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
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

// TokenStore manages email authentication tokens
type TokenStore struct {
	tokens map[string]*Token
	mu     sync.RWMutex
	secret []byte
}

// NewTokenStore creates a new token store
func NewTokenStore(secret string) *TokenStore {
	return &TokenStore{
		tokens: make(map[string]*Token),
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

	// Store token
	token := &Token{
		Value:     tokenValue,
		Email:     email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
		Used:      false,
	}

	s.mu.Lock()
	s.tokens[tokenValue] = token
	s.mu.Unlock()

	return tokenValue, nil
}

// VerifyToken verifies a token and returns the associated email
func (s *TokenStore) VerifyToken(tokenValue string) (string, error) {
	s.mu.RLock()
	token, exists := s.tokens[tokenValue]
	s.mu.RUnlock()

	if !exists {
		return "", ErrTokenNotFound
	}

	if token.Used {
		return "", ErrTokenAlreadyUsed
	}

	if time.Now().After(token.ExpiresAt) {
		return "", ErrTokenExpired
	}

	// Mark as used
	s.mu.Lock()
	token.Used = true
	s.mu.Unlock()

	return token.Email, nil
}

// DeleteToken removes a token from the store
func (s *TokenStore) DeleteToken(tokenValue string) {
	s.mu.Lock()
	delete(s.tokens, tokenValue)
	s.mu.Unlock()
}

// CleanupExpired removes expired tokens
func (s *TokenStore) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for tokenValue, token := range s.tokens {
		if now.After(token.ExpiresAt) {
			delete(s.tokens, tokenValue)
		}
	}
}

// Count returns the number of tokens (for testing)
func (s *TokenStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tokens)
}
