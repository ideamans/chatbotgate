package email

import (
	"fmt"
	"time"

	hermes "github.com/ideamans/hermes"
	"github.com/ideamans/chatbotgate/pkg/i18n"
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
func (t *EmailTemplate) GenerateLoginEmail(loginURL string, validMinutes int, lang i18n.Language, translator *i18n.Translator) (htmlBody, textBody string, err error) {
	// Translation helper
	tr := func(key string, args ...interface{}) string {
		text := translator.T(lang, key)
		if len(args) > 0 {
			return fmt.Sprintf(text, args...)
		}
		return text
	}
	// Get current year for copyright
	currentYear := time.Now().Year()

	h := hermes.Hermes{
		Product: hermes.Product{
			Name:          t.serviceName,
			Link:          t.baseURL,
			Logo:          t.logoURL,
			LogoWidth:     t.logoWidth,
			Icon:          t.iconURL,
			Copyright:     fmt.Sprintf("© %d %s", currentYear, t.serviceName),
			HideSignature: true, // Hide signature line
			HideGreeting:  true, // Hide default greeting
			TroubleText:   tr("email.login.trouble", tr("email.login.button")),
		},
	}

	email := hermes.Email{
		Body: hermes.Body{
			Name: "", // No personalization
			Intros: []string{
				tr("email.login.greeting"),
				tr("email.login.intro1", t.serviceName),
				tr("email.login.intro2", validMinutes),
			},
			Actions: []hermes.Action{
				{
					Instructions: tr("email.login.instructions"),
					Button: hermes.Button{
						Color: "#3B82F6", // Primary blue color
						Text:  tr("email.login.button"),
						Link:  loginURL,
					},
				},
			},
			Outros: []string{
				tr("email.login.outro"),
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
