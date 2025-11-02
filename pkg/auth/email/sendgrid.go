package email

import (
	"fmt"

	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGridSender sends emails via SendGrid API
type SendGridSender struct {
	config config.SendGridConfig
	client *sendgrid.Client
}

// NewSendGridSender creates a new SendGrid email sender
func NewSendGridSender(cfg config.SendGridConfig) *SendGridSender {
	client := sendgrid.NewSendClient(cfg.APIKey)
	return &SendGridSender{
		config: cfg,
		client: client,
	}
}

// Send sends an email via SendGrid API
func (s *SendGridSender) Send(to, subject, body string) error {
	from := mail.NewEmail(s.config.FromName, s.config.From)
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
	from := mail.NewEmail(s.config.FromName, s.config.From)
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
