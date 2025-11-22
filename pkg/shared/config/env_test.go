package config

import (
	"os"
	"reflect"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		envVars map[string]string
		want    string
	}{
		{
			name:    "simple variable substitution",
			input:   "host: ${DB_HOST}",
			envVars: map[string]string{"DB_HOST": "localhost"},
			want:    "host: localhost",
		},
		{
			name:    "variable with default value - env set",
			input:   "port: ${DB_PORT:-5432}",
			envVars: map[string]string{"DB_PORT": "3306"},
			want:    "port: 3306",
		},
		{
			name:    "variable with default value - env not set",
			input:   "port: ${DB_PORT:-5432}",
			envVars: map[string]string{},
			want:    "port: 5432",
		},
		{
			name:    "variable with default value - env empty",
			input:   "port: ${DB_PORT:-5432}",
			envVars: map[string]string{"DB_PORT": ""},
			want:    "port: 5432",
		},
		{
			name:    "variable without default - env not set",
			input:   "host: ${DB_HOST}",
			envVars: map[string]string{},
			want:    "host: ",
		},
		{
			name:    "multiple variables",
			input:   "host: ${DB_HOST}, port: ${DB_PORT}",
			envVars: map[string]string{"DB_HOST": "localhost", "DB_PORT": "5432"},
			want:    "host: localhost, port: 5432",
		},
		{
			name:    "multiple variables with defaults",
			input:   "host: ${DB_HOST:-localhost}, port: ${DB_PORT:-5432}",
			envVars: map[string]string{},
			want:    "host: localhost, port: 5432",
		},
		{
			name:    "mixed variables - some set, some with defaults",
			input:   "host: ${DB_HOST}, port: ${DB_PORT:-5432}, user: ${DB_USER:-admin}",
			envVars: map[string]string{"DB_HOST": "mydb.com"},
			want:    "host: mydb.com, port: 5432, user: admin",
		},
		{
			name:    "no variables",
			input:   "host: localhost",
			envVars: map[string]string{},
			want:    "host: localhost",
		},
		{
			name:    "empty string",
			input:   "",
			envVars: map[string]string{},
			want:    "",
		},
		{
			name:    "default value with special characters",
			input:   "url: ${API_URL:-https://api.example.com:8080/v1}",
			envVars: map[string]string{},
			want:    "url: https://api.example.com:8080/v1",
		},
		{
			name:    "default value with spaces",
			input:   "message: ${GREETING:-Hello World}",
			envVars: map[string]string{},
			want:    "message: Hello World",
		},
		{
			name:    "default value empty string",
			input:   "value: ${EMPTY:-}",
			envVars: map[string]string{},
			want:    "value: ",
		},
		{
			name:    "variable names with underscores and numbers",
			input:   "${VAR_1} ${VAR_2_TEST} ${_PRIVATE}",
			envVars: map[string]string{"VAR_1": "a", "VAR_2_TEST": "b", "_PRIVATE": "c"},
			want:    "a b c",
		},
		{
			name:    "YAML-like config",
			input:   "session:\n  cookie_secret: ${COOKIE_SECRET}\noauth2:\n  providers:\n    - client_id: ${GOOGLE_CLIENT_ID}\n      client_secret: ${GOOGLE_CLIENT_SECRET}",
			envVars: map[string]string{"COOKIE_SECRET": "mysecret", "GOOGLE_CLIENT_ID": "id123", "GOOGLE_CLIENT_SECRET": "secret456"},
			want:    "session:\n  cookie_secret: mysecret\noauth2:\n  providers:\n    - client_id: id123\n      client_secret: secret456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for k := range tt.envVars {
				_ = os.Unsetenv(k)
			}

			// Set test environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			// Clean up after test
			defer func() {
				for k := range tt.envVars {
					_ = os.Unsetenv(k)
				}
			}()

			got := ExpandEnv(tt.input)
			if got != tt.want {
				t.Errorf("ExpandEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandEnvBytes(t *testing.T) {
	_ = os.Setenv("TEST_VAR", "test_value")
	defer func() { _ = os.Unsetenv("TEST_VAR") }()

	input := []byte("value: ${TEST_VAR}")
	want := []byte("value: test_value")

	got := ExpandEnvBytes(input)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExpandEnvBytes() = %q, want %q", got, want)
	}
}

func TestExtractEnvVars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single variable",
			input: "${VAR1}",
			want:  []string{"VAR1"},
		},
		{
			name:  "multiple variables",
			input: "${VAR1} ${VAR2} ${VAR3}",
			want:  []string{"VAR1", "VAR2", "VAR3"},
		},
		{
			name:  "duplicate variables",
			input: "${VAR1} ${VAR1} ${VAR2}",
			want:  []string{"VAR1", "VAR2"},
		},
		{
			name:  "variables with defaults",
			input: "${VAR1:-default} ${VAR2}",
			want:  []string{"VAR1", "VAR2"},
		},
		{
			name:  "no variables",
			input: "no variables here",
			want:  []string{},
		},
		{
			name:  "empty string",
			input: "",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractEnvVars(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractEnvVars() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		envVars map[string]string
		want    []string
	}{
		{
			name:    "all variables set",
			input:   "${VAR1} ${VAR2}",
			envVars: map[string]string{"VAR1": "value1", "VAR2": "value2"},
			want:    []string{},
		},
		{
			name:    "some variables missing",
			input:   "${VAR1} ${VAR2}",
			envVars: map[string]string{"VAR1": "value1"},
			want:    []string{"VAR2"},
		},
		{
			name:    "variables with defaults not required",
			input:   "${VAR1} ${VAR2:-default}",
			envVars: map[string]string{},
			want:    []string{"VAR1"},
		},
		{
			name:    "all variables have defaults",
			input:   "${VAR1:-default1} ${VAR2:-default2}",
			envVars: map[string]string{},
			want:    []string{},
		},
		{
			name:    "no variables",
			input:   "no variables",
			envVars: map[string]string{},
			want:    []string{},
		},
		{
			name:    "duplicate missing variables",
			input:   "${VAR1} ${VAR1} ${VAR2}",
			envVars: map[string]string{"VAR2": "value"},
			want:    []string{"VAR1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for k := range tt.envVars {
				_ = os.Unsetenv(k)
			}
			_ = os.Unsetenv("VAR1")
			_ = os.Unsetenv("VAR2")

			// Set test environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			// Clean up after test
			defer func() {
				for k := range tt.envVars {
					_ = os.Unsetenv(k)
				}
				_ = os.Unsetenv("VAR1")
				_ = os.Unsetenv("VAR2")
			}()

			got := ValidateEnvVars(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateEnvVars() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplaceEnvVarsForDisplay(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		envVars map[string]string
		want    string
	}{
		{
			name:    "mask sensitive variables",
			input:   "password: ${DB_PASSWORD}",
			envVars: map[string]string{"DB_PASSWORD": "secret123"},
			want:    "password: ***",
		},
		{
			name:    "show non-sensitive variables",
			input:   "host: ${DB_HOST}",
			envVars: map[string]string{"DB_HOST": "localhost"},
			want:    "host: localhost",
		},
		{
			name:    "mixed sensitive and non-sensitive",
			input:   "host: ${DB_HOST}, password: ${DB_PASSWORD}",
			envVars: map[string]string{"DB_HOST": "localhost", "DB_PASSWORD": "secret123"},
			want:    "host: localhost, password: ***",
		},
		{
			name:    "sensitive keywords - api_key",
			input:   "api_key: ${API_KEY}",
			envVars: map[string]string{"API_KEY": "key123"},
			want:    "api_key: ***",
		},
		{
			name:    "sensitive keywords - token",
			input:   "token: ${AUTH_TOKEN}",
			envVars: map[string]string{"AUTH_TOKEN": "token123"},
			want:    "token: ***",
		},
		{
			name:    "default value shown for missing vars",
			input:   "port: ${DB_PORT:-5432}",
			envVars: map[string]string{},
			want:    "port: 5432",
		},
		{
			name:    "sensitive default value - still masked",
			input:   "secret: ${SECRET:-default_secret}",
			envVars: map[string]string{"SECRET": "actual_secret"},
			want:    "secret: ***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for k := range tt.envVars {
				_ = os.Unsetenv(k)
			}

			// Set test environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			// Clean up after test
			defer func() {
				for k := range tt.envVars {
					_ = os.Unsetenv(k)
				}
			}()

			got := ReplaceEnvVarsForDisplay(tt.input)
			if got != tt.want {
				t.Errorf("ReplaceEnvVarsForDisplay() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsValidVarName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid uppercase", "VAR", true},
		{"valid lowercase", "var", true},
		{"valid with underscore", "VAR_NAME", true},
		{"valid with numbers", "VAR123", true},
		{"valid starting with underscore", "_PRIVATE", true},
		{"invalid starting with number", "1VAR", false},
		{"invalid with dash", "VAR-NAME", false},
		{"invalid with dot", "VAR.NAME", false},
		{"invalid with space", "VAR NAME", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidVarName(tt.input)
			if got != tt.want {
				t.Errorf("isValidVarName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsSensitiveVar(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"password", "DB_PASSWORD", true},
		{"secret", "COOKIE_SECRET", true},
		{"key", "API_KEY", true},
		{"token", "AUTH_TOKEN", true},
		{"apikey", "GOOGLE_APIKEY", true},
		{"credential", "AWS_CREDENTIAL", true},
		{"private", "PRIVATE_KEY", true},
		{"non-sensitive", "DB_HOST", false},
		{"non-sensitive with numbers", "VAR123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSensitiveVar(tt.input)
			if got != tt.want {
				t.Errorf("isSensitiveVar(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
