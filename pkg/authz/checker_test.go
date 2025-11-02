package authz

import (
	"testing"

	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
)

func TestEmailChecker_IsAllowed(t *testing.T) {
	cfg := config.AuthorizationConfig{
		Allowed: []string{
			"user@example.com",
			"admin@test.org",
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
		Allowed: []string{},
	}

	checker := NewEmailChecker(cfg)

	// When no whitelist is configured, RequiresEmail should return false
	if checker.RequiresEmail() {
		t.Error("RequiresEmail() should return false with empty config")
	}

	// When no whitelist is configured, all emails should be allowed
	if !checker.IsAllowed("user@example.com") {
		t.Error("expected all emails to be allowed with empty config (no whitelist)")
	}

	if !checker.IsAllowed("any@domain.com") {
		t.Error("expected all emails to be allowed with empty config (no whitelist)")
	}

	// Even empty email should be allowed when no whitelist is configured
	if !checker.IsAllowed("") {
		t.Error("expected even empty email to be allowed with empty config (no whitelist)")
	}
}

func TestNewEmailChecker(t *testing.T) {
	cfg := config.AuthorizationConfig{
		Allowed: []string{
			"User@Example.COM",
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

func TestEmailChecker_RequiresEmail(t *testing.T) {
	tests := []struct {
		name   string
		config config.AuthorizationConfig
		want   bool
	}{
		{
			name: "no whitelist - email not required",
			config: config.AuthorizationConfig{
				Allowed: []string{},
			},
			want: false,
		},
		{
			name: "only allowed emails - email required",
			config: config.AuthorizationConfig{
				Allowed: []string{"user@example.com"},
			},
			want: true,
		},
		{
			name: "only allowed domains - email required",
			config: config.AuthorizationConfig{
				Allowed: []string{"@example.com"},
			},
			want: true,
		},
		{
			name: "both emails and domains - email required",
			config: config.AuthorizationConfig{
				Allowed: []string{
					"user@example.com",
					"@company.com",
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewEmailChecker(tt.config)
			got := checker.RequiresEmail()
			if got != tt.want {
				t.Errorf("RequiresEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}
