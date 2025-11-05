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
func NewGoogleProvider(clientID, clientSecret, redirectURL string, scopes []string, resetScopes bool) *GoogleProvider {
	// Default scopes (used only when scopes is empty)
	defaultScopes := []string{
		"openid",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}

	// Use default scopes only when no scopes are provided
	var finalScopes []string
	if len(scopes) == 0 {
		finalScopes = defaultScopes
	} else {
		finalScopes = scopes
	}

	return &GoogleProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       finalScopes,
			Endpoint:     google.Endpoint,
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

// GetUserInfo retrieves the user's information from Google
func (p *GoogleProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := p.config.Client(ctx, token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var apiUserInfo struct {
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiUserInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	if apiUserInfo.Email == "" {
		return nil, ErrEmailNotFound
	}

	// Set common fields for forwarding
	extra := make(map[string]any)
	extra["_email"] = apiUserInfo.Email
	extra["_username"] = apiUserInfo.Name
	if apiUserInfo.Picture != "" {
		extra["_avatar_url"] = apiUserInfo.Picture
	} else {
		extra["_avatar_url"] = ""
	}

	return &UserInfo{
		Email: apiUserInfo.Email,
		Name:  apiUserInfo.Name,
		Extra: extra,
	}, nil
}

// GetUserEmail retrieves the user's email from Google (deprecated, use GetUserInfo)
func (p *GoogleProvider) GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	userInfo, err := p.GetUserInfo(ctx, token)
	if err != nil {
		return "", err
	}
	return userInfo.Email, nil
}
