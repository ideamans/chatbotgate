package email

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/i18n"
	"github.com/ideamans/chatbotgate/pkg/kvs"
)

// MockAuthzChecker is a mock authorization checker
type MockAuthzChecker struct {
	allowed        bool
	requiresEmail  bool
}

func (m *MockAuthzChecker) RequiresEmail() bool {
	return m.requiresEmail
}

func (m *MockAuthzChecker) IsAllowed(email string) bool {
	return m.allowed
}

// createTestTokenKVS creates a memory-based token KVS for testing
func createTestTokenKVS() kvs.Store {
	kvsStore, _ := kvs.NewMemoryStore("token:", kvs.MemoryConfig{
		CleanupInterval: 1 * time.Minute,
	})
	return kvsStore
}

// createTestRateLimitKVS creates a memory-based rate limit KVS for testing
func createTestRateLimitKVS() kvs.Store {
	kvsStore, _ := kvs.NewMemoryStore("ratelimit:", kvs.MemoryConfig{
		CleanupInterval: 1 * time.Minute,
	})
	return kvsStore
}

// testServiceConfig returns a default ServiceConfig for testing
func testServiceConfig() config.ServiceConfig {
	return config.ServiceConfig{
		Name:      "Test Service",
		LogoURL:   "https://example.com/logo.svg",
		LogoWidth: "200px",
		IconURL:   "https://example.com/icon.svg",
	}
}

// testTranslator returns a default Translator for testing
func testTranslator() *i18n.Translator {
	return i18n.NewTranslator()
}

func TestNewHandler(t *testing.T) {
	cfg := config.EmailAuthConfig{
		Enabled:    true,
		SenderType: "smtp",
		SMTP: config.SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "noreply@example.com",
		},
		Token: config.EmailTokenConfig{
			Expire: "15m",
		},
	}

	authzChecker := &MockAuthzChecker{allowed: true}

	handler, err := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, testTranslator(), "test-secret", createTestTokenKVS(), createTestRateLimitKVS())
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	if handler == nil {
		t.Error("NewHandler() returned nil")
	}
}

func TestNewHandler_InvalidSenderType(t *testing.T) {
	cfg := config.EmailAuthConfig{
		Enabled:    true,
		SenderType: "invalid",
	}

	authzChecker := &MockAuthzChecker{allowed: true}

	_, err := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, testTranslator(), "test-secret", createTestTokenKVS(), createTestRateLimitKVS())
	if err == nil {
		t.Error("NewHandler() should return error for invalid sender type")
	}
}

func TestHandler_SendLoginLink(t *testing.T) {
	cfg := config.EmailAuthConfig{
		Enabled:    true,
		SenderType: "smtp",
		SMTP: config.SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "noreply@example.com",
		},
		Token: config.EmailTokenConfig{
			Expire: "15m",
		},
	}

	authzChecker := &MockAuthzChecker{allowed: true}
	mockSender := &MockSender{}

	handler, _ := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, testTranslator(), "test-secret", createTestTokenKVS(), createTestRateLimitKVS())
	handler.sender = mockSender // Replace with mock

	email := "user@example.com"

	err := handler.SendLoginLink(email, i18n.English)
	if err != nil {
		t.Fatalf("SendLoginLink() error = %v", err)
	}

	// Verify HTML email was sent
	if len(mockSender.HTMLCalls) != 1 {
		t.Fatalf("expected 1 HTML email sent, got %d", len(mockSender.HTMLCalls))
	}

	call := mockSender.HTMLCalls[0]
	if call.To != email {
		t.Errorf("email sent to %s, want %s", call.To, email)
	}

	if !strings.Contains(call.Subject, "Test Service") {
		t.Error("subject should contain service name")
	}

	if !strings.Contains(call.HTMLBody, "http://localhost:4180/_auth/email/verify?token=") {
		t.Error("HTML body should contain verification link")
	}

	if !strings.Contains(call.TextBody, "http://localhost:4180/_auth/email/verify?token=") {
		t.Error("text body should contain verification link")
	}
}

func TestHandler_SendLoginLink_NotAuthorized(t *testing.T) {
	cfg := config.EmailAuthConfig{
		Enabled:    true,
		SenderType: "smtp",
		SMTP: config.SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "noreply@example.com",
		},
	}

	authzChecker := &MockAuthzChecker{allowed: false}
	mockSender := &MockSender{}

	handler, _ := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, testTranslator(), "test-secret", createTestTokenKVS(), createTestRateLimitKVS())
	handler.sender = mockSender

	email := "unauthorized@example.com"

	err := handler.SendLoginLink(email, i18n.English)
	if err == nil {
		t.Error("SendLoginLink() should return error for unauthorized email")
	}

	// No email should be sent
	if len(mockSender.HTMLCalls) != 0 {
		t.Error("no HTML email should be sent for unauthorized user")
	}
}

func TestHandler_SendLoginLink_RateLimit(t *testing.T) {
	cfg := config.EmailAuthConfig{
		Enabled:    true,
		SenderType: "smtp",
		SMTP: config.SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "noreply@example.com",
		},
	}

	authzChecker := &MockAuthzChecker{allowed: true}
	mockSender := &MockSender{}

	handler, _ := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, testTranslator(), "test-secret", createTestTokenKVS(), createTestRateLimitKVS())
	handler.sender = mockSender

	email := "user@example.com"

	// Send 3 emails (should succeed)
	for i := 0; i < 3; i++ {
		if err := handler.SendLoginLink(email, i18n.English); err != nil {
			t.Fatalf("request %d should succeed", i+1)
		}
	}

	// 4th should be rate limited
	err := handler.SendLoginLink(email, i18n.English)
	if err == nil {
		t.Error("4th request should be rate limited")
	}
}

func TestHandler_SendLoginLink_SendFails(t *testing.T) {
	cfg := config.EmailAuthConfig{
		Enabled:    true,
		SenderType: "smtp",
		SMTP: config.SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "noreply@example.com",
		},
	}

	authzChecker := &MockAuthzChecker{allowed: true}
	mockSender := &MockSender{
		SendHTMLFunc: func(to, subject, htmlBody, textBody string) error {
			return errors.New("send failed")
		},
	}

	handler, _ := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, testTranslator(), "test-secret", createTestTokenKVS(), createTestRateLimitKVS())
	handler.sender = mockSender

	email := "user@example.com"

	err := handler.SendLoginLink(email, i18n.English)
	if err == nil {
		t.Error("SendLoginLink() should return error when send fails")
	}
}

func TestHandler_VerifyToken(t *testing.T) {
	cfg := config.EmailAuthConfig{
		Enabled:    true,
		SenderType: "smtp",
		SMTP: config.SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "noreply@example.com",
		},
	}

	authzChecker := &MockAuthzChecker{allowed: true}
	mockSender := &MockSender{}

	handler, _ := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, testTranslator(), "test-secret", createTestTokenKVS(), createTestRateLimitKVS())
	handler.sender = mockSender

	email := "user@example.com"

	// Send login link
	handler.SendLoginLink(email, i18n.English)

	// Extract token from HTML email body
	call := mockSender.HTMLCalls[0]
	// Parse token from HTML body (simplified)
	body := call.HTMLBody
	tokenStart := strings.Index(body, "token=") + 6
	tokenEnd := strings.IndexAny(body[tokenStart:], "\"& ")
	if tokenEnd == -1 {
		tokenEnd = len(body) - tokenStart
	}
	token := body[tokenStart : tokenStart+tokenEnd]

	// Verify token
	verifiedEmail, err := handler.VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}

	if verifiedEmail != email {
		t.Errorf("VerifyToken() email = %s, want %s", verifiedEmail, email)
	}

	// Second verification should fail (one-time use)
	_, err = handler.VerifyToken(token)
	if err == nil {
		t.Error("second VerifyToken() should fail")
	}
}
