package session

import (
	"errors"
	"time"

	"github.com/ideamans/chatbotgate/pkg/kvs"
)

// Common errors
var (
	ErrSessionNotFound = errors.New("session: session not found")
)

// Session represents a user session
type Session struct {
	ID            string
	Email         string
	Name          string                 // User's display name from OAuth2 provider
	Provider      string                 // OAuth2 provider name or "email" for email auth
	Extra         map[string]interface{} // Additional user data from OAuth2 provider (for custom forwarding)
	CreatedAt     time.Time
	ExpiresAt     time.Time
	Authenticated bool
}

// IsValid checks if the session is still valid
func (s *Session) IsValid() bool {
	if !s.Authenticated {
		return false
	}
	return time.Now().Before(s.ExpiresAt)
}

// Store is an alias for kvs.Store for backward compatibility
// Use the session helper functions (Get, Set, Delete) to work with sessions
type Store = kvs.Store
