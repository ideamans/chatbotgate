package email

import (
	"fmt"

	hermes "github.com/ideamans/hermes"
)

// EmailTemplate generates HTML emails using Hermes
type EmailTemplate struct {
	serviceName string
	logoURL     string
	logoWidth   string
	iconURL     string
	baseURL     string
}

// NewEmailTemplate creates a new email template generator
func NewEmailTemplate(serviceName, logoURL, logoWidth, iconURL, baseURL string) *EmailTemplate {
	return &EmailTemplate{
		serviceName: serviceName,
		logoURL:     logoURL,
		logoWidth:   logoWidth,
		iconURL:     iconURL,
		baseURL:     baseURL,
	}
}

// GenerateLoginEmail generates HTML and plain text for login link email
func (t *EmailTemplate) GenerateLoginEmail(loginURL string, validMinutes int) (htmlBody, textBody string, err error) {
	h := hermes.Hermes{
		Product: hermes.Product{
			Name:      t.serviceName,
			Link:      t.baseURL,
			Logo:      t.logoURL,
			LogoWidth: t.logoWidth,
			Icon:      t.iconURL,
			Copyright: "", // No copyright footer
		},
	}

	email := hermes.Email{
		Body: hermes.Body{
			Name: "", // No personalization
			Intros: []string{
				fmt.Sprintf("Click the button below to log in to %s.", t.serviceName),
				fmt.Sprintf("This link is valid for %d minutes.", validMinutes),
			},
			Actions: []hermes.Action{
				{
					Instructions: "Please click the button below to complete your login:",
					Button: hermes.Button{
						Color: "#3B82F6", // Primary blue color
						Text:  "Log In",
						Link:  loginURL,
					},
				},
			},
			Outros: []string{
				"If you did not request this email, please ignore it.",
			},
		},
	}

	// Generate HTML body
	htmlBody, err = h.GenerateHTML(email)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate HTML email: %w", err)
	}

	// Generate plain text body
	textBody, err = h.GeneratePlainText(email)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate plain text email: %w", err)
	}

	return htmlBody, textBody, nil
}
