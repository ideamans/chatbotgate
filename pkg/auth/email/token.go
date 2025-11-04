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

	"github.com/ideamans/chatbotgate/pkg/kvs"
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
	OTP       string // One-Time Password (12-character alphanumeric)
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

// generateOTP generates a random 12-character OTP using uppercase letters and digits
func generateOTP() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const otpLength = 12

	otpBytes := make([]byte, otpLength)
	randomBytes := make([]byte, otpLength)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	for i := 0; i < otpLength; i++ {
		otpBytes[i] = charset[int(randomBytes[i])%len(charset)]
	}

	return string(otpBytes), nil
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
	// Generate OTP
	otp, err := generateOTP()
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}

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
		OTP:       otp,
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

	// Store OTP-to-token mapping
	otpKey := "otp:" + otp
	if err := s.kvs.Set(ctx, otpKey, []byte(tokenValue), duration); err != nil {
		// Clean up token if OTP mapping fails
		_ = s.kvs.Delete(ctx, tokenValue)
		return "", fmt.Errorf("failed to store OTP mapping: %w", err)
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

// normalizeOTP removes non-alphanumeric characters and takes first 12 characters
func normalizeOTP(input string) string {
	const maxLength = 12
	result := make([]byte, 0, maxLength)

	for i := 0; i < len(input) && len(result) < maxLength; i++ {
		c := input[i]
		// Only keep uppercase letters and digits
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result = append(result, c)
		} else if c >= 'a' && c <= 'z' {
			// Convert lowercase to uppercase
			result = append(result, c-'a'+'A')
		}
	}

	return string(result)
}

// VerifyOTP verifies an OTP and returns the associated email
func (s *TokenStore) VerifyOTP(otpInput string) (string, error) {
	// Normalize the input OTP
	normalizedOTP := normalizeOTP(otpInput)

	if len(normalizedOTP) != 12 {
		return "", ErrTokenNotFound
	}

	ctx := context.Background()

	// We need to find the token by OTP
	// Since KVS doesn't support querying by field, we'll need to add an OTP-to-token mapping
	// For now, we'll use a simple approach: store OTP as a secondary key
	otpKey := "otp:" + normalizedOTP
	tokenValueBytes, err := s.kvs.Get(ctx, otpKey)
	if err != nil {
		if errors.Is(err, kvs.ErrNotFound) {
			return "", ErrTokenNotFound
		}
		return "", fmt.Errorf("failed to get token by OTP: %w", err)
	}

	tokenValue := string(tokenValueBytes)

	// Now verify the actual token
	return s.VerifyToken(tokenValue)
}

// DeleteToken removes a token from the store
func (s *TokenStore) DeleteToken(tokenValue string) {
	ctx := context.Background()

	// Try to get the token to find its OTP
	if data, err := s.kvs.Get(ctx, tokenValue); err == nil {
		var token Token
		if err := json.Unmarshal(data, &token); err == nil && token.OTP != "" {
			// Delete OTP mapping
			otpKey := "otp:" + token.OTP
			_ = s.kvs.Delete(ctx, otpKey)
		}
	}

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
