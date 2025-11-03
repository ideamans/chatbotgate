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
	config *oauth2.Config
}

// NewMicrosoftProvider creates a new Microsoft OAuth2 provider
func NewMicrosoftProvider(clientID, clientSecret, redirectURL string, scopes []string, resetScopes bool) *MicrosoftProvider {
	// Default scopes
	defaultScopes := []string{
		"openid",
		"profile",
		"email",
		"User.Read",
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

	return &MicrosoftProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       finalScopes,
			Endpoint:     microsoft.AzureADEndpoint("common"),
		},
	}
}

// Name returns the provider name
func (p *MicrosoftProvider) Name() string {
	return "microsoft"
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
	defer resp.Body.Close()

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

	return &UserInfo{
		Email: email,
		Name:  apiUserInfo.DisplayName,
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
