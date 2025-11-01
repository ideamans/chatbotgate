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

// Provider is an interface for OAuth2 providers
type Provider interface {
	Name() string
	Config() *oauth2.Config
	GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error)
}
