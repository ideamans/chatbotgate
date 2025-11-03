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
func NewGitHubProvider(clientID, clientSecret, redirectURL string, scopes []string, resetScopes bool) *GitHubProvider {
	// Default scopes
	defaultScopes := []string{
		"user:email",
		"read:user", // For accessing user profile (name)
	}

	// Determine final scopes based on resetScopes flag
	var finalScopes []string
	if resetScopes {
		// Replace default scopes with provided scopes
		if len(scopes) == 0 {
			finalScopes = defaultScopes // Fallback to default if no scopes provided
		} else {
			finalScopes = scopes
		}
	} else {
		// Add provided scopes to default scopes
		finalScopes = append(defaultScopes, scopes...)
	}

	return &GitHubProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       finalScopes,
			Endpoint:     github.Endpoint,
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

// GetUserInfo retrieves the user's information from GitHub
func (p *GitHubProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := p.config.Client(ctx, token)

	// Get user profile (name)
	var userName string
	userResp, err := client.Get("https://api.github.com/user")
	if err == nil && userResp.StatusCode == 200 {
		defer userResp.Body.Close()
		var user struct {
			Name  string `json:"name"`
			Login string `json:"login"`
		}
		if json.NewDecoder(userResp.Body).Decode(&user) == nil {
			userName = user.Name
			if userName == "" {
				userName = user.Login // Fallback to login if name is not set
			}
		}
	}

	// Get user's emails
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return nil, fmt.Errorf("failed to get user emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get user emails: status %d", resp.StatusCode)
	}

	var emails []struct {
		Email      string `json:"email"`
		Primary    bool   `json:"primary"`
		Verified   bool   `json:"verified"`
		Visibility string `json:"visibility"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, fmt.Errorf("failed to decode user emails: %w", err)
	}

	var email string
	// Find primary verified email
	for _, e := range emails {
		if e.Primary && e.Verified {
			email = e.Email
			break
		}
	}

	// Fallback to first verified email
	if email == "" {
		for _, e := range emails {
			if e.Verified {
				email = e.Email
				break
			}
		}
	}

	if email == "" {
		return nil, ErrEmailNotFound
	}

	return &UserInfo{
		Email: email,
		Name:  userName,
	}, nil
}

// GetUserEmail retrieves the user's email from GitHub (deprecated, use GetUserInfo)
func (p *GitHubProvider) GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	userInfo, err := p.GetUserInfo(ctx, token)
	if err != nil {
		return "", err
	}
	return userInfo.Email, nil
}
