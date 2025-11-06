package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
)

// Sender is an interface for sending emails
type Sender interface {
	Send(to, subject, body string) error
	SendHTML(to, subject, htmlBody, textBody string) error
}

// SMTPSender sends emails via SMTP
type SMTPSender struct {
	config config.SMTPConfig
}

// NewSMTPSender creates a new SMTP email sender
func NewSMTPSender(cfg config.SMTPConfig) *SMTPSender {
	return &SMTPSender{config: cfg}
}

// Send sends an email via SMTP
func (s *SMTPSender) Send(to, subject, body string) error {
	from := s.config.From
	if s.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.From)
	}

	// Compose message
	message := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", from, to, subject, body)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Setup authentication
	var auth smtp.Auth
	if s.config.Username != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}

	// Send based on TLS/STARTTLS configuration
	if s.config.TLS {
		// Use TLS from the start
		return s.sendWithTLS(addr, auth, from, []string{to}, []byte(message))
	}

	// Use STARTTLS or plain connection
	return smtp.SendMail(addr, auth, s.config.From, []string{to}, []byte(message))
}

// sendWithTLS sends email using TLS from the start
func (s *SMTPSender) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// Create TLS configuration
	tlsConfig := &tls.Config{
		ServerName: s.config.Host,
	}

	// Connect with TLS
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Authenticate
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to create data writer: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}

// SendHTML sends an HTML email with plain text fallback via SMTP
func (s *SMTPSender) SendHTML(to, subject, htmlBody, textBody string) error {
	from := s.config.From
	fromHeader := s.config.From
	if s.config.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.From)
	}

	// Build multipart message
	var builder strings.Builder
	boundary := "----=_Part_MultiOAuth2Proxy"

	// Headers
	builder.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", to))
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	builder.WriteString("\r\n")

	// Plain text part
	builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	builder.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(textBody)
	builder.WriteString("\r\n\r\n")

	// HTML part
	builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	builder.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	builder.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(htmlBody)
	builder.WriteString("\r\n\r\n")

	// End boundary
	builder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	message := builder.String()
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Setup authentication
	var auth smtp.Auth
	if s.config.Username != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}

	// Send based on TLS/STARTTLS configuration
	if s.config.TLS {
		// Use TLS from the start
		return s.sendWithTLS(addr, auth, from, []string{to}, []byte(message))
	}

	// Use STARTTLS or plain connection
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(message))
}
