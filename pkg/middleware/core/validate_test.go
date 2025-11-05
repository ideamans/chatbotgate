package middleware

import (
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
)

// Helper function to create a valid base configuration
func validConfig() *config.Config {
	return &config.Config{
		Service: config.ServiceConfig{
			Name: "Test Service",
		},
		Server: config.ServerConfig{
			AuthPathPrefix: "/_auth",
		},
		Session: config.SessionConfig{
			CookieName:   "_oauth2_proxy",
			CookieSecret: "this-is-a-very-long-secret-key-with-at-least-32-characters",
			CookieExpire: "168h",
		},
		OAuth2: config.OAuth2Config{
			Providers: []config.OAuth2Provider{
				{
					Name:         "google",
					Type:         "google",
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
				},
			},
		},
	}
}

func TestValidateConfig_ValidConfiguration(t *testing.T) {
	cfg := validConfig()
	errs := ValidateConfig(cfg)

	if errs != nil {
		t.Errorf("Expected no validation errors, got %d errors: %v", len(errs), errs)
	}
}

func TestValidateConfig_MissingServiceName(t *testing.T) {
	cfg := validConfig()
	cfg.Service.Name = ""

	errs := ValidateConfig(cfg)
	if errs == nil {
		t.Fatal("Expected validation errors, got nil")
	}

	found := false
	for _, err := range errs {
		if err.Field == "service.name" {
			found = true
			if !strings.Contains(err.Message, "required") {
				t.Errorf("Expected 'required' in error message, got: %s", err.Message)
			}
		}
	}
	if !found {
		t.Error("Expected validation error for service.name")
	}
}

func TestValidateConfig_InvalidAuthPathPrefix(t *testing.T) {
	tests := []struct {
		name          string
		prefix        string
		expectedError string
	}{
		{
			name:          "Missing leading slash",
			prefix:        "auth",
			expectedError: "must start with '/'",
		},
		{
			name:          "Trailing slash",
			prefix:        "/_auth/",
			expectedError: "must not end with '/'",
		},
		{
			name:          "Contains whitespace",
			prefix:        "/_auth path",
			expectedError: "must not contain whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Server.AuthPathPrefix = tt.prefix

			errs := ValidateConfig(cfg)
			if errs == nil {
				t.Fatal("Expected validation errors, got nil")
			}

			found := false
			for _, err := range errs {
				if err.Field == "server.auth_path_prefix" {
					found = true
					if !strings.Contains(err.Message, tt.expectedError) {
						t.Errorf("Expected error message to contain '%s', got: %s", tt.expectedError, err.Message)
					}
				}
			}
			if !found {
				t.Error("Expected validation error for server.auth_path_prefix")
			}
		})
	}
}

func TestValidateConfig_SessionValidation(t *testing.T) {
	tests := []struct {
		name          string
		modifyConfig  func(*config.Config)
		expectedField string
		expectedError string
	}{
		{
			name: "Missing cookie name",
			modifyConfig: func(cfg *config.Config) {
				cfg.Session.CookieName = ""
			},
			expectedField: "session.cookie_name",
			expectedError: "required",
		},
		{
			name: "Missing cookie secret",
			modifyConfig: func(cfg *config.Config) {
				cfg.Session.CookieSecret = ""
			},
			expectedField: "session.cookie_secret",
			expectedError: "required",
		},
		{
			name: "Cookie secret too short",
			modifyConfig: func(cfg *config.Config) {
				cfg.Session.CookieSecret = "short"
			},
			expectedField: "session.cookie_secret",
			expectedError: "at least 32 characters",
		},
		{
			name: "Missing cookie expire",
			modifyConfig: func(cfg *config.Config) {
				cfg.Session.CookieExpire = ""
			},
			expectedField: "session.cookie_expire",
			expectedError: "required",
		},
		{
			name: "Invalid cookie expire format",
			modifyConfig: func(cfg *config.Config) {
				cfg.Session.CookieExpire = "invalid"
			},
			expectedField: "session.cookie_expire",
			expectedError: "invalid duration format",
		},
		{
			name: "Invalid store type",
			modifyConfig: func(cfg *config.Config) {
				cfg.Session.StoreType = "invalid"
			},
			expectedField: "session.store_type",
			expectedError: "must be 'memory' or 'redis'",
		},
		{
			name: "Redis store without address",
			modifyConfig: func(cfg *config.Config) {
				cfg.Session.StoreType = "redis"
				cfg.Session.Redis.Addr = ""
			},
			expectedField: "session.redis.addr",
			expectedError: "required when using redis store",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.modifyConfig(cfg)

			errs := ValidateConfig(cfg)
			if errs == nil {
				t.Fatal("Expected validation errors, got nil")
			}

			found := false
			for _, err := range errs {
				if err.Field == tt.expectedField {
					found = true
					if !strings.Contains(err.Message, tt.expectedError) {
						t.Errorf("Expected error message to contain '%s', got: %s", tt.expectedError, err.Message)
					}
				}
			}
			if !found {
				t.Errorf("Expected validation error for %s", tt.expectedField)
			}
		})
	}
}

func TestValidateConfig_OAuth2Validation(t *testing.T) {
	tests := []struct {
		name          string
		modifyConfig  func(*config.Config)
		expectedField string
		expectedError string
	}{
		{
			name: "No available providers (all disabled)",
			modifyConfig: func(cfg *config.Config) {
				cfg.OAuth2.Providers[0].Disabled = true
			},
			expectedField: "oauth2.providers / email_auth.enabled",
			expectedError: "at least one authentication method must be enabled",
		},
		{
			name: "Missing provider name",
			modifyConfig: func(cfg *config.Config) {
				cfg.OAuth2.Providers[0].Name = ""
			},
			expectedField: "oauth2.providers[0].name",
			expectedError: "required",
		},
		{
			name: "Missing client ID",
			modifyConfig: func(cfg *config.Config) {
				cfg.OAuth2.Providers[0].ClientID = ""
			},
			expectedField: "oauth2.providers[0].client_id",
			expectedError: "required",
		},
		{
			name: "Missing client secret",
			modifyConfig: func(cfg *config.Config) {
				cfg.OAuth2.Providers[0].ClientSecret = ""
			},
			expectedField: "oauth2.providers[0].client_secret",
			expectedError: "required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.modifyConfig(cfg)

			errs := ValidateConfig(cfg)
			if errs == nil {
				t.Fatal("Expected validation errors, got nil")
			}

			found := false
			for _, err := range errs {
				if err.Field == tt.expectedField {
					found = true
					if !strings.Contains(err.Message, tt.expectedError) {
						t.Errorf("Expected error message to contain '%s', got: %s", tt.expectedError, err.Message)
					}
				}
			}
			if !found {
				t.Errorf("Expected validation error for %s, got errors: %v", tt.expectedField, errs)
			}
		})
	}
}

func TestValidateConfig_CustomOAuth2Provider(t *testing.T) {
	cfg := validConfig()
	cfg.OAuth2.Providers = []config.OAuth2Provider{
		{
			Name:         "custom",
			Type:         "custom",
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			// Missing required custom provider URLs
		},
	}

	errs := ValidateConfig(cfg)
	if errs == nil {
		t.Fatal("Expected validation errors, got nil")
	}

	expectedFields := []string{
		"oauth2.providers[0].auth_url",
		"oauth2.providers[0].token_url",
		"oauth2.providers[0].userinfo_url",
	}

	for _, field := range expectedFields {
		found := false
		for _, err := range errs {
			if err.Field == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected validation error for %s", field)
		}
	}
}

func TestValidateConfig_EmailAuthValidation(t *testing.T) {
	tests := []struct {
		name          string
		modifyConfig  func(*config.Config)
		expectedField string
		expectedError string
	}{
		{
			name: "Email auth enabled without sender type",
			modifyConfig: func(cfg *config.Config) {
				cfg.OAuth2.Providers[0].Disabled = true
				cfg.EmailAuth.Enabled = true
				cfg.EmailAuth.SenderType = ""
			},
			expectedField: "email_auth.sender_type",
			expectedError: "required when email authentication is enabled",
		},
		{
			name: "Invalid sender type",
			modifyConfig: func(cfg *config.Config) {
				cfg.EmailAuth.Enabled = true
				cfg.EmailAuth.SenderType = "invalid"
			},
			expectedField: "email_auth.sender_type",
			expectedError: "must be 'smtp' or 'sendgrid'",
		},
		{
			name: "SMTP without host",
			modifyConfig: func(cfg *config.Config) {
				cfg.EmailAuth.Enabled = true
				cfg.EmailAuth.SenderType = "smtp"
				cfg.EmailAuth.SMTP.Host = ""
			},
			expectedField: "email_auth.smtp.host",
			expectedError: "required when using SMTP sender",
		},
		{
			name: "SMTP without port",
			modifyConfig: func(cfg *config.Config) {
				cfg.EmailAuth.Enabled = true
				cfg.EmailAuth.SenderType = "smtp"
				cfg.EmailAuth.SMTP.Host = "smtp.example.com"
				cfg.EmailAuth.SMTP.Port = 0
			},
			expectedField: "email_auth.smtp.port",
			expectedError: "required when using SMTP sender",
		},
		{
			name: "SMTP without from address",
			modifyConfig: func(cfg *config.Config) {
				cfg.EmailAuth.Enabled = true
				cfg.EmailAuth.SenderType = "smtp"
				cfg.EmailAuth.SMTP.Host = "smtp.example.com"
				cfg.EmailAuth.SMTP.Port = 587
				cfg.EmailAuth.SMTP.From = ""
			},
			expectedField: "email_auth.smtp.from",
			expectedError: "required when using SMTP sender",
		},
		{
			name: "SendGrid without API key",
			modifyConfig: func(cfg *config.Config) {
				cfg.EmailAuth.Enabled = true
				cfg.EmailAuth.SenderType = "sendgrid"
				cfg.EmailAuth.SendGrid.APIKey = ""
			},
			expectedField: "email_auth.sendgrid.api_key",
			expectedError: "required when using SendGrid sender",
		},
		{
			name: "SendGrid without from address",
			modifyConfig: func(cfg *config.Config) {
				cfg.EmailAuth.Enabled = true
				cfg.EmailAuth.SenderType = "sendgrid"
				cfg.EmailAuth.SendGrid.APIKey = "test-api-key"
				cfg.EmailAuth.SendGrid.From = ""
			},
			expectedField: "email_auth.sendgrid.from",
			expectedError: "required when using SendGrid sender",
		},
		{
			name: "Invalid token expiration",
			modifyConfig: func(cfg *config.Config) {
				cfg.EmailAuth.Enabled = true
				cfg.EmailAuth.SenderType = "smtp"
				cfg.EmailAuth.SMTP.Host = "smtp.example.com"
				cfg.EmailAuth.SMTP.Port = 587
				cfg.EmailAuth.SMTP.From = "test@example.com"
				cfg.EmailAuth.Token.Expire = "invalid"
			},
			expectedField: "email_auth.token.expire",
			expectedError: "invalid duration format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.modifyConfig(cfg)

			errs := ValidateConfig(cfg)

			// If expectedField is empty, we expect no errors
			if tt.expectedField == "" {
				if errs != nil {
					t.Errorf("Expected no validation errors, got: %v", errs)
				}
				return
			}

			// Otherwise, we expect an error
			if errs == nil {
				t.Fatal("Expected validation errors, got nil")
			}

			found := false
			for _, err := range errs {
				if err.Field == tt.expectedField {
					found = true
					if !strings.Contains(err.Message, tt.expectedError) {
						t.Errorf("Expected error message to contain '%s', got: %s", tt.expectedError, err.Message)
					}
				}
			}
			if !found {
				t.Errorf("Expected validation error for %s, got errors: %v", tt.expectedField, errs)
			}
		})
	}
}

func TestValidationErrors_Error(t *testing.T) {
	errs := ValidationErrors{
		{Field: "field1", Message: "error1"},
		{Field: "field2", Message: "error2"},
	}

	errMsg := errs.Error()

	if !strings.Contains(errMsg, "configuration validation failed") {
		t.Error("Expected error message to contain 'configuration validation failed'")
	}
	if !strings.Contains(errMsg, "field1") || !strings.Contains(errMsg, "error1") {
		t.Error("Expected error message to contain field1 and error1")
	}
	if !strings.Contains(errMsg, "field2") || !strings.Contains(errMsg, "error2") {
		t.Error("Expected error message to contain field2 and error2")
	}
}

func TestValidateConfig_MultipleErrors(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{
			Name: "", // Missing
		},
		Session: config.SessionConfig{
			CookieName:   "_oauth2_proxy",
			CookieSecret: "short", // Too short
			CookieExpire: "invalid", // Invalid format
		},
		OAuth2: config.OAuth2Config{
			Providers: []config.OAuth2Provider{}, // No providers
		},
	}

	errs := ValidateConfig(cfg)
	if errs == nil {
		t.Fatal("Expected validation errors, got nil")
	}

	// Should have multiple errors (excluding proxy.upstream which is validated by config.Validate())
	if len(errs) < 3 {
		t.Errorf("Expected at least 3 validation errors, got %d", len(errs))
	}

	// Check that all expected errors are present
	expectedFields := []string{
		"service.name",
		"session.cookie_secret",
		"session.cookie_expire",
	}

	for _, expectedField := range expectedFields {
		found := false
		for _, err := range errs {
			if err.Field == expectedField {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected validation error for %s", expectedField)
		}
	}
}
