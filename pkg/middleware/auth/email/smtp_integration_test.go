package email

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
)

// mockSMTPServer is a simple SMTP server mock for testing
type mockSMTPServer struct {
	listener     net.Listener
	receivedMail []string
	shouldFail   bool
	requireAuth  bool
	useTLS       bool
}

func newMockSMTPServer(t *testing.T) *mockSMTPServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create mock SMTP server: %v", err)
	}

	mock := &mockSMTPServer{
		listener:     listener,
		receivedMail: make([]string, 0),
	}

	go mock.serve()

	return mock
}

func (m *mockSMTPServer) Close() {
	if m.listener != nil {
		_ = m.listener.Close()
	}
}

func (m *mockSMTPServer) Port() int {
	return m.listener.Addr().(*net.TCPAddr).Port
}

func (m *mockSMTPServer) serve() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			return // Server closed
		}
		go m.handleConnection(conn)
	}
}

func (m *mockSMTPServer) handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Send greeting
	_, _ = writer.WriteString("220 mock.smtp.server ESMTP\r\n")
	_ = writer.Flush()

	var mailData strings.Builder
	inData := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)

		if inData {
			if line == "." {
				// End of data
				m.receivedMail = append(m.receivedMail, mailData.String())
				_, _ = writer.WriteString("250 OK: Message accepted\r\n")
				_ = writer.Flush()
				inData = false
				mailData.Reset()
				continue
			}
			mailData.WriteString(line)
			mailData.WriteString("\r\n")
			continue
		}

		// Parse SMTP commands
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "EHLO", "HELO":
			if m.requireAuth {
				_, _ = writer.WriteString("250-mock.smtp.server\r\n")
				_, _ = writer.WriteString("250 AUTH PLAIN\r\n")
			} else {
				_, _ = writer.WriteString("250 mock.smtp.server\r\n")
			}
			_ = writer.Flush()

		case "AUTH":
			if m.shouldFail {
				_, _ = writer.WriteString("535 Authentication failed\r\n")
				_ = writer.Flush()
				return
			}
			_, _ = writer.WriteString("235 Authentication successful\r\n")
			_ = writer.Flush()

		case "MAIL":
			_, _ = writer.WriteString("250 OK\r\n")
			_ = writer.Flush()

		case "RCPT":
			_, _ = writer.WriteString("250 OK\r\n")
			_ = writer.Flush()

		case "DATA":
			_, _ = writer.WriteString("354 Start mail input; end with <CRLF>.<CRLF>\r\n")
			_ = writer.Flush()
			inData = true

		case "QUIT":
			_, _ = writer.WriteString("221 Bye\r\n")
			_ = writer.Flush()
			return

		default:
			_, _ = writer.WriteString("502 Command not implemented\r\n")
			_ = writer.Flush()
		}
	}
}

// TestSMTPSender_Send tests SMTP plain text email sending
func TestSMTPSender_Send(t *testing.T) {
	mockServer := newMockSMTPServer(t)
	defer mockServer.Close()

	tests := []struct {
		name        string
		config      config.SMTPConfig
		to          string
		subject     string
		body        string
		wantError   bool
		checkResult func(*testing.T, []string)
	}{
		{
			name: "Successful plain text email without auth",
			config: config.SMTPConfig{
				Host: "127.0.0.1",
				Port: mockServer.Port(),
			},
			to:      "recipient@example.com",
			subject: "Test Subject",
			body:    "This is a test email body.",
			checkResult: func(t *testing.T, mails []string) {
				if len(mails) != 1 {
					t.Errorf("Expected 1 email, got %d", len(mails))
					return
				}
				mail := mails[0]
				if !strings.Contains(mail, "To: recipient@example.com") {
					t.Error("Email should contain To: recipient@example.com")
				}
				if !strings.Contains(mail, "Subject: Test Subject") {
					t.Error("Email should contain Subject: Test Subject")
				}
				if !strings.Contains(mail, "This is a test email body.") {
					t.Error("Email should contain body text")
				}
			},
		},
		{
			name: "Email with from name",
			config: config.SMTPConfig{
				Host:     "127.0.0.1",
				Port:     mockServer.Port(),
				From:     "service@example.com",
				FromName: "Test Service",
			},
			to:      "user@example.com",
			subject: "Hello",
			body:    "Test body",
			checkResult: func(t *testing.T, mails []string) {
				if len(mails) != 2 { // Previous test + this one
					t.Errorf("Expected 2 emails total, got %d", len(mails))
					return
				}
				mail := mails[len(mails)-1] // Get latest
				if !strings.Contains(mail, "Test Service") {
					t.Error("Email should contain sender name 'Test Service'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := NewSMTPSender(tt.config, "noreply@example.com", "Default Name")

			err := sender.Send(tt.to, tt.subject, tt.body)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if tt.checkResult != nil {
				tt.checkResult(t, mockServer.receivedMail)
			}
		})
	}
}

// TestSMTPSender_SendHTML tests SMTP HTML email sending
func TestSMTPSender_SendHTML(t *testing.T) {
	mockServer := newMockSMTPServer(t)
	defer mockServer.Close()

	cfg := config.SMTPConfig{
		Host: "127.0.0.1",
		Port: mockServer.Port(),
	}

	sender := NewSMTPSender(cfg, "noreply@example.com", "Test Service")

	htmlBody := "<html><body><h1>Hello</h1><p>This is HTML</p></body></html>"
	textBody := "Hello\nThis is plain text"

	err := sender.SendHTML("user@example.com", "HTML Test", htmlBody, textBody)
	if err != nil {
		t.Fatalf("SendHTML() error = %v", err)
	}

	if len(mockServer.receivedMail) == 0 {
		t.Fatal("No email received")
	}

	mail := mockServer.receivedMail[len(mockServer.receivedMail)-1]

	// Check multipart structure
	if !strings.Contains(mail, "Content-Type: multipart/alternative") {
		t.Error("Email should have multipart/alternative content type")
	}

	// Check boundary
	if !strings.Contains(mail, "boundary=") {
		t.Error("Email should have boundary marker")
	}

	// Check plain text part
	if !strings.Contains(mail, "Content-Type: text/plain") {
		t.Error("Email should contain plain text part")
	}
	if !strings.Contains(mail, "This is plain text") {
		t.Error("Email should contain plain text body")
	}

	// Check HTML part
	if !strings.Contains(mail, "Content-Type: text/html") {
		t.Error("Email should contain HTML part")
	}
	if !strings.Contains(mail, "<html>") {
		t.Error("Email should contain HTML body")
	}
	if !strings.Contains(mail, "<h1>Hello</h1>") {
		t.Error("Email should contain HTML heading")
	}
}

// TestSendGridSender_Send tests SendGrid plain text email sending
func TestSendGridSender_Send(t *testing.T) {
	// Create mock SendGrid API server
	var receivedRequests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Check authorization header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Error("Missing or invalid Authorization header")
		}

		// Read body
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedRequests = append(receivedRequests, string(buf[:n]))

		// Return success response
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"message":"success"}`))
	}))
	defer server.Close()

	cfg := config.SendGridConfig{
		APIKey:      "test-api-key",
		EndpointURL: server.URL, // Use mock server
	}

	sender := NewSendGridSender(cfg, "noreply@example.com", "Test Service")

	err := sender.Send("recipient@example.com", "Test Subject", "Test email body")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if len(receivedRequests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(receivedRequests))
	}

	// Verify request contains expected data
	reqBody := receivedRequests[0]
	if !strings.Contains(reqBody, "recipient@example.com") {
		t.Error("Request should contain recipient email")
	}
	if !strings.Contains(reqBody, "Test Subject") {
		t.Error("Request should contain subject")
	}
}

// TestSendGridSender_SendHTML tests SendGrid HTML email sending
func TestSendGridSender_SendHTML(t *testing.T) {
	var receivedRequests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 2048)
		n, _ := r.Body.Read(buf)
		receivedRequests = append(receivedRequests, string(buf[:n]))

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"message":"success"}`))
	}))
	defer server.Close()

	cfg := config.SendGridConfig{
		APIKey:      "test-api-key",
		EndpointURL: server.URL,
	}

	sender := NewSendGridSender(cfg, "noreply@example.com", "Test Service")

	htmlBody := "<html><body><h1>HTML Email</h1></body></html>"
	textBody := "Plain text version"

	err := sender.SendHTML("user@example.com", "HTML Test", htmlBody, textBody)
	if err != nil {
		t.Fatalf("SendHTML() error = %v", err)
	}

	if len(receivedRequests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(receivedRequests))
	}

	reqBody := receivedRequests[0]

	// Both HTML and text should be in the request
	if !strings.Contains(reqBody, "HTML Email") {
		t.Error("Request should contain HTML content")
	}
	if !strings.Contains(reqBody, "Plain text version") {
		t.Error("Request should contain plain text content")
	}
}

// TestSendGridSender_ErrorResponse tests SendGrid error handling
func TestSendGridSender_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return error response
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Invalid API key"}]}`))
	}))
	defer server.Close()

	cfg := config.SendGridConfig{
		APIKey:      "invalid-key",
		EndpointURL: server.URL,
	}

	sender := NewSendGridSender(cfg, "noreply@example.com", "Test Service")

	err := sender.Send("user@example.com", "Test", "Body")
	if err == nil {
		t.Error("Expected error for invalid API key, got nil")
	}

	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Error should mention status code 400, got: %v", err)
	}
}

// TestSendmailSender_Send tests sendmail command execution
func TestSendmailSender_Send(t *testing.T) {
	// This test creates a mock sendmail script
	// Note: This is a basic test - in production, sendmail integration would be tested differently

	cfg := config.SendmailConfig{
		Path: "/usr/sbin/sendmail", // Will fail if sendmail not installed, which is OK for unit tests
	}

	sender := NewSendmailSender(cfg, "noreply@example.com", "Test Service")

	// We expect this to fail in test environments without sendmail
	// The purpose is to verify the code path compiles and structures the message correctly
	err := sender.Send("user@example.com", "Test", "Body")

	// We don't assert on error because sendmail might not be installed
	// This test mainly verifies the code compiles and doesn't panic
	t.Logf("Sendmail Send() result: %v (expected to fail without sendmail installed)", err)
}

// TestSendmailSender_SendHTML tests sendmail HTML email
func TestSendmailSender_SendHTML(t *testing.T) {
	cfg := config.SendmailConfig{
		Path: "/usr/sbin/sendmail",
	}

	sender := NewSendmailSender(cfg, "noreply@example.com", "Test Service")

	htmlBody := "<html><body>HTML</body></html>"
	textBody := "Plain text"

	err := sender.SendHTML("user@example.com", "Test", htmlBody, textBody)

	t.Logf("Sendmail SendHTML() result: %v (expected to fail without sendmail installed)", err)
}

// TestSendmailSender_GetSendmailPath tests path resolution
func TestSendmailSender_GetSendmailPath(t *testing.T) {
	tests := []struct {
		name         string
		configPath   string
		expectedPath string
	}{
		{
			name:         "Default path when not configured",
			configPath:   "",
			expectedPath: "/usr/sbin/sendmail",
		},
		{
			name:         "Custom path when configured",
			configPath:   "/custom/path/sendmail",
			expectedPath: "/custom/path/sendmail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.SendmailConfig{
				Path: tt.configPath,
			}

			sender := NewSendmailSender(cfg, "test@example.com", "Test")

			path := sender.getSendmailPath()
			if path != tt.expectedPath {
				t.Errorf("getSendmailPath() = %q, want %q", path, tt.expectedPath)
			}
		})
	}
}

// TestSMTPSender_MessageFormat tests SMTP message formatting
func TestSMTPSender_MessageFormat(t *testing.T) {
	mockServer := newMockSMTPServer(t)
	defer mockServer.Close()

	tests := []struct {
		name      string
		fromName  string
		to        string
		subject   string
		body      string
		checkMail func(*testing.T, string)
	}{
		{
			name:     "Message with from name",
			fromName: "Service Name",
			to:       "user@example.com",
			subject:  "Test Subject",
			body:     "Test Body",
			checkMail: func(t *testing.T, mail string) {
				// Check From header includes name
				if !strings.Contains(mail, "From: Service Name <noreply@example.com>") {
					t.Error("From header should include sender name")
				}
			},
		},
		{
			name:     "Message without from name",
			fromName: "",
			to:       "user@example.com",
			subject:  "Test",
			body:     "Body",
			checkMail: func(t *testing.T, mail string) {
				// Check From header is just email
				if !strings.Contains(mail, "From: noreply@example.com") {
					t.Error("From header should be just email address")
				}
			},
		},
		{
			name:     "UTF-8 content",
			fromName: "テストサービス",
			to:       "user@example.com",
			subject:  "日本語の件名",
			body:     "これは日本語のメール本文です。",
			checkMail: func(t *testing.T, mail string) {
				if !strings.Contains(mail, "Content-Type: text/plain; charset=UTF-8") {
					t.Error("Should specify UTF-8 charset")
				}
				if !strings.Contains(mail, "これは日本語のメール本文です。") {
					t.Error("Should contain Japanese text")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.SMTPConfig{
				Host:     "127.0.0.1",
				Port:     mockServer.Port(),
				From:     "noreply@example.com",
				FromName: tt.fromName,
			}

			sender := NewSMTPSender(cfg, "", "")

			err := sender.Send(tt.to, tt.subject, tt.body)
			if err != nil {
				t.Fatalf("Send() error = %v", err)
			}

			if len(mockServer.receivedMail) == 0 {
				t.Fatal("No mail received")
			}

			mail := mockServer.receivedMail[len(mockServer.receivedMail)-1]
			if tt.checkMail != nil {
				tt.checkMail(t, mail)
			}
		})
	}
}

// TestSendGridSender_CustomEndpoint tests custom endpoint URL handling
func TestSendGridSender_CustomEndpoint(t *testing.T) {
	customEndpoint := "https://custom.sendgrid.endpoint/v3/mail/send"

	// We can't actually test HTTP requests without a server,
	// but we can verify the client is configured correctly
	cfg := config.SendGridConfig{
		APIKey:      "test-key",
		EndpointURL: customEndpoint,
	}

	sender := NewSendGridSender(cfg, "test@example.com", "Test")

	if sender.client.BaseURL != customEndpoint {
		t.Errorf("BaseURL = %q, want %q", sender.client.BaseURL, customEndpoint)
	}
}
