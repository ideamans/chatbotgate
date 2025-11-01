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
func NewMicrosoftProvider(clientID, clientSecret, redirectURL string) *MicrosoftProvider {
	return &MicrosoftProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"openid",
				"profile",
				"email",
				"User.Read",
			},
			Endpoint: microsoft.AzureADEndpoint("common"),
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

// GetUserEmail retrieves the user's email from Microsoft Graph API
func (p *MicrosoftProvider) GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	client := p.config.Client(ctx, token)

	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var userInfo struct {
		Mail                string `json:"mail"`
		UserPrincipalName   string `json:"userPrincipalName"`
		PreferredUsername   string `json:"preferredUsername"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", fmt.Errorf("failed to decode user info: %w", err)
	}

	// Try mail field first (most reliable)
	if userInfo.Mail != "" {
		return userInfo.Mail, nil
	}

	// Fallback to userPrincipalName
	if userInfo.UserPrincipalName != "" {
		return userInfo.UserPrincipalName, nil
	}

	// Last resort: preferredUsername
	if userInfo.PreferredUsername != "" {
		return userInfo.PreferredUsername, nil
	}

	return "", ErrEmailNotFound
}
