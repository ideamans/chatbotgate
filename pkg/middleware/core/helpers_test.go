package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// TestIsValidRedirectURL tests open redirect prevention
func TestIsValidRedirectURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Valid relative URLs
		{"Valid simple path", "/", true},
		{"Valid path", "/path", true},
		{"Valid nested path", "/path/to/page", true},
		{"Valid with query", "/path?foo=bar", true},
		{"Valid with fragment", "/path#section", true},
		{"Valid with query and fragment", "/path?foo=bar#section", true},
		{"Valid auth path", "/_auth/login", true},

		// Invalid cases - Empty
		{"Empty URL", "", false},

		// Invalid cases - Protocol-relative URLs (open redirect)
		{"Protocol relative", "//evil.com", false},
		{"Protocol relative with path", "//evil.com/path", false},

		// Invalid cases - Absolute URLs (open redirect)
		{"Absolute HTTP", "http://evil.com", false},
		{"Absolute HTTPS", "https://evil.com", false},
		{"Absolute FTP", "ftp://evil.com", false},
		{"Absolute with path", "http://evil.com/path", false},
		{"Absolute localhost", "http://localhost:8080", false},

		// Invalid cases - Not starting with /
		{"Relative without slash", "path", false},
		{"Domain only", "evil.com", false},

		// Edge cases - Special characters (should be valid)
		{"With space encoded", "/path%20with%20space", true},
		{"With special chars", "/path?key=value&other=123", true},
		{"With encoded slash", "/path%2Fencoded", true},

		// Edge cases - Tab/newline in path (should be valid - sanitization happens elsewhere)
		{"With tab", "/path\t/page", true},
		{"With newline", "/path\n/page", true},

		// Edge cases - Contains :// (currently rejected by implementation for safety)
		// Note: The current implementation rejects any URL containing "://" even in query/fragment
		// This is conservative but prevents edge cases in URL parsing
		{"Contains :// in query", "/redirect?url=http://example.com", false},
		{"Contains :// in fragment", "/page#section://test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidRedirectURL(tt.url)
			if result != tt.expected {
				t.Errorf("isValidRedirectURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

// TestIsValidEmail tests email validation and SMTP injection prevention
func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		// Valid emails
		{"Valid simple", "user@example.com", true},
		{"Valid with plus", "user+tag@example.com", true},
		{"Valid with dots", "user.name@example.com", true},
		{"Valid with numbers", "user123@example.com", true},
		{"Valid with hyphen", "user-name@example.com", true},
		{"Valid subdomain", "user@mail.example.com", true},
		{"Valid long domain", "user@subdomain.example.co.uk", true},
		{"Valid with underscore", "user_name@example.com", true},

		// Invalid - Malformed
		{"Empty", "", false},
		{"No @", "userexample.com", false},
		{"Double @", "user@@example.com", false},
		{"Missing domain", "user@", false},
		{"Missing local", "@example.com", false},
		{"Just @", "@", false},
		{"No domain extension", "user@localhost", true}, // mail.ParseAddress allows this (valid per RFC)

		// Invalid - SMTP injection attempts (control characters)
		{"CR injection", "user@example.com\r", false},
		{"LF injection", "user@example.com\n", false},
		{"CRLF injection", "user@example.com\r\n", false},
		{"CR in local", "user\r@example.com", false},
		{"LF in local", "user\n@example.com", false},
		{"CRLF in middle", "user\r\n@example.com", false},
		{"Null byte", "user\x00@example.com", false},
		{"Tab character", "user\t@example.com", false},
		{"Multiple control chars", "user\r\n\x00@example.com", false},

		// Invalid - SMTP header injection attempts
		{"CC injection", "user@example.com\nCC: evil@evil.com", false},
		{"BCC injection", "user@example.com\rBCC: evil@evil.com", false},
		{"Subject injection", "user@example.com\nSubject: spam", false},

		// Edge cases - Valid but unusual
		{"With name", "User Name <user@example.com>", true}, // mail.ParseAddress extracts the address
		{"Quoted local", "\"user.name\"@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidEmail(tt.email)
			if result != tt.expected {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, result, tt.expected)
			}
		})
	}
}

// TestSanitizeHeaderValue tests header injection prevention and DoS protection
func TestSanitizeHeaderValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Clean values - passthrough
		{"Clean alphanumeric", "abc123", "abc123"},
		{"Clean with spaces", "hello world", "hello world"},
		{"Clean with symbols", "user@example.com", "user@example.com"},
		{"Clean URL", "https://example.com/path", "https://example.com/path"},
		{"Clean JSON", `{"key":"value"}`, `{"key":"value"}`},

		// Control character removal - CR/LF injection
		{"CR removal", "value\rinjection", "valueinjection"},
		{"LF removal", "value\ninjection", "valueinjection"},
		{"CRLF removal", "value\r\ninjection", "valueinjection"},
		{"Multiple CRLF", "a\r\nb\r\nc\r\n", "abc"},

		// Control character removal - Other control chars
		{"Null byte", "value\x00injection", "valueinjection"},
		{"Tab removal", "value\tinjection", "valueinjection"},
		{"Bell character", "value\x07injection", "valueinjection"},
		{"Escape character", "value\x1binjection", "valueinjection"},
		{"DEL character", "value\x7finjection", "valueinjection"},

		// Header injection attempts
		{"Header injection 1", "value\r\nX-Injected: malicious", "valueX-Injected: malicious"},
		{"Header injection 2", "value\nSet-Cookie: evil=true", "valueSet-Cookie: evil=true"},
		{"Header injection 3", "value\rLocation: http://evil.com", "valueLocation: http://evil.com"},

		// Length limiting (DoS protection)
		{"Short string", "short", "short"},
		{"Max length", strings.Repeat("a", 8192), strings.Repeat("a", 8192)},
		{"Over max length", strings.Repeat("a", 10000), strings.Repeat("a", 8192)},
		{"Way over max", strings.Repeat("a", 100000), strings.Repeat("a", 8192)},

		// Edge cases
		{"Empty string", "", ""},
		{"Only control chars", "\r\n\t\x00", ""},
		{"Mixed control and normal", "a\rb\nc\td", "abcd"},
		{"Unicode preserved", "Hello 世界", "Hello 世界"},
		{"Emoji preserved", "Hello 👋", "Hello 👋"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeHeaderValue(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeHeaderValue() mismatch\ninput:    %q\nexpected: %q\ngot:      %q",
					tt.input, tt.expected, result)
			}
			// Verify length constraint
			if len(result) > 8192 {
				t.Errorf("sanitizeHeaderValue() result exceeds max length: %d > 8192", len(result))
			}
		})
	}
}

// TestIsStaticResource tests static resource detection
func TestIsStaticResource(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Static resources
		{"Favicon", "/favicon.ico", true},
		{"Robots", "/robots.txt", true},
		{"Apple touch icon", "/apple-touch-icon.png", true},
		{"Apple touch icon precomposed", "/apple-touch-icon-precomposed.png", true},
		{"Auth assets CSS", "/_auth/assets/main.css", true},
		{"Auth assets icon", "/_auth/assets/icons/google.svg", true},
		{"Auth assets nested", "/_auth/assets/nested/file.js", true},

		// Non-static resources
		{"Root", "/", false},
		{"Login page", "/_auth/login", false},
		{"Regular page", "/page", false},
		{"Similar path", "/favicon.ico.bak", true}, // Starts with /favicon.ico, so matches
		{"Assets outside auth", "/assets/file.css", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStaticResource(tt.path)
			if result != tt.expected {
				t.Errorf("isStaticResource(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestNormalizeAuthPrefix tests auth path prefix normalization
func TestNormalizeAuthPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty defaults to /_auth", "", "/_auth"},
		{"Already normalized", "/_auth", "/_auth"},
		{"Without leading slash", "auth", "/auth"},
		{"With trailing slash", "/_auth/", "/_auth"},
		{"Without leading, with trailing", "auth/", "/auth"},
		{"Custom prefix", "/_oauth2_proxy", "/_oauth2_proxy"},
		{"Just slash", "/", "/"},
		{"Multiple trailing slashes", "/_auth//", "/_auth/"}, // Only removes one trailing slash
		{"Complex path", "/custom/path/", "/custom/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeAuthPrefix(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeAuthPrefix(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestExtractPathParam tests path parameter extraction
func TestExtractPathParam(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		prefix   string
		expected string
	}{
		{"Extract provider", "/_auth/oauth2/start/google", "/_auth/oauth2/start/", "google"},
		{"Extract with query", "/_auth/oauth2/start/github?state=abc", "/_auth/oauth2/start/", "github"},
		{"Extract with trailing slash", "/_auth/oauth2/start/microsoft/", "/_auth/oauth2/start/", "microsoft"},
		{"Empty when no match", "/other/path", "/_auth/oauth2/start/", ""},
		{"Empty when prefix only", "/_auth/oauth2/start/", "/_auth/oauth2/start/", ""},
		{"Multi-segment", "/_auth/icons/provider/google.svg", "/_auth/icons/provider/", "google.svg"},
		{"With query and fragment", "/_auth/callback?code=123#section", "/_auth/", "callback"}, // Query is removed by implementation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPathParam(tt.path, tt.prefix)
			if result != tt.expected {
				t.Errorf("extractPathParam(%q, %q) = %q, want %q", tt.path, tt.prefix, result, tt.expected)
			}
		})
	}
}

// TestMaskEmail tests email masking for logging
func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{"Normal email", "user@example.com", "u***@example.com"},
		{"Long local part", "verylongusername@example.com", "v***@example.com"},
		{"Single char local", "a@example.com", "*@example.com"},
		{"Two char local", "ab@example.com", "a***@example.com"},
		{"With plus", "user+tag@example.com", "u***@example.com"},
		{"Empty email", "", "[EMPTY]"},
		{"No @ symbol", "notanemail", "[INVALID_EMAIL]"},
		{"@ at start", "@example.com", "[INVALID_EMAIL]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskEmail(tt.email)
			if result != tt.expected {
				t.Errorf("maskEmail(%q) = %q, want %q", tt.email, result, tt.expected)
			}
		})
	}
}

// TestGetRedirectURL tests redirect URL retrieval and security validation
func TestGetRedirectURL(t *testing.T) {
	// Create minimal middleware for testing
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Session: config.SessionConfig{
			Cookie: config.CookieConfig{
				Name:   "test_session",
				Secret: "test-secret-key-32-bytes-long!",
			},
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
	}

	logger := logging.NewSimpleLogger("test", logging.LevelError, false)
	translator := i18n.NewTranslator()
	sessionStore, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{})
	defer func() { _ = sessionStore.Close() }()

	mw, err := New(cfg, sessionStore, nil, nil, nil, nil, nil, nil, translator, logger)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	tests := []struct {
		name         string
		cookieValue  string
		hasCookie    bool
		expectedURL  string
		expectDelete bool
	}{
		{"No cookie returns /", "", false, "/", false},
		{"Valid relative URL", "/dashboard", true, "/dashboard", true},
		{"Valid nested path", "/path/to/page", true, "/path/to/page", true},
		{"Invalid protocol-relative", "//evil.com", true, "/", true},
		{"Invalid absolute URL", "http://evil.com", true, "/", true},
		{"Invalid empty", "", true, "/", true},
		{"Valid with query", "/page?foo=bar", true, "/page?foo=bar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/_auth/login", nil)
			if tt.hasCookie {
				req.AddCookie(&http.Cookie{
					Name:  redirectCookieName,
					Value: tt.cookieValue,
				})
			}

			rec := httptest.NewRecorder()
			result := mw.getRedirectURL(rec, req)

			if result != tt.expectedURL {
				t.Errorf("getRedirectURL() = %q, want %q", result, tt.expectedURL)
			}

			// Check if cookie was deleted
			cookies := rec.Result().Cookies()
			foundDeleteCookie := false
			for _, cookie := range cookies {
				if cookie.Name == redirectCookieName && cookie.MaxAge == -1 {
					foundDeleteCookie = true
					break
				}
			}

			if tt.expectDelete && !foundDeleteCookie {
				t.Error("Expected redirect cookie to be deleted, but it wasn't")
			}
		})
	}
}

// TestSetSecurityHeaders tests security header setting
func TestSetSecurityHeaders(t *testing.T) {
	tests := []struct {
		name        string
		development bool
		checkCSP    func(string) bool
	}{
		{
			name:        "Production mode - strict CSP",
			development: false,
			checkCSP: func(csp string) bool {
				// Check that script-src does NOT contain unsafe-inline
				// (style-src CAN have unsafe-inline, so we need to check script-src specifically)
				return (strings.Contains(csp, "script-src 'self';") || strings.Contains(csp, "script-src 'self'; ")) &&
					!strings.Contains(csp, "script-src 'self' 'unsafe-inline'")
			},
		},
		{
			name:        "Development mode - allows unsafe-inline",
			development: true,
			checkCSP: func(csp string) bool {
				return strings.Contains(csp, "script-src 'self' 'unsafe-inline';")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Service: config.ServiceConfig{
					Name: "Test Service",
				},
				Session: config.SessionConfig{
					Cookie: config.CookieConfig{
						Name:   "test_session",
						Secret: "test-secret-key-32-bytes-long!",
					},
				},
				Server: config.ServerConfig{
					AuthPathPrefix: "/_auth",
					Development:    tt.development,
				},
			}

			logger := logging.NewSimpleLogger("test", logging.LevelError, false)
			translator := i18n.NewTranslator()
			sessionStore, _ := kvs.NewMemoryStore("test", kvs.MemoryConfig{})
			defer func() { _ = sessionStore.Close() }()

			mw, err := New(cfg, sessionStore, nil, nil, nil, nil, nil, nil, translator, logger)
			if err != nil {
				t.Fatalf("Failed to create middleware: %v", err)
			}

			rec := httptest.NewRecorder()
			mw.setSecurityHeaders(rec)

			headers := rec.Header()

			// Check CSP
			csp := headers.Get("Content-Security-Policy")
			if csp == "" {
				t.Error("Content-Security-Policy header not set")
			} else if !tt.checkCSP(csp) {
				t.Errorf("CSP check failed: %q", csp)
			}

			// Check other security headers
			expectedHeaders := map[string]string{
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
				"X-XSS-Protection":       "1; mode=block",
				"Referrer-Policy":        "strict-origin-when-cross-origin",
			}

			for header, expected := range expectedHeaders {
				actual := headers.Get(header)
				if actual != expected {
					t.Errorf("Header %q = %q, want %q", header, actual, expected)
				}
			}
		})
	}
}
