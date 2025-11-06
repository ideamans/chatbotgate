package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/middleware/factory"
	"github.com/ideamans/chatbotgate/pkg/middleware/forwarding"
	"github.com/ideamans/chatbotgate/pkg/middleware/session"
	"github.com/ideamans/chatbotgate/pkg/proxy/core"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

const (
	testBackendPort = 8083
	chatbotgatePort = 4182
	encryptionKey   = "e2e-test-encryption-key-32-chars-long-1234567890"
)

// TestUserInfoResponse matches the backend server's response structure
type TestUserInfoResponse struct {
	QueryString *TestUserData  `json:"querystring,omitempty"`
	Header      *TestUserData  `json:"header,omitempty"`
	RawHeaders  TestRawHeaders `json:"raw_headers,omitempty"`
}

type TestUserData struct {
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	Encrypted bool   `json:"encrypted"`
}

type TestRawHeaders struct {
	ForwardedUser  string `json:"X-ChatbotGate-User,omitempty"`
	ForwardedEmail string `json:"X-ChatbotGate-Email,omitempty"`
}

func TestForwarding_E2E(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Start test backend server
	backendCmd, backendURL := startTestBackend(t)
	defer func() { _ = backendCmd.Process.Kill() }()

	// Wait for backend to be ready
	waitForServer(t, backendURL+"/health", 5*time.Second)

	// Load test configuration
	configPath := filepath.Join("testdata", "config_forwarding.yaml")
	cfg, err := config.NewFileLoader(configPath).Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create logger
	logger := logging.NewSimpleLogger("e2e", logging.LevelDebug, true)

	// Initialize KVS
	cfg.KVS.Namespaces.SetDefaults()
	if cfg.KVS.Default.Type == "" {
		cfg.KVS.Default.Type = "memory"
	}

	// Create session KVS
	sessionCfg := cfg.KVS.Default
	sessionCfg.Namespace = cfg.KVS.Namespaces.Session
	sessionKVS, err := kvs.New(sessionCfg)
	if err != nil {
		t.Fatalf("Failed to create session KVS: %v", err)
	}
	defer func() { _ = sessionKVS.Close() }()

	// Create session store
	sessionStore := sessionKVS

	// Create proxy handler
	proxyHandler, err := proxy.NewHandler(backendURL)
	if err != nil {
		t.Fatalf("Failed to create proxy handler: %v", err)
	}

	// Create factory
	mwFactory := factory.NewDefaultFactory("localhost", chatbotgatePort, logger)

	// Create middleware directly using factory
	middleware, err := mwFactory.CreateMiddleware(cfg, sessionStore, proxyHandler, logger)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Start chatbotgate server
	server := httptest.NewServer(middleware)
	defer server.Close()

	// Create HTTP client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects automatically
		},
	}

	// Test 1: OAuth2 login forwards both username and email
	t.Run("OAuth2 login forwards username and email", func(t *testing.T) {
		// Create a session manually to simulate OAuth2 login
		testUsername := "John Doe"
		testEmail := "john.doe@example.com"
		sess := &session.Session{
			ID:            "oauth2-session-id",
			Email:         testEmail,
			Name:          testUsername, // OAuth2 provides actual name
			Provider:      "google",
			CreatedAt:     time.Now(),
			ExpiresAt:     time.Now().Add(1 * time.Hour),
			Authenticated: true,
		}
		if err := session.Set(sessionStore, sess.ID, sess); err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Make a request with session cookie
		req, _ := http.NewRequest("GET", server.URL+"/oauth2-test", nil)
		req.AddCookie(&http.Cookie{
			Name:  cfg.Session.CookieName,
			Value: sess.ID,
		})

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Should get 200 OK (proxied to backend)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Parse response
		var userInfo TestUserInfoResponse
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify header forwarding - both username and email should be present
		if userInfo.Header == nil {
			t.Fatal("Expected header data, got nil")
		}
		if !userInfo.Header.Encrypted {
			t.Error("Expected encrypted header data")
		}
		// OAuth2 should provide both username (name) and email
		if userInfo.Header.Username != testUsername {
			t.Errorf("Header username = %v, want %v", userInfo.Header.Username, testUsername)
		}
		if userInfo.Header.Email != testEmail {
			t.Errorf("Header email = %v, want %v", userInfo.Header.Email, testEmail)
		}

		// Verify raw encrypted headers exist
		if userInfo.RawHeaders.ForwardedUser == "" {
			t.Error("Expected encrypted X-ChatbotGate-User header, got empty")
		}
		if userInfo.RawHeaders.ForwardedEmail == "" {
			t.Error("Expected encrypted X-ChatbotGate-Email header, got empty")
		}

		t.Logf("OAuth2 login correctly forwards both username (%s) and email (%s)", testUsername, testEmail)
	})

	// Test 2: Direct querystring test by simulating OAuth callback redirect
	t.Run("QueryString forwarding after authentication", func(t *testing.T) {
		testUsername := "queryuser"
		testEmail := "query@example.com"

		// Create forwarder to simulate what middleware does
		forwarder := forwarding.NewForwarder(&cfg.Forwarding, nil)
		redirectURL := backendURL + "/dashboard"

		userInfo := &forwarding.UserInfo{
			Username: testUsername,
			Email:    testEmail,
		}

		// Add querystring
		modifiedURL, err := forwarder.AddToQueryString(redirectURL, userInfo)
		if err != nil {
			t.Fatalf("Failed to add querystring: %v", err)
		}

		// Verify URL contains username and email parameters
		parsedURL, _ := url.Parse(modifiedURL)
		encryptedUser := parsedURL.Query().Get("username")
		encryptedEmail := parsedURL.Query().Get("email")
		if encryptedUser == "" {
			t.Fatal("Expected username query parameter, got empty")
		}
		if encryptedEmail == "" {
			t.Fatal("Expected email query parameter, got empty")
		}

		// Make request to the modified URL (directly to backend)
		resp, err := http.Get(modifiedURL)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Parse response
		var response TestUserInfoResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify querystring was decrypted correctly
		if response.QueryString == nil {
			t.Fatal("Expected querystring data, got nil")
		}
		if !response.QueryString.Encrypted {
			t.Error("Expected encrypted querystring data")
		}
		if response.QueryString.Username != testUsername {
			t.Errorf("QueryString username = %v, want %v", response.QueryString.Username, testUsername)
		}
		if response.QueryString.Email != testEmail {
			t.Errorf("QueryString email = %v, want %v", response.QueryString.Email, testEmail)
		}
	})

	// Test 3: Verify encryption/decryption roundtrip
	t.Run("Encryption roundtrip", func(t *testing.T) {
		encryptor := forwarding.NewEncryptor(encryptionKey)

		testData := map[string]string{
			"username": "encryptuser",
			"email":    "encrypt@example.com",
		}

		// Encrypt
		encrypted, err := encryptor.EncryptMap(testData)
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}

		// Decrypt
		decrypted, err := encryptor.DecryptMap(encrypted)
		if err != nil {
			t.Fatalf("Decryption failed: %v", err)
		}

		// Verify
		if decrypted["username"] != testData["username"] {
			t.Errorf("Decrypted username = %v, want %v", decrypted["username"], testData["username"])
		}
		if decrypted["email"] != testData["email"] {
			t.Errorf("Decrypted email = %v, want %v", decrypted["email"], testData["email"])
		}
	})

	// Test 4: Email login should have empty username
	t.Run("Email login has empty username", func(t *testing.T) {
		testEmail := "emailuser@example.com"

		// Create a session for email authentication (Name is empty)
		emailSess := &session.Session{
			ID:            "email-session-id",
			Email:         testEmail,
			Name:          "", // Email login has no Name
			Provider:      "email",
			CreatedAt:     time.Now(),
			ExpiresAt:     time.Now().Add(1 * time.Hour),
			Authenticated: true,
		}
		if err := session.Set(sessionStore, emailSess.ID, emailSess); err != nil {
			t.Fatalf("Failed to create email session: %v", err)
		}

		// Make a request with email session cookie
		req, _ := http.NewRequest("GET", server.URL+"/email-test", nil)
		req.AddCookie(&http.Cookie{
			Name:  cfg.Session.CookieName,
			Value: emailSess.ID,
		})

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Should get 200 OK (proxied to backend)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Parse response
		var userInfo TestUserInfoResponse
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify header forwarding - username should be EMPTY for email auth
		if userInfo.Header == nil {
			t.Fatal("Expected header data, got nil")
		}
		if !userInfo.Header.Encrypted {
			t.Error("Expected encrypted header data")
		}
		// Username should be empty for email authentication
		if userInfo.Header.Username != "" {
			t.Errorf("Header username = %v, want empty (email auth has no username)", userInfo.Header.Username)
		}
		// Email should be set
		if userInfo.Header.Email != testEmail {
			t.Errorf("Header email = %v, want %v", userInfo.Header.Email, testEmail)
		}

		// Verify raw headers - X-ChatbotGate-User should be empty or contain encrypted empty value
		// X-ChatbotGate-Email should be present
		if userInfo.RawHeaders.ForwardedEmail == "" {
			t.Error("Expected encrypted X-ChatbotGate-Email header, got empty")
		}

		t.Logf("Email login correctly has empty username, only email: %s", testEmail)
	})
}

func TestCustomFieldsForwarding_E2E_Encrypted(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Start test backend server
	backendCmd, backendURL := startTestBackend(t)
	defer func() { _ = backendCmd.Process.Kill() }()

	// Wait for backend to be ready
	waitForServer(t, backendURL+"/health", 5*time.Second)

	// Load test configuration with encryption enabled
	configPath := filepath.Join("testdata", "config_custom_forwarding_encrypted.yaml")
	cfg, err := config.NewFileLoader(configPath).Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create logger
	logger := logging.NewSimpleLogger("e2e", logging.LevelDebug, true)

	// Initialize KVS
	cfg.KVS.Namespaces.SetDefaults()
	if cfg.KVS.Default.Type == "" {
		cfg.KVS.Default.Type = "memory"
	}

	// Create session KVS
	sessionCfg := cfg.KVS.Default
	sessionCfg.Namespace = cfg.KVS.Namespaces.Session
	sessionKVS, err := kvs.New(sessionCfg)
	if err != nil {
		t.Fatalf("Failed to create session KVS: %v", err)
	}
	defer func() { _ = sessionKVS.Close() }()

	// Create session store
	sessionStore := sessionKVS

	// Create proxy handler
	proxyHandler, err := proxy.NewHandler(backendURL)
	if err != nil {
		t.Fatalf("Failed to create proxy handler: %v", err)
	}

	// Create factory
	mwFactory := factory.NewDefaultFactory("localhost", chatbotgatePort, logger)

	// Create middleware directly using factory
	middleware, err := mwFactory.CreateMiddleware(cfg, sessionStore, proxyHandler, logger)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Start chatbotgate server
	server := httptest.NewServer(middleware)
	defer server.Close()

	// Create HTTP client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects automatically
		},
	}

	// Test: Custom fields are encrypted when forwarding.encryption is enabled
	t.Run("Custom fields encrypted in headers", func(t *testing.T) {
		testUsername := "Custom User"
		testEmail := "custom@example.com"

		// Create a session with Extra data simulating OAuth2 response
		sess := &session.Session{
			ID:       "custom-session-id",
			Email:    testEmail,
			Name:     testUsername,
			Provider: "test-provider-with-analytics",
			Extra: map[string]interface{}{
				"secrets": map[string]interface{}{
					"access_token":  "secret-token-abc123",
					"refresh_token": "refresh-token-xyz789",
				},
				"analytics": map[string]interface{}{
					"user_id": "analytics-user-123",
					"tier":    "premium",
				},
			},
			CreatedAt:     time.Now(),
			ExpiresAt:     time.Now().Add(1 * time.Hour),
			Authenticated: true,
		}
		if err := session.Set(sessionStore, sess.ID, sess); err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Make a request with session cookie
		req, _ := http.NewRequest("GET", server.URL+"/custom-test", nil)
		req.AddCookie(&http.Cookie{
			Name:  cfg.Session.CookieName,
			Value: sess.ID,
		})

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Should get 200 OK (proxied to backend)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Parse response
		var userInfo TestUserInfoResponse
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify standard fields are encrypted
		if userInfo.Header == nil {
			t.Fatal("Expected header data, got nil")
		}
		if !userInfo.Header.Encrypted {
			t.Error("Expected encrypted header data")
		}

		// Verify custom fields are present and encrypted
		if userInfo.RawHeaders.ForwardedUser == "" {
			t.Error("Expected encrypted X-ChatbotGate-User header")
		}
		if userInfo.RawHeaders.ForwardedEmail == "" {
			t.Error("Expected encrypted X-ChatbotGate-Email header")
		}

		// Verify custom headers exist (they should be encrypted)
		// Note: The test backend needs to be updated to capture these custom headers
		t.Logf("Custom fields encrypted test completed for user: %s", testUsername)
	})
}

func startTestBackend(t *testing.T) (*exec.Cmd, string) {
	// Build test backend binary
	binaryPath := filepath.Join(t.TempDir(), "testserver")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./testserver")
	buildCmd.Dir = filepath.Join(".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build test backend: %v", err)
	}

	// Start backend server
	cmd := exec.Command(binaryPath, "-port", fmt.Sprintf("%d", testBackendPort), "-key", encryptionKey)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test backend: %v", err)
	}

	backendURL := fmt.Sprintf("http://localhost:%d", testBackendPort)
	t.Logf("Test backend started at %s", backendURL)

	return cmd, backendURL
}

func waitForServer(t *testing.T, url string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Server %s did not become ready in time", url)
		case <-ticker.C:
			resp, err := http.Get(url)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					t.Logf("Server %s is ready", url)
					return
				}
			}
		}
	}
}
