package oauth2

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleProvider is the OAuth2 provider for Google
type GoogleProvider struct {
	config *oauth2.Config
}

// NewGoogleProvider creates a new Google OAuth2 provider
func NewGoogleProvider(clientID, clientSecret, redirectURL string) *GoogleProvider {
	return &GoogleProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
	}
}

// Name returns the provider name
func (p *GoogleProvider) Name() string {
	return "google"
}

// Config returns the OAuth2 config
func (p *GoogleProvider) Config() *oauth2.Config {
	return p.config
}

// GetUserEmail retrieves the user's email from Google
func (p *GoogleProvider) GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	client := p.config.Client(ctx, token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var userInfo struct {
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", fmt.Errorf("failed to decode user info: %w", err)
	}

	if userInfo.Email == "" {
		return "", ErrEmailNotFound
	}

	return userInfo.Email, nil
}
