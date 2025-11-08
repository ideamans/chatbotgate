package email

import (
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
)

func TestNewSMTPSender(t *testing.T) {
	cfg := config.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
	}

	sender := NewSMTPSender(cfg, "noreply@example.com", "Test Service")
	if sender == nil {
		t.Fatal("NewSMTPSender() returned nil")
	}

	if sender.config.Host != "smtp.example.com" {
		t.Errorf("config.Host = %s, want smtp.example.com", sender.config.Host)
	}

	if sender.from != "noreply@example.com" {
		t.Errorf("from = %s, want noreply@example.com", sender.from)
	}

	if sender.fromName != "Test Service" {
		t.Errorf("fromName = %s, want Test Service", sender.fromName)
	}
}

func TestNewSMTPSender_Override(t *testing.T) {
	// Test that SMTP-specific config overrides parent config
	cfg := config.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "smtp-specific@example.com",
		FromName: "SMTP Service",
	}

	sender := NewSMTPSender(cfg, "parent@example.com", "Parent Service")
	if sender == nil {
		t.Fatal("NewSMTPSender() returned nil")
	}

	// Should use SMTP-specific config, not parent
	if sender.from != "smtp-specific@example.com" {
		t.Errorf("from = %s, want smtp-specific@example.com", sender.from)
	}

	if sender.fromName != "SMTP Service" {
		t.Errorf("fromName = %s, want SMTP Service", sender.fromName)
	}
}

func TestNewSendGridSender(t *testing.T) {
	cfg := config.SendGridConfig{
		APIKey: "test-api-key",
	}

	sender := NewSendGridSender(cfg, "noreply@example.com", "Test Service")
	if sender == nil {
		t.Fatal("NewSendGridSender() returned nil")
	}

	if sender.config.APIKey != "test-api-key" {
		t.Errorf("config.APIKey = %s, want test-api-key", sender.config.APIKey)
	}

	if sender.from != "noreply@example.com" {
		t.Errorf("from = %s, want noreply@example.com", sender.from)
	}

	if sender.fromName != "Test Service" {
		t.Errorf("fromName = %s, want Test Service", sender.fromName)
	}
}

func TestNewSendGridSender_Override(t *testing.T) {
	// Test that SendGrid-specific config overrides parent config
	cfg := config.SendGridConfig{
		APIKey:   "test-api-key",
		From:     "sendgrid-specific@example.com",
		FromName: "SendGrid Service",
	}

	sender := NewSendGridSender(cfg, "parent@example.com", "Parent Service")
	if sender == nil {
		t.Fatal("NewSendGridSender() returned nil")
	}

	// Should use SendGrid-specific config, not parent
	if sender.from != "sendgrid-specific@example.com" {
		t.Errorf("from = %s, want sendgrid-specific@example.com", sender.from)
	}

	if sender.fromName != "SendGrid Service" {
		t.Errorf("fromName = %s, want SendGrid Service", sender.fromName)
	}
}

func TestMockSender(t *testing.T) {
	mock := &MockSender{}

	err := mock.Send("user@example.com", "Test Subject", "Test Body")
	if err != nil {
		t.Fatalf("MockSender.Send() error = %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.Calls))
	}

	call := mock.Calls[0]
	if call.To != "user@example.com" {
		t.Errorf("To = %s, want user@example.com", call.To)
	}

	if call.Subject != "Test Subject" {
		t.Errorf("Subject = %s, want Test Subject", call.Subject)
	}

	if call.Body != "Test Body" {
		t.Errorf("Body = %s, want Test Body", call.Body)
	}
}

// Note: Actual SMTP and SendGrid tests would require either:
// 1. A test SMTP server / SendGrid sandbox
// 2. Integration tests (not unit tests)
// For now, we test the constructors and use mocks in higher-level tests
