package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/auth/email"
	"github.com/ideamans/chatbotgate/pkg/middleware/authz"
	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// mockEmailSender is a mock implementation of email.Sender for testing
type mockEmailSender struct {
	sendError     error
	sendHTMLError error
	sentEmails    []sentEmail
}

type sentEmail struct {
	to       string
	subject  string
	htmlBody string
	textBody string
}

func (m *mockEmailSender) Send(to, subject, body string) error {
	m.sentEmails = append(m.sentEmails, sentEmail{to: to, subject: subject, textBody: body})
	return m.sendError
}

func (m *mockEmailSender) SendHTML(to, subject, htmlBody, textBody string) error {
	m.sentEmails = append(m.sentEmails, sentEmail{to: to, subject: subject, htmlBody: htmlBody, textBody: textBody})
	return m.sendHTMLError
}

// createEmailHandler creates a real email handler with mock sender for testing
func createEmailHandler(t *testing.T, mockSender *mockEmailSender, authzConfig config.AccessControlConfig, limitPerMinute int) *email.Handler {
	tokenKVS, _ := kvs.NewMemoryStore("email-tokens-"+t.Name(), kvs.MemoryConfig{})
	t.Cleanup(func() { _ = tokenKVS.Close() })
	quotaKVS, _ := kvs.NewMemoryStore("email-quota-"+t.Name(), kvs.MemoryConfig{})
	t.Cleanup(func() { _ = quotaKVS.Close() })

	cfg := config.EmailAuthConfig{
		Enabled:        true,
		SenderType:     "smtp", // Will be replaced with mock
		From:           "noreply@test.com",
		FromName:       "Test Service",
		LimitPerMinute: limitPerMinute,
		Token: config.EmailTokenConfig{
			Expire: "15m",
		},
		SMTP: config.SMTPConfig{
			Host: "localhost",
			Port: 25,
		},
	}

	serviceCfg := config.ServiceConfig{
		Name: "Test Service",
	}

	authzChecker := authz.NewEmailChecker(authzConfig)

	handler, err := email.NewHandler(
		cfg,
		serviceCfg,
		"http://localhost:4180",
		"/_auth",
		authzChecker,
		i18n.NewTranslator(),
		"test-secret-key-32-bytes-long!!",
		tokenKVS,
		quotaKVS,
	)
	if err != nil {
		t.Fatalf("Failed to create email handler: %v", err)
	}

	// Inject mock sender
	if mockSender != nil {
		handler.SetSender(mockSender)
	}

	return handler
}

// Helper to extract OTP from sent email
func extractOTPFromEmail(sentEmail sentEmail) string {
	// OTP is in the text body in format "XXXX XXXX XXXX" (12 chars with spaces)
	// Or as a plain 12-character alphanumeric string
	lines := strings.Split(sentEmail.textBody, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this line contains "code" or "enter" (near the OTP)
		if strings.Contains(strings.ToLower(trimmed), "code") || strings.Contains(strings.ToLower(trimmed), "enter") {
			// OTP should be on next line
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				// Remove spaces and check if it's 12 alphanumeric chars
				otp := strings.ReplaceAll(nextLine, " ", "")
				if len(otp) == 12 && isAlphanumeric(otp) {
					return otp
				}
			}
			// Also check if OTP is on the same line after ":"
			if strings.Contains(trimmed, ":") {
				parts := strings.Split(trimmed, ":")
				if len(parts) > 1 {
					otp := strings.TrimSpace(parts[len(parts)-1])
					otp = strings.ReplaceAll(otp, " ", "")
					if len(otp) == 12 && isAlphanumeric(otp) {
						return otp
					}
				}
			}
		}

		// Also try to find any line that looks like an OTP (4 chars, space, 4 chars, space, 4 chars)
		words := strings.Fields(trimmed)
		if len(words) == 3 && len(words[0]) == 4 && len(words[1]) == 4 && len(words[2]) == 4 {
			otp := words[0] + words[1] + words[2]
			if isAlphanumeric(otp) {
				return otp
			}
		}
	}
	return ""
}

// Helper to check if string is alphanumeric
func isAlphanumeric(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// Helper to extract token from sent email
func extractTokenFromEmail(sentEmail sentEmail) string {
	// Token is in the URL - extract it
	// Format: http://localhost:4180/_auth/email/verify?token=XXXX
	lines := strings.Split(sentEmail.textBody, "\n")
	for _, line := range lines {
		if strings.Contains(line, "verify?token=") {
			parts := strings.Split(line, "token=")
			if len(parts) >= 2 {
				token := strings.TrimSpace(parts[1])
				// Remove any trailing punctuation
				token = strings.Trim(token, ".,;:!?")
				return token
			}
		}
	}
	return ""
}

// TestHandleEmailSend tests the email send handler
func TestHandleEmailSend(t *testing.T) {
	tests := []struct {
		name            string
		formData        url.Values
		setupMock       func(*mockEmailSender)
		authzConfig     config.AccessControlConfig
		limitPerMinute  int
		wantStatus      int
		checkLocation   bool
		locationContain string
	}{
		{
			name: "Successful email send",
			formData: url.Values{
				"email": {"user@example.com"},
			},
			setupMock: func(m *mockEmailSender) {
				m.sendHTMLError = nil
			},
			authzConfig: config.AccessControlConfig{
				Emails: []string{}, // No whitelist
			},
			limitPerMinute:  10,
			wantStatus:      http.StatusSeeOther,
			checkLocation:   true,
			locationContain: "/email/sent",
		},
		{
			name: "Empty email",
			formData: url.Values{
				"email": {""},
			},
			setupMock:      func(m *mockEmailSender) {},
			authzConfig:    config.AccessControlConfig{},
			limitPerMinute: 10,
			wantStatus:     http.StatusBadRequest,
		},
		{
			name: "Invalid email format",
			formData: url.Values{
				"email": {"not-an-email"},
			},
			setupMock:      func(m *mockEmailSender) {},
			authzConfig:    config.AccessControlConfig{},
			limitPerMinute: 10,
			wantStatus:     http.StatusBadRequest,
		},
		{
			name: "Email with SMTP injection attempt",
			formData: url.Values{
				"email": {"user@example.com\r\nBcc: attacker@evil.com"},
			},
			setupMock:      func(m *mockEmailSender) {},
			authzConfig:    config.AccessControlConfig{},
			limitPerMinute: 10,
			wantStatus:     http.StatusBadRequest,
		},
		{
			name: "Unauthorized email with whitelist",
			formData: url.Values{
				"email": {"unauthorized@example.com"},
			},
			setupMock: func(m *mockEmailSender) {},
			authzConfig: config.AccessControlConfig{
				Emails: []string{"authorized@example.com"},
			},
			limitPerMinute: 10,
			wantStatus:     http.StatusForbidden,
		},
		{
			name: "Authorized email with whitelist",
			formData: url.Values{
				"email": {"authorized@example.com"},
			},
			setupMock: func(m *mockEmailSender) {
				m.sendHTMLError = nil
			},
			authzConfig: config.AccessControlConfig{
				Emails: []string{"authorized@example.com"},
			},
			limitPerMinute:  10,
			wantStatus:      http.StatusSeeOther,
			checkLocation:   true,
			locationContain: "/email/sent",
		},
		{
			name: "Email send internal error",
			formData: url.Values{
				"email": {"user@example.com"},
			},
			setupMock: func(m *mockEmailSender) {
				m.sendHTMLError = errors.New("SMTP connection failed")
			},
			authzConfig:    config.AccessControlConfig{},
			limitPerMinute: 10,
			wantStatus:     http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Service: config.ServiceConfig{
					Name: "Test Service",
				},
				Server: config.ServerConfig{
					AuthPathPrefix: "/_auth",
				},
				Session: config.SessionConfig{
					Cookie: config.CookieConfig{
						Name: "_test",
					},
				},
				AccessControl: tt.authzConfig,
			}

			sessionStore, _ := kvs.NewMemoryStore("test-"+t.Name(), kvs.MemoryConfig{})
			defer func() { _ = sessionStore.Close() }()
			translator := i18n.NewTranslator()
			logger := logging.NewTestLogger()

			mockSender := &mockEmailSender{}
			tt.setupMock(mockSender)

			emailHandler := createEmailHandler(t, mockSender, tt.authzConfig, tt.limitPerMinute)
			authzChecker := authz.NewEmailChecker(tt.authzConfig)

			middleware, err := New(
				cfg,
				sessionStore,
				nil, // oauth manager
				emailHandler,
				nil, // password handler
				authzChecker,
				nil, // forwarder
				nil, // rules evaluator
				translator,
				logger,
			)
			if err != nil {
				t.Fatalf("Failed to create middleware: %v", err)
			}

			req := httptest.NewRequest("POST", "/_auth/email/send", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			middleware.handleEmailSend(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.checkLocation {
				location := w.Header().Get("Location")
				if !strings.Contains(location, tt.locationContain) {
					t.Errorf("Location = %q, want to contain %q", location, tt.locationContain)
				}
			}
		})
	}
}

// TestHandleEmailVerify tests the email verification handler
func TestHandleEmailVerify(t *testing.T) {
	tests := []struct {
		name            string
		email           string
		authzConfig     config.AccessControlConfig
		invalidToken    bool
		emptyToken      bool
		wantStatus      int
		checkLocation   bool
		locationContain string
	}{
		{
			name:            "Valid token without whitelist",
			email:           "user@example.com",
			authzConfig:     config.AccessControlConfig{},
			wantStatus:      http.StatusFound,
			checkLocation:   true,
			locationContain: "/",
		},
		{
			name:        "Empty token",
			email:       "user@example.com",
			emptyToken:  true,
			authzConfig: config.AccessControlConfig{},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:         "Invalid token",
			email:        "user@example.com",
			invalidToken: true,
			authzConfig:  config.AccessControlConfig{},
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:  "Valid token with authorized email",
			email: "authorized@example.com",
			authzConfig: config.AccessControlConfig{
				Emails: []string{"authorized@example.com"},
			},
			wantStatus:      http.StatusFound,
			checkLocation:   true,
			locationContain: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Service: config.ServiceConfig{
					Name: "Test Service",
				},
				Server: config.ServerConfig{
					AuthPathPrefix: "/_auth",
				},
				Session: config.SessionConfig{
					Cookie: config.CookieConfig{
						Name:   "_test",
						Expire: "24h",
					},
				},
				AccessControl: tt.authzConfig,
			}

			sessionStore, _ := kvs.NewMemoryStore("test-"+t.Name(), kvs.MemoryConfig{})
			defer func() { _ = sessionStore.Close() }()
			translator := i18n.NewTranslator()
			logger := logging.NewTestLogger()

			mockSender := &mockEmailSender{}
			emailHandler := createEmailHandler(t, mockSender, tt.authzConfig, 10)
			authzChecker := authz.NewEmailChecker(tt.authzConfig)

			middleware, err := New(
				cfg,
				sessionStore,
				nil, // oauth manager
				emailHandler,
				nil, // password handler
				authzChecker,
				nil, // forwarder
				nil, // rules evaluator
				translator,
				logger,
			)
			if err != nil {
				t.Fatalf("Failed to create middleware: %v", err)
			}

			// Generate a token by sending login link
			var token string
			if tt.emptyToken {
				token = ""
			} else if tt.invalidToken {
				token = "invalid-token-that-does-not-exist"
			} else {
				// Send login link to generate token
				err := emailHandler.SendLoginLink(tt.email, i18n.English)
				if err != nil && tt.authzConfig.Emails != nil {
					// Authorization failed as expected for unauthorized emails
					// Skip token extraction
				} else if err != nil {
					t.Fatalf("Failed to send login link: %v", err)
				} else {
					// Extract token from sent email
					if len(mockSender.sentEmails) == 0 {
						t.Fatal("No email was sent")
					}
					token = extractTokenFromEmail(mockSender.sentEmails[0])
					if token == "" {
						t.Fatal("Failed to extract token from email")
					}
				}
			}

			url := "/_auth/email/verify"
			if token != "" {
				url += "?token=" + token
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			middleware.handleEmailVerify(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.checkLocation {
				location := w.Header().Get("Location")
				if !strings.Contains(location, tt.locationContain) {
					t.Errorf("Location = %q, want to contain %q", location, tt.locationContain)
				}

				// Check that session cookie is set
				cookies := w.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == "_test" && cookie.Value != "" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected session cookie to be set")
				}
			}
		})
	}
}

// TestHandleEmailVerifyOTP tests the OTP verification handler
func TestHandleEmailVerifyOTP(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		email           string
		invalidOTP      bool
		emptyOTP        bool
		authzConfig     config.AccessControlConfig
		wantStatus      int
		checkLocation   bool
		locationContain string
	}{
		{
			name:            "Valid OTP",
			method:          "POST",
			email:           "user@example.com",
			authzConfig:     config.AccessControlConfig{},
			wantStatus:      http.StatusFound,
			checkLocation:   true,
			locationContain: "/",
		},
		{
			name:        "GET request (method not allowed)",
			method:      "GET",
			email:       "user@example.com",
			authzConfig: config.AccessControlConfig{},
			wantStatus:  http.StatusMethodNotAllowed,
		},
		{
			name:        "Empty OTP",
			method:      "POST",
			email:       "user@example.com",
			emptyOTP:    true,
			authzConfig: config.AccessControlConfig{},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:            "Invalid OTP",
			method:          "POST",
			email:           "user@example.com",
			invalidOTP:      true,
			authzConfig:     config.AccessControlConfig{},
			wantStatus:      http.StatusFound,
			checkLocation:   true,
			locationContain: "/email/sent",
		},
		{
			name:   "Valid OTP with authorized email",
			method: "POST",
			email:  "authorized@example.com",
			authzConfig: config.AccessControlConfig{
				Emails: []string{"authorized@example.com"},
			},
			wantStatus:      http.StatusFound,
			checkLocation:   true,
			locationContain: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Service: config.ServiceConfig{
					Name: "Test Service",
				},
				Server: config.ServerConfig{
					AuthPathPrefix: "/_auth",
				},
				Session: config.SessionConfig{
					Cookie: config.CookieConfig{
						Name:   "_test",
						Expire: "24h",
					},
				},
				AccessControl: tt.authzConfig,
			}

			sessionStore, _ := kvs.NewMemoryStore("test-"+t.Name(), kvs.MemoryConfig{})
			defer func() { _ = sessionStore.Close() }()
			translator := i18n.NewTranslator()
			logger := logging.NewTestLogger()

			mockSender := &mockEmailSender{}
			emailHandler := createEmailHandler(t, mockSender, tt.authzConfig, 10)
			authzChecker := authz.NewEmailChecker(tt.authzConfig)

			middleware, err := New(
				cfg,
				sessionStore,
				nil, // oauth manager
				emailHandler,
				nil, // password handler
				authzChecker,
				nil, // forwarder
				nil, // rules evaluator
				translator,
				logger,
			)
			if err != nil {
				t.Fatalf("Failed to create middleware: %v", err)
			}

			// Generate OTP by sending login link
			var otp string
			if tt.emptyOTP {
				otp = ""
			} else if tt.invalidOTP {
				otp = "INVALID-OTP-123"
			} else {
				// Send login link to generate OTP
				err := emailHandler.SendLoginLink(tt.email, i18n.English)
				if err != nil && tt.authzConfig.Emails != nil {
					// Authorization failed as expected for unauthorized emails
					// Skip OTP extraction
				} else if err != nil {
					t.Fatalf("Failed to send login link: %v", err)
				} else {
					// Extract OTP from sent email
					if len(mockSender.sentEmails) == 0 {
						t.Fatal("No email was sent")
					}
					otp = extractOTPFromEmail(mockSender.sentEmails[0])
					if otp == "" {
						t.Fatal("Failed to extract OTP from email")
					}
				}
			}

			formData := url.Values{
				"otp": {otp},
			}

			req := httptest.NewRequest(tt.method, "/_auth/email/verify-otp", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			middleware.handleEmailVerifyOTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.checkLocation {
				location := w.Header().Get("Location")
				if !strings.Contains(location, tt.locationContain) {
					t.Errorf("Location = %q, want to contain %q", location, tt.locationContain)
				}

				// For successful OTP verification, check that session cookie is set
				if strings.Contains(tt.locationContain, "/") && !strings.Contains(tt.locationContain, "/email/sent") {
					cookies := w.Result().Cookies()
					found := false
					for _, cookie := range cookies {
						if cookie.Name == "_test" && cookie.Value != "" {
							found = true
							break
						}
					}
					if !found {
						t.Error("Expected session cookie to be set")
					}
				}
			}
		})
	}
}

// TestExtractUserpart tests the extractUserpart helper function
func TestExtractUserpart(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "Standard email",
			email: "user@example.com",
			want:  "user",
		},
		{
			name:  "Email with dots",
			email: "first.last@example.com",
			want:  "first.last",
		},
		{
			name:  "Email with plus",
			email: "user+tag@example.com",
			want:  "user+tag",
		},
		{
			name:  "No @ sign",
			email: "notanemail",
			want:  "notanemail",
		},
		{
			name:  "Empty string",
			email: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractUserpart(tt.email)
			if got != tt.want {
				t.Errorf("extractUserpart(%q) = %q, want %q", tt.email, got, tt.want)
			}
		})
	}
}
