package authz

import (
	"testing"

	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
)

func TestEmailChecker_IsAllowed(t *testing.T) {
	cfg := config.AuthorizationConfig{
		AllowedEmails: []string{
			"user@example.com",
			"admin@test.org",
		},
		AllowedDomains: []string{
			"@ideamans.com",
			"@company.net",
		},
	}

	checker := NewEmailChecker(cfg)

	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{
			name:  "allowed email - exact match",
			email: "user@example.com",
			want:  true,
		},
		{
			name:  "allowed email - case insensitive",
			email: "USER@EXAMPLE.COM",
			want:  true,
		},
		{
			name:  "allowed email - with whitespace",
			email: "  user@example.com  ",
			want:  true,
		},
		{
			name:  "allowed domain",
			email: "anyone@ideamans.com",
			want:  true,
		},
		{
			name:  "allowed domain - case insensitive",
			email: "ANYONE@IDEAMANS.COM",
			want:  true,
		},
		{
			name:  "allowed domain - different user",
			email: "developer@company.net",
			want:  true,
		},
		{
			name:  "not allowed email",
			email: "stranger@unknown.com",
			want:  false,
		},
		{
			name:  "not allowed domain",
			email: "user@notallowed.com",
			want:  false,
		},
		{
			name:  "invalid email format - no @",
			email: "notanemail",
			want:  false,
		},
		{
			name:  "invalid email format - multiple @",
			email: "user@@example.com",
			want:  false,
		},
		{
			name:  "empty email",
			email: "",
			want:  false,
		},
		{
			name:  "subdomain should not match",
			email: "user@sub.ideamans.com",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checker.IsAllowed(tt.email)
			if got != tt.want {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestEmailChecker_EmptyConfig(t *testing.T) {
	cfg := config.AuthorizationConfig{
		AllowedEmails:  []string{},
		AllowedDomains: []string{},
	}

	checker := NewEmailChecker(cfg)

	if checker.IsAllowed("user@example.com") {
		t.Error("expected no emails to be allowed with empty config")
	}
}

func TestNewEmailChecker(t *testing.T) {
	cfg := config.AuthorizationConfig{
		AllowedEmails: []string{
			"User@Example.COM",
		},
		AllowedDomains: []string{
			"@IDEAMANS.COM",
		},
	}

	checker := NewEmailChecker(cfg)

	// Verify that emails are stored in lowercase
	if !checker.IsAllowed("user@example.com") {
		t.Error("email should be case-insensitive")
	}

	// Verify that domains are stored in lowercase
	if !checker.IsAllowed("test@ideamans.com") {
		t.Error("domain should be case-insensitive")
	}
}
