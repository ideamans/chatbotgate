package email

import (
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/kvs"
)

// createTestTokenStore creates a token store with memory-based KVS for testing
func createTestTokenStore(secret string) *TokenStore {
	kvsStore, _ := kvs.NewMemoryStore("token:", kvs.MemoryConfig{
		CleanupInterval: 1 * time.Minute,
	})
	return NewTokenStore(secret, kvsStore)
}

func TestToken_IsValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		token *Token
		want  bool
	}{
		{
			name: "valid token",
			token: &Token{
				Value:     "test",
				Email:     "user@example.com",
				CreatedAt: now,
				ExpiresAt: now.Add(15 * time.Minute),
				Used:      false,
			},
			want: true,
		},
		{
			name: "expired token",
			token: &Token{
				Value:     "test",
				Email:     "user@example.com",
				CreatedAt: now.Add(-30 * time.Minute),
				ExpiresAt: now.Add(-15 * time.Minute),
				Used:      false,
			},
			want: false,
		},
		{
			name: "used token",
			token: &Token{
				Value:     "test",
				Email:     "user@example.com",
				CreatedAt: now,
				ExpiresAt: now.Add(15 * time.Minute),
				Used:      true,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsValid(); got != tt.want {
				t.Errorf("Token.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenStore_GenerateToken(t *testing.T) {
	store := createTestTokenStore("test-secret")

	email := "user@example.com"
	duration := 15 * time.Minute

	token, err := store.GenerateToken(email, duration)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}

	// Verify token exists in store
	if store.Count() != 1 {
		t.Errorf("store should have 1 token, got %d", store.Count())
	}
}

func TestTokenStore_VerifyToken(t *testing.T) {
	store := createTestTokenStore("test-secret")

	email := "user@example.com"
	token, err := store.GenerateToken(email, 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Verify valid token
	verifiedEmail, err := store.VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}

	if verifiedEmail != email {
		t.Errorf("VerifyToken() email = %s, want %s", verifiedEmail, email)
	}

	// Try to verify again - should fail (one-time use)
	_, err = store.VerifyToken(token)
	if err != ErrTokenAlreadyUsed {
		t.Errorf("VerifyToken() second call error = %v, want ErrTokenAlreadyUsed", err)
	}
}

func TestTokenStore_VerifyToken_NotFound(t *testing.T) {
	store := createTestTokenStore("test-secret")

	_, err := store.VerifyToken("nonexistent-token")
	if err != ErrTokenNotFound {
		t.Errorf("VerifyToken() error = %v, want ErrTokenNotFound", err)
	}
}

func TestTokenStore_VerifyToken_Expired(t *testing.T) {
	store := createTestTokenStore("test-secret")

	email := "user@example.com"
	token, err := store.GenerateToken(email, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Wait for token to expire (KVS will auto-delete)
	time.Sleep(10 * time.Millisecond)

	_, err = store.VerifyToken(token)
	// With KVS, expired tokens are auto-deleted by TTL, so we get ErrTokenNotFound
	if err != ErrTokenNotFound {
		t.Errorf("VerifyToken() error = %v, want ErrTokenNotFound (expired tokens are auto-deleted by KVS TTL)", err)
	}
}

func TestTokenStore_DeleteToken(t *testing.T) {
	store := createTestTokenStore("test-secret")

	email := "user@example.com"
	token, err := store.GenerateToken(email, 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Delete token
	store.DeleteToken(token)

	// Verify token is gone
	_, err = store.VerifyToken(token)
	if err != ErrTokenNotFound {
		t.Errorf("VerifyToken() after delete error = %v, want ErrTokenNotFound", err)
	}
}

func TestTokenStore_CleanupExpired(t *testing.T) {
	store := createTestTokenStore("test-secret")

	// Create an expired token
	store.GenerateToken("expired@example.com", 1*time.Millisecond)

	// Create a valid token
	store.GenerateToken("valid@example.com", 15*time.Minute)

	// Wait for first token to expire
	time.Sleep(10 * time.Millisecond)

	// Clean up
	store.CleanupExpired()

	// Should have only 1 token left
	if store.Count() != 1 {
		t.Errorf("after cleanup, store should have 1 token, got %d", store.Count())
	}
}

func TestTokenStore_MultipleTokens(t *testing.T) {
	store := createTestTokenStore("test-secret")

	// Generate multiple tokens
	token1, _ := store.GenerateToken("user1@example.com", 15*time.Minute)
	token2, _ := store.GenerateToken("user2@example.com", 15*time.Minute)

	// Verify they are different
	if token1 == token2 {
		t.Error("tokens should be unique")
	}

	// Verify both work
	email1, err := store.VerifyToken(token1)
	if err != nil || email1 != "user1@example.com" {
		t.Error("token1 verification failed")
	}

	email2, err := store.VerifyToken(token2)
	if err != nil || email2 != "user2@example.com" {
		t.Error("token2 verification failed")
	}
}
