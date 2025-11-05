package oauth2

import (
	"context"
	"errors"

	"golang.org/x/oauth2"
)

var (
	// ErrInvalidToken is returned when the OAuth2 token is invalid
	ErrInvalidToken = errors.New("invalid OAuth2 token")

	// ErrEmailNotFound is returned when user email is not found in OAuth2 response
	ErrEmailNotFound = errors.New("user email not found in OAuth2 response")
)

// UserInfo represents user information from OAuth2 provider
type UserInfo struct {
	Email string                 // User's email address
	Name  string                 // User's display name (optional)
	Extra map[string]interface{} // Additional data from OAuth2 provider (for custom forwarding)
}

// Provider is an interface for OAuth2 providers
type Provider interface {
	Name() string
	Config() *oauth2.Config
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
	// Deprecated: Use GetUserInfo instead
	GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error)
}
