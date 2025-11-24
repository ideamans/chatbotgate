package rules

import (
	"testing"
)

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

func TestEvaluator_BasicRules(t *testing.T) {
	allTrue := true
	config := Config{
		{Prefix: "/static/", Action: ActionAllow},
		{Exact: "/health", Action: ActionAllow},
		{Prefix: "/api/", Action: ActionAuth},
		{Regex: "^/admin/", Action: ActionDeny},
		{All: &allTrue, Action: ActionAuth},
	}

	evaluator, err := NewEvaluator(&config)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	tests := []struct {
		path           string
		expectedAction Action
	}{
		{"/static/css/main.css", ActionAllow},
		{"/static/js/app.js", ActionAllow},
		{"/health", ActionAllow},
		{"/healthz", ActionAuth}, // Not exact match
		{"/api/users", ActionAuth},
		{"/api/", ActionAuth},
		{"/admin/users", ActionDeny},
		{"/admin/settings", ActionDeny},
		{"/", ActionAuth},
		{"/about", ActionAuth},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			action := evaluator.Evaluate(tt.path)
			if action != tt.expectedAction {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.path, action, tt.expectedAction)
			}
		})
	}
}

func TestEvaluator_Minimatch(t *testing.T) {
	allTrue := true
	config := Config{
		{Minimatch: "**/*.{js,css}", Action: ActionAllow},
		{All: &allTrue, Action: ActionAuth},
	}

	evaluator, err := NewEvaluator(&config)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	tests := []struct {
		path           string
		expectedAction Action
	}{
		{"/static/app.js", ActionAllow},
		{"/css/main.css", ActionAllow},
		{"/deep/nested/script.js", ActionAllow},
		{"/index.html", ActionAuth},
		{"/api/users", ActionAuth},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			action := evaluator.Evaluate(tt.path)
			if action != tt.expectedAction {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.path, action, tt.expectedAction)
			}
		})
	}
}

func TestEvaluator_DefaultConfig(t *testing.T) {
	// nil config should use default (require auth for all)
	evaluator, err := NewEvaluator(nil)
	if err != nil {
		t.Fatalf("Failed to create evaluator with nil config: %v", err)
	}

	tests := []string{"/", "/api", "/static/app.js", "/admin"}
	for _, path := range tests {
		action := evaluator.Evaluate(path)
		if action != ActionAuth {
			t.Errorf("Default config: Evaluate(%q) = %v, want %v", path, action, ActionAuth)
		}
	}
}

func TestEvaluator_OrderMatters(t *testing.T) {
	// First matching rule wins
	config := Config{
		{Prefix: "/api/", Action: ActionAuth},
		{Prefix: "/api/public/", Action: ActionAllow}, // This won't match because /api/ matches first
	}

	evaluator, err := NewEvaluator(&config)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	// /api/public/data matches /api/ first, so it requires auth
	action := evaluator.Evaluate("/api/public/data")
	if action != ActionAuth {
		t.Errorf("Evaluate(/api/public/data) = %v, want %v (first rule should win)", action, ActionAuth)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				{Prefix: "/static/", Action: ActionAllow},
			},
			expectError: false,
		},
		{
			name: "no matcher",
			config: Config{
				{Action: ActionAllow},
			},
			expectError: true,
		},
		{
			name: "multiple matchers",
			config: Config{
				{Prefix: "/api/", Regex: "^/api/", Action: ActionAuth},
			},
			expectError: true,
		},
		{
			name: "invalid action",
			config: Config{
				{Prefix: "/api/", Action: "invalid"},
			},
			expectError: true,
		},
		{
			name: "invalid regex",
			config: Config{
				{Regex: "[invalid(", Action: ActionAuth},
			},
			expectError: true,
		},
		{
			name:        "empty rules",
			config:      Config{},
			expectError: false, // Empty rules is OK (uses default)
		},
		{
			name: "all: false explicitly set",
			config: Config{
				{All: boolPtr(false), Action: ActionAuth},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.expectError {
				t.Errorf("Validate() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}

func TestEvaluator_HelperMethods(t *testing.T) {
	allTrue := true
	config := Config{
		{Prefix: "/static/", Action: ActionAllow},
		{Prefix: "/admin/", Action: ActionDeny},
		{All: &allTrue, Action: ActionAuth},
	}

	evaluator, err := NewEvaluator(&config)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	tests := []struct {
		path        string
		shouldAllow bool
		shouldAuth  bool
		shouldDeny  bool
	}{
		{"/static/app.js", true, false, false},
		{"/admin/users", false, false, true},
		{"/api/data", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if evaluator.ShouldAllow(tt.path) != tt.shouldAllow {
				t.Errorf("ShouldAllow(%q) = %v, want %v", tt.path, evaluator.ShouldAllow(tt.path), tt.shouldAllow)
			}
			if evaluator.ShouldAuth(tt.path) != tt.shouldAuth {
				t.Errorf("ShouldAuth(%q) = %v, want %v", tt.path, evaluator.ShouldAuth(tt.path), tt.shouldAuth)
			}
			if evaluator.ShouldDeny(tt.path) != tt.shouldDeny {
				t.Errorf("ShouldDeny(%q) = %v, want %v", tt.path, evaluator.ShouldDeny(tt.path), tt.shouldDeny)
			}
		})
	}
}
