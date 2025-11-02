package email

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
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

// testServiceConfig returns a default ServiceConfig for testing
func testServiceConfig() config.ServiceConfig {
	return config.ServiceConfig{
		Name:      "Test Service",
		LogoURL:   "https://example.com/logo.svg",
		LogoWidth: "200px",
		IconURL:   "https://example.com/icon.svg",
	}
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

	handler, err := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, "test-secret")
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

	_, err := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, "test-secret")
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

	handler, _ := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, "test-secret")
	handler.sender = mockSender // Replace with mock

	email := "user@example.com"

	err := handler.SendLoginLink(email)
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

	handler, _ := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, "test-secret")
	handler.sender = mockSender

	email := "unauthorized@example.com"

	err := handler.SendLoginLink(email)
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

	handler, _ := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, "test-secret")
	handler.sender = mockSender

	email := "user@example.com"

	// Send 3 emails (should succeed)
	for i := 0; i < 3; i++ {
		if err := handler.SendLoginLink(email); err != nil {
			t.Fatalf("request %d should succeed", i+1)
		}
	}

	// 4th should be rate limited
	err := handler.SendLoginLink(email)
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

	handler, _ := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, "test-secret")
	handler.sender = mockSender

	email := "user@example.com"

	err := handler.SendLoginLink(email)
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

	handler, _ := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, "test-secret")
	handler.sender = mockSender

	email := "user@example.com"

	// Send login link
	handler.SendLoginLink(email)

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

func TestHandler_SendLoginLink_OTPFile(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "otp-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	otpFile := filepath.Join(tempDir, "otp.json")

	cfg := config.EmailAuthConfig{
		Enabled:       true,
		SenderType:    "smtp", // Required but won't be used
		OTPOutputFile: otpFile,
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

	handler, _ := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, "test-secret")
	handler.sender = mockSender

	email := "user@example.com"

	// Send login link
	err = handler.SendLoginLink(email)
	if err != nil {
		t.Fatalf("SendLoginLink() error = %v", err)
	}

	// Verify no email was sent (should use OTP file instead)
	if len(mockSender.HTMLCalls) != 0 {
		t.Error("no HTML email should be sent when OTP file is configured")
	}

	// Read OTP file
	data, err := os.ReadFile(otpFile)
	if err != nil {
		t.Fatalf("Failed to read OTP file: %v", err)
	}

	// Parse JSON record
	var record OTPRecord
	if err := json.Unmarshal(data[:len(data)-1], &record); err != nil { // Remove trailing newline
		t.Fatalf("Failed to unmarshal OTP record: %v", err)
	}

	// Verify record
	if record.Email != email {
		t.Errorf("Email = %s, want %s", record.Email, email)
	}

	if record.Token == "" {
		t.Error("Token should not be empty")
	}

	expectedLoginURL := "http://localhost:4180/_auth/email/verify?token=" + record.Token
	if record.LoginURL != expectedLoginURL {
		t.Errorf("LoginURL = %s, want %s", record.LoginURL, expectedLoginURL)
	}

	if record.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should not be zero")
	}

	// Verify token can be used for authentication
	verifiedEmail, err := handler.VerifyToken(record.Token)
	if err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}

	if verifiedEmail != email {
		t.Errorf("VerifyToken() email = %s, want %s", verifiedEmail, email)
	}
}

func TestHandler_SendLoginLink_OTPFile_MultipleUsers(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "otp-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	otpFile := filepath.Join(tempDir, "otp.json")

	cfg := config.EmailAuthConfig{
		Enabled:       true,
		SenderType:    "smtp",
		OTPOutputFile: otpFile,
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
	handler, _ := NewHandler(cfg, testServiceConfig(), "http://localhost:4180", "/_auth", authzChecker, "test-secret")

	// Send login links for multiple users
	users := []string{"user1@example.com", "user2@example.com", "user3@example.com"}
	for _, email := range users {
		if err := handler.SendLoginLink(email); err != nil {
			t.Fatalf("SendLoginLink() error = %v", err)
		}
	}

	// Read OTP file
	data, err := os.ReadFile(otpFile)
	if err != nil {
		t.Fatalf("Failed to read OTP file: %v", err)
	}

	// Count lines (JSON Lines format)
	lines := 0
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines++
			var record OTPRecord
			if err := json.Unmarshal(data[start:i], &record); err != nil {
				t.Fatalf("Failed to unmarshal line %d: %v", lines, err)
			}
			start = i + 1
		}
	}

	if lines != len(users) {
		t.Errorf("Expected %d lines, got %d", len(users), lines)
	}
}
