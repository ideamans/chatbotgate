package email

import (
	"fmt"
	"strings"
	"time"

	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
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
func (t *EmailTemplate) GenerateLoginEmail(loginURL, otp string, validMinutes int, lang i18n.Language, translator *i18n.Translator) (htmlBody, textBody string, err error) {
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

	// Use a placeholder for OTP that we'll replace later
	otpPlaceholder := "{{OTP_CODE_PLACEHOLDER}}"

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
				otpPlaceholder,
				tr("email.login.outro"),
			},
		},
	}

	// Generate HTML body
	htmlBody, err = h.GenerateHTML(email)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate HTML email: %w", err)
	}

	// Format OTP: split into 4-digit groups
	// Note: No whitespace between spans to prevent unwanted spaces when copying
	otpHTML := fmt.Sprintf(
		`<div style="text-align: center; margin: 24px 0;"><p style="color: #6b7280; font-size: 14px; margin-bottom: 12px;">%s</p><div style="font-family: 'Courier New', monospace; font-size: 18px; font-weight: 600; letter-spacing: 0.05em; background-color: #f3f4f6; border: 2px solid #d1d5db; border-radius: 8px; padding: 16px; display: inline-block;"><span style="margin: 0 4px;">%s</span><span style="margin: 0 4px;">%s</span><span style="margin: 0 4px;">%s</span></div></div>`,
		tr("email.login.otp_label"),
		otp[0:4], otp[4:8], otp[8:12],
	)

	// Replace placeholder with actual OTP HTML
	htmlBody = strings.ReplaceAll(htmlBody, otpPlaceholder, otpHTML)

	// Generate plain text body
	textBody, err = h.GeneratePlainText(email)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate plain text email: %w", err)
	}

	// Format OTP for plain text: just the code with spaces
	otpPlainText := fmt.Sprintf("%s\n\n%s %s %s\n",
		tr("email.login.otp_label"),
		otp[0:4], otp[4:8], otp[8:12],
	)

	// Replace placeholder in plain text too
	textBody = strings.ReplaceAll(textBody, otpPlaceholder, otpPlainText)

	return htmlBody, textBody, nil
}
