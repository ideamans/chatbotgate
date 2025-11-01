package session

import "time"

// Session represents a user session
type Session struct {
	ID           string
	Email        string
	Provider     string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	Authenticated bool
}

// IsValid checks if the session is still valid
func (s *Session) IsValid() bool {
	if !s.Authenticated {
		return false
	}
	return time.Now().Before(s.ExpiresAt)
}

// Store is an interface for session storage
type Store interface {
	Get(id string) (*Session, error)
	Set(id string, session *Session) error
	Delete(id string) error
}
