package oauth2

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

// MicrosoftProvider is the OAuth2 provider for Microsoft (Azure AD)
type MicrosoftProvider struct {
	id     string
	config *oauth2.Config
}

// NewMicrosoftProvider creates a new Microsoft OAuth2 provider
func NewMicrosoftProvider(id, clientID, clientSecret, redirectURL string, scopes []string, resetScopes bool) *MicrosoftProvider {
	// Default scopes (used only when scopes is empty)
	defaultScopes := []string{
		"openid",
		"profile",
		"email",
		"User.Read",
	}

	// Use default scopes only when no scopes are provided
	var finalScopes []string
	if len(scopes) == 0 {
		finalScopes = defaultScopes
	} else {
		finalScopes = scopes
	}

	return &MicrosoftProvider{
		id: id,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       finalScopes,
			Endpoint:     microsoft.AzureADEndpoint("common"),
		},
	}
}

// Name returns the provider name (ID)
func (p *MicrosoftProvider) Name() string {
	return p.id
}

// Config returns the OAuth2 config
func (p *MicrosoftProvider) Config() *oauth2.Config {
	return p.config
}

// GetUserInfo retrieves the user's information from Microsoft Graph API
func (p *MicrosoftProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := p.config.Client(ctx, token)

	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var apiUserInfo struct {
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"`
		PreferredUsername string `json:"preferredUsername"`
		DisplayName       string `json:"displayName"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiUserInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	var email string
	// Try mail field first (most reliable)
	if apiUserInfo.Mail != "" {
		email = apiUserInfo.Mail
	} else if apiUserInfo.UserPrincipalName != "" {
		// Fallback to userPrincipalName
		email = apiUserInfo.UserPrincipalName
	} else if apiUserInfo.PreferredUsername != "" {
		// Last resort: preferredUsername
		email = apiUserInfo.PreferredUsername
	}

	if email == "" {
		return nil, ErrEmailNotFound
	}

	// Note: Microsoft Graph /me/photo endpoint returns binary data, not a URL
	// For now, we don't support avatar URL for Microsoft provider
	// Future enhancement: proxy the photo through our own endpoint

	// Set common fields for forwarding
	extra := make(map[string]any)
	extra["_email"] = email
	extra["_username"] = apiUserInfo.DisplayName
	extra["_avatar_url"] = "" // Microsoft doesn't provide a direct URL

	return &UserInfo{
		Email: email,
		Name:  apiUserInfo.DisplayName,
		Extra: extra,
	}, nil
}

// GetUserEmail retrieves the user's email from Microsoft Graph API (deprecated, use GetUserInfo)
func (p *MicrosoftProvider) GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	userInfo, err := p.GetUserInfo(ctx, token)
	if err != nil {
		return "", err
	}
	return userInfo.Email, nil
}
