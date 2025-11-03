package passthrough

import (
	"testing"

	"github.com/ideamans/chatbotgate/pkg/config"
)

func TestMatcher_Prefix(t *testing.T) {
	cfg := &config.PassthroughConfig{
		Prefix: []string{
			"/embed.js",
			"/public/",
			"/static/",
		},
	}

	m := NewMatcher(cfg)

	tests := []struct {
		path     string
		expected bool
	}{
		{"/embed.js", true},
		{"/embed.js?v=123", true},
		{"/public/image.png", true},
		{"/public/css/style.css", true},
		{"/static/", true},
		{"/static/file.txt", true},
		{"/api/users", false},
		{"/embeddings", false},
		{"/publicapi", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := m.Match(tt.path)
			if result != tt.expected {
				t.Errorf("Match(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestMatcher_Regex(t *testing.T) {
	cfg := &config.PassthroughConfig{
		Regex: []string{
			`^/api/public/.*$`,
			`\.js$`,
			`^/v\d+/.*$`,
		},
	}

	m := NewMatcher(cfg)

	if m.HasErrors() {
		t.Fatalf("Unexpected compilation errors: %v", m.Errors())
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/public/data", true},
		{"/api/public/users/123", true},
		{"/api/private/data", false},
		{"/script.js", true},
		{"/static/bundle.js", true},
		{"/style.css", false},
		{"/v1/api", true},
		{"/v2/users", true},
		{"/v10/data", true},
		{"/version/api", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := m.Match(tt.path)
			if result != tt.expected {
				t.Errorf("Match(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestMatcher_Minimatch(t *testing.T) {
	cfg := &config.PassthroughConfig{
		Minimatch: []string{
			"/**/*.js",
			"/static/**",
			"/assets/**/*.{css,js}",
		},
	}

	m := NewMatcher(cfg)

	tests := []struct {
		path     string
		expected bool
	}{
		// /**/*.js pattern
		{"/script.js", true},
		{"/app.js", true},
		{"/static/bundle.js", true},
		{"/path/to/file.js", true},
		{"/style.css", false},

		// /static/** pattern
		{"/static/file.txt", true},
		{"/static/images/photo.jpg", true},
		{"/static/", true},
		{"/public/file.txt", false},

		// Complex pattern (not fully supported by simple glob)
		// These may not work as expected with the basic glob implementation
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := m.Match(tt.path)
			if result != tt.expected {
				t.Errorf("Match(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestMatcher_Combined(t *testing.T) {
	cfg := &config.PassthroughConfig{
		Prefix: []string{
			"/embed.js",
		},
		Regex: []string{
			`^/api/public/.*$`,
		},
		Minimatch: []string{
			"/**/*.css",
		},
	}

	m := NewMatcher(cfg)

	tests := []struct {
		path     string
		expected bool
		reason   string
	}{
		{"/embed.js", true, "prefix match"},
		{"/api/public/data", true, "regex match"},
		{"/styles/main.css", true, "glob match"},
		{"/api/private/data", false, "no match"},
		{"/script.js", false, "no match"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := m.Match(tt.path)
			if result != tt.expected {
				t.Errorf("Match(%q) = %v, want %v (%s)", tt.path, result, tt.expected, tt.reason)
			}
		})
	}
}

func TestMatcher_Empty(t *testing.T) {
	// Empty config should not match anything
	cfg := &config.PassthroughConfig{}
	m := NewMatcher(cfg)

	paths := []string{
		"/",
		"/api/users",
		"/public/file.txt",
		"/embed.js",
	}

	for _, path := range paths {
		if m.Match(path) {
			t.Errorf("Empty config matched %q, expected no match", path)
		}
	}
}

func TestMatcher_Nil(t *testing.T) {
	// Nil config should not match anything
	m := NewMatcher(nil)

	paths := []string{
		"/",
		"/api/users",
		"/public/file.txt",
	}

	for _, path := range paths {
		if m.Match(path) {
			t.Errorf("Nil config matched %q, expected no match", path)
		}
	}
}

func TestMatcher_InvalidRegex(t *testing.T) {
	cfg := &config.PassthroughConfig{
		Regex: []string{
			`[invalid(`,  // Invalid regex
			`^/valid/.*$`, // Valid regex
		},
	}

	m := NewMatcher(cfg)

	if !m.HasErrors() {
		t.Error("Expected compilation errors for invalid regex")
	}

	// Valid regex should still work
	if !m.Match("/valid/path") {
		t.Error("Valid regex should still match")
	}

	// Invalid regex should be skipped (not panic)
	m.Match("/invalid")
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		// ** patterns
		{"/**/*.js", "/app.js", true},
		{"/**/*.js", "/path/to/app.js", true},
		{"/static/**", "/static/file.txt", true},
		{"/static/**", "/static/", true},
		{"/static/**", "/public/file.txt", false},
		{"/api/**/users", "/api/users", true},
		{"/api/**/users", "/api/v1/users", true},

		// * patterns (using path.Match)
		{"/*.js", "/app.js", true},
		{"/*.js", "/path/app.js", false},
		{"/static/*", "/static/file.txt", true},

		// ? patterns
		{"/file?.txt", "/file1.txt", true},
		{"/file?.txt", "/file12.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			result := matchGlob(tt.pattern, tt.path)
			if result != tt.expected {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}
