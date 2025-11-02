package email

import (
	"testing"

	"github.com/ideamans/chatbotgate/pkg/config"
)

func TestNewSMTPSender(t *testing.T) {
	cfg := config.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "noreply@example.com",
		FromName: "Test Service",
	}

	sender := NewSMTPSender(cfg)
	if sender == nil {
		t.Error("NewSMTPSender() returned nil")
	}

	if sender.config.Host != "smtp.example.com" {
		t.Errorf("config.Host = %s, want smtp.example.com", sender.config.Host)
	}
}

func TestNewSendGridSender(t *testing.T) {
	cfg := config.SendGridConfig{
		APIKey:   "test-api-key",
		From:     "noreply@example.com",
		FromName: "Test Service",
	}

	sender := NewSendGridSender(cfg)
	if sender == nil {
		t.Error("NewSendGridSender() returned nil")
	}

	if sender.config.APIKey != "test-api-key" {
		t.Errorf("config.APIKey = %s, want test-api-key", sender.config.APIKey)
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
