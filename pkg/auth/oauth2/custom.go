package oauth2

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

// CustomProvider is a generic OAuth2 provider for custom/OIDC-compatible servers
type CustomProvider struct {
	name               string
	config             *oauth2.Config
	userInfoURL        string
	insecureSkipVerify bool
}

// NewCustomProvider creates a new custom OAuth2 provider
func NewCustomProvider(
	name string,
	clientID string,
	clientSecret string,
	redirectURL string,
	authURL string,
	tokenURL string,
	userInfoURL string,
	insecureSkipVerify bool,
) *CustomProvider {
	return &CustomProvider{
		name: name,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"openid",
				"email",
				"profile",
			},
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
		},
		userInfoURL:        userInfoURL,
		insecureSkipVerify: insecureSkipVerify,
	}
}

// Name returns the provider name
func (p *CustomProvider) Name() string {
	return p.name
}

// Config returns the OAuth2 config
func (p *CustomProvider) Config() *oauth2.Config {
	return p.config
}

// GetUserEmail retrieves the user's email from the custom provider
func (p *CustomProvider) GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	// Create HTTP client with custom transport if insecure skip verify is enabled
	var httpClient *http.Client
	if p.insecureSkipVerify {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient = &http.Client{Transport: transport}
		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	}

	client := p.config.Client(ctx, token)

	resp, err := client.Get(p.userInfoURL)
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var userInfo struct {
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"email_verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", fmt.Errorf("failed to decode user info: %w", err)
	}

	if userInfo.Email == "" {
		return "", ErrEmailNotFound
	}

	return userInfo.Email, nil
}
