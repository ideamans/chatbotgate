package server

import (
	"errors"
	"strings"
	"testing"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
)

// TestFormatConfigError tests the formatConfigError function
func TestFormatConfigError(t *testing.T) {
	tests := []struct {
		name          string
		component     string
		inputError    error
		checkContains []string
	}{
		{
			name:      "ValidationError with multiple errors",
			component: "middleware",
			inputError: &config.ValidationError{
				Errors: []error{
					errors.New("service name is required"),
					errors.New("cookie secret is required"),
				},
			},
			checkContains: []string{
				"Configuration validation failed for middleware with 2 error(s)",
				"1. service name is required",
				"2. cookie secret is required",
				"Please fix the errors above",
			},
		},
		{
			name:       "ErrServiceNameRequired",
			component:  "middleware",
			inputError: config.ErrServiceNameRequired,
			checkContains: []string{
				"configuration validation error in middleware",
				"service name is required",
				"please check your configuration file",
			},
		},
		{
			name:       "ErrCookieSecretRequired",
			component:  "middleware",
			inputError: config.ErrCookieSecretRequired,
			checkContains: []string{
				"configuration validation error in middleware",
				"cookie secret is required",
			},
		},
		{
			name:       "ErrCookieSecretTooShort",
			component:  "middleware",
			inputError: config.ErrCookieSecretTooShort,
			checkContains: []string{
				"configuration validation error in middleware",
				"cookie secret must be at least 32 characters",
			},
		},
		{
			name:       "ErrNoEnabledProviders",
			component:  "middleware",
			inputError: config.ErrNoEnabledProviders,
			checkContains: []string{
				"configuration validation error in middleware",
				"at least one OAuth2 provider must be enabled",
			},
		},
		{
			name:       "ErrConfigFileNotFound",
			component:  "middleware",
			inputError: config.ErrConfigFileNotFound,
			checkContains: []string{
				"configuration file not found",
				"please create a configuration file",
			},
		},
		{
			name:       "Error with 'validation failed' in message",
			component:  "proxy",
			inputError: errors.New("validation failed: upstream URL is required"),
			checkContains: []string{
				"configuration validation error in proxy",
				"validation failed: upstream URL is required",
				"please check your configuration file",
			},
		},
		{
			name:       "Generic error",
			component:  "proxy",
			inputError: errors.New("some random error"),
			checkContains: []string{
				"failed to initialize proxy",
				"some random error",
			},
		},
		{
			name:       "ErrEncryptionKeyRequired",
			component:  "middleware",
			inputError: config.ErrEncryptionKeyRequired,
			checkContains: []string{
				"configuration validation error in middleware",
				"encryption key is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatConfigError(tt.component, tt.inputError)

			// Check that result is not nil
			if result == nil {
				t.Fatal("Expected error but got nil")
			}

			resultStr := result.Error()

			// Check that all expected strings are contained in the result
			for _, expected := range tt.checkContains {
				if !strings.Contains(resultStr, expected) {
					t.Errorf("Expected error message to contain %q, but got: %s", expected, resultStr)
				}
			}
		})
	}
}
