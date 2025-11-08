package email

import (
	"fmt"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGridSender sends emails via SendGrid API
type SendGridSender struct {
	config   config.SendGridConfig
	client   *sendgrid.Client
	from     string // Email address
	fromName string // Display name
}

// NewSendGridSender creates a new SendGrid email sender
func NewSendGridSender(cfg config.SendGridConfig, parentEmail, parentName string) *SendGridSender {
	client := sendgrid.NewSendClient(cfg.APIKey)

	// Set custom endpoint URL if configured
	if cfg.EndpointURL != "" {
		client.BaseURL = cfg.EndpointURL
	}

	email, name := cfg.GetFromAddress(parentEmail, parentName)

	return &SendGridSender{
		config:   cfg,
		client:   client,
		from:     email,
		fromName: name,
	}
}

// Send sends an email via SendGrid API
func (s *SendGridSender) Send(to, subject, body string) error {
	from := mail.NewEmail(s.fromName, s.from)
	toEmail := mail.NewEmail("", to)
	message := mail.NewSingleEmail(from, subject, toEmail, body, body)

	response, err := s.client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send email via SendGrid: %w", err)
	}

	// Check response status
	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid returned error status: %d %s", response.StatusCode, response.Body)
	}

	return nil
}

// SendHTML sends an HTML email with plain text fallback via SendGrid API
func (s *SendGridSender) SendHTML(to, subject, htmlBody, textBody string) error {
	from := mail.NewEmail(s.fromName, s.from)
	toEmail := mail.NewEmail("", to)
	message := mail.NewSingleEmail(from, subject, toEmail, textBody, htmlBody)

	response, err := s.client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send HTML email via SendGrid: %w", err)
	}

	// Check response status
	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid returned error status: %d %s", response.StatusCode, response.Body)
	}

	return nil
}
