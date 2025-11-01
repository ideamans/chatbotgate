package oauth2

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// GitHubProvider is the OAuth2 provider for GitHub
type GitHubProvider struct {
	config *oauth2.Config
}

// NewGitHubProvider creates a new GitHub OAuth2 provider
func NewGitHubProvider(clientID, clientSecret, redirectURL string) *GitHubProvider {
	return &GitHubProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"user:email",
			},
			Endpoint: github.Endpoint,
		},
	}
}

// Name returns the provider name
func (p *GitHubProvider) Name() string {
	return "github"
}

// Config returns the OAuth2 config
func (p *GitHubProvider) Config() *oauth2.Config {
	return p.config
}

// GetUserEmail retrieves the user's email from GitHub
func (p *GitHubProvider) GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	client := p.config.Client(ctx, token)

	// Get user's emails
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", fmt.Errorf("failed to get user emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to get user emails: status %d", resp.StatusCode)
	}

	var emails []struct {
		Email      string `json:"email"`
		Primary    bool   `json:"primary"`
		Verified   bool   `json:"verified"`
		Visibility string `json:"visibility"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("failed to decode user emails: %w", err)
	}

	// Find primary verified email
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	// Fallback to first verified email
	for _, email := range emails {
		if email.Verified {
			return email.Email, nil
		}
	}

	return "", ErrEmailNotFound
}
