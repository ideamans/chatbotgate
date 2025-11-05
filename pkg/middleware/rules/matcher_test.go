package rules

import (
	"testing"
)

// TestExactMatcher tests the ExactMatcher implementation
func TestExactMatcher(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		{
			name:     "exact match",
			pattern:  "/api/health",
			path:     "/api/health",
			expected: true,
		},
		{
			name:     "exact match with query string",
			pattern:  "/api/health",
			path:     "/api/health?check=true",
			expected: false, // query strings are part of the path
		},
		{
			name:     "no match - different path",
			pattern:  "/api/health",
			path:     "/api/status",
			expected: false,
		},
		{
			name:     "no match - prefix only",
			pattern:  "/api",
			path:     "/api/health",
			expected: false,
		},
		{
			name:     "no match - suffix",
			pattern:  "/health",
			path:     "/api/health",
			expected: false,
		},
		{
			name:     "exact match root path",
			pattern:  "/",
			path:     "/",
			expected: true,
		},
		{
			name:     "no match - root vs subpath",
			pattern:  "/",
			path:     "/api",
			expected: false,
		},
		{
			name:     "case sensitive",
			pattern:  "/API/Health",
			path:     "/api/health",
			expected: false,
		},
		{
			name:     "empty pattern",
			pattern:  "",
			path:     "/api/health",
			expected: false,
		},
		{
			name:     "empty path",
			pattern:  "/api/health",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewExactMatcher(tt.pattern)
			result := matcher.Match(tt.path)
			if result != tt.expected {
				t.Errorf("ExactMatcher.Match(%q) with pattern %q = %v, want %v",
					tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestPrefixMatcher tests the PrefixMatcher implementation
func TestPrefixMatcher(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		{
			name:     "exact match",
			pattern:  "/api/health",
			path:     "/api/health",
			expected: true,
		},
		{
			name:     "prefix match",
			pattern:  "/api",
			path:     "/api/health",
			expected: true,
		},
		{
			name:     "prefix match with trailing slash in pattern",
			pattern:  "/api/",
			path:     "/api/health",
			expected: true,
		},
		{
			name:     "prefix match with trailing slash in path",
			pattern:  "/api",
			path:     "/api/",
			expected: true,
		},
		{
			name:     "no match - different prefix",
			pattern:  "/admin",
			path:     "/api/health",
			expected: false,
		},
		{
			name:     "no match - partial word match",
			pattern:  "/api",
			path:     "/application",
			expected: false,
		},
		{
			name:     "root prefix matches all",
			pattern:  "/",
			path:     "/api/health",
			expected: true,
		},
		{
			name:     "case sensitive",
			pattern:  "/API",
			path:     "/api/health",
			expected: false,
		},
		{
			name:     "empty pattern matches all",
			pattern:  "",
			path:     "/api/health",
			expected: true, // Empty prefix matches everything
		},
		{
			name:     "empty path",
			pattern:  "/api",
			path:     "",
			expected: false,
		},
		{
			name:     "long prefix",
			pattern:  "/api/v1/users/profile/settings",
			path:     "/api/v1/users/profile/settings/notifications",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewPrefixMatcher(tt.pattern)
			result := matcher.Match(tt.path)
			if result != tt.expected {
				t.Errorf("PrefixMatcher.Match(%q) with prefix %q = %v, want %v",
					tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestRegexMatcher tests the RegexMatcher implementation
func TestRegexMatcher(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
		wantErr  bool
	}{
		{
			name:     "simple regex match",
			pattern:  "^/api/.*$",
			path:     "/api/health",
			expected: true,
		},
		{
			name:     "regex with groups",
			pattern:  "^/api/(health|status)$",
			path:     "/api/health",
			expected: true,
		},
		{
			name:     "regex no match",
			pattern:  "^/api/.*$",
			path:     "/admin/users",
			expected: false,
		},
		{
			name:     "regex partial match allowed",
			pattern:  "/api/",
			path:     "/api/health",
			expected: true,
		},
		{
			name:     "regex with character classes",
			pattern:  "^/api/[a-z]+$",
			path:     "/api/health",
			expected: true,
		},
		{
			name:     "regex with character classes no match",
			pattern:  "^/api/[a-z]+$",
			path:     "/api/health123",
			expected: false,
		},
		{
			name:     "regex with numbers",
			pattern:  "^/api/users/[0-9]+$",
			path:     "/api/users/123",
			expected: true,
		},
		{
			name:     "regex case insensitive flag",
			pattern:  "(?i)^/API/.*$",
			path:     "/api/health",
			expected: true,
		},
		{
			name:     "regex with optional group",
			pattern:  "^/api(/public)?/data$",
			path:     "/api/data",
			expected: true,
		},
		{
			name:     "regex with optional group matched",
			pattern:  "^/api(/public)?/data$",
			path:     "/api/public/data",
			expected: true,
		},
		{
			name:     "complex regex with multiple groups",
			pattern:  "^/(admin|api)/v[0-9]+/(users|posts)/[0-9]+$",
			path:     "/api/v1/users/456",
			expected: true,
		},
		{
			name:     "empty path",
			pattern:  "^/api/.*$",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewRegexMatcher(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRegexMatcher(%q) error = %v, wantErr %v", tt.pattern, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			result := matcher.Match(tt.path)
			if result != tt.expected {
				t.Errorf("RegexMatcher.Match(%q) with pattern %q = %v, want %v",
					tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestRegexMatcher_InvalidPattern tests error handling for invalid regex patterns
func TestRegexMatcher_InvalidPattern(t *testing.T) {
	invalidPatterns := []string{
		"[invalid",         // unclosed bracket
		"(invalid",         // unclosed parenthesis
		"(?P<invalid",      // invalid named group
		"*invalid",         // invalid quantifier
		"(?invalid)",       // invalid flag
	}

	for _, pattern := range invalidPatterns {
		t.Run(pattern, func(t *testing.T) {
			_, err := NewRegexMatcher(pattern)
			if err == nil {
				t.Errorf("NewRegexMatcher(%q) expected error for invalid pattern, got nil", pattern)
			}
		})
	}
}

// TestMinimatchMatcher tests the MinimatchMatcher implementation
func TestMinimatchMatcher(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
		wantErr  bool
	}{
		{
			name:     "single star wildcard",
			pattern:  "/api/*.json",
			path:     "/api/data.json",
			expected: true,
		},
		{
			name:     "single star no match",
			pattern:  "/api/*.json",
			path:     "/api/data.xml",
			expected: false,
		},
		{
			name:     "double star wildcard",
			pattern:  "/api/**/*.json",
			path:     "/api/v1/data.json",
			expected: true,
		},
		{
			name:     "double star deep path",
			pattern:  "/api/**/*.json",
			path:     "/api/v1/v2/v3/data.json",
			expected: true,
		},
		{
			name:     "question mark wildcard",
			pattern:  "/api/file?.txt",
			path:     "/api/file1.txt",
			expected: true,
		},
		{
			name:     "question mark no match",
			pattern:  "/api/file?.txt",
			path:     "/api/file12.txt",
			expected: false,
		},
		{
			name:     "brace expansion",
			pattern:  "/api/*.{js,css}",
			path:     "/api/app.js",
			expected: true,
		},
		{
			name:     "brace expansion css",
			pattern:  "/api/*.{js,css}",
			path:     "/api/style.css",
			expected: true,
		},
		{
			name:     "brace expansion no match",
			pattern:  "/api/*.{js,css}",
			path:     "/api/data.json",
			expected: false,
		},
		{
			name:     "complex pattern with multiple wildcards",
			pattern:  "/api/*/v?/**/*.{json,xml}",
			path:     "/api/users/v1/data/file.json",
			expected: true,
		},
		{
			name:     "bracket expression",
			pattern:  "/api/file[0-9].txt",
			path:     "/api/file5.txt",
			expected: true,
		},
		{
			name:     "bracket expression no match",
			pattern:  "/api/file[0-9].txt",
			path:     "/api/fileA.txt",
			expected: false,
		},
		{
			name:     "negation in bracket",
			pattern:  "/api/file[!0-9].txt",
			path:     "/api/fileA.txt",
			expected: true,
		},
		{
			name:     "negation in bracket no match",
			pattern:  "/api/file[!0-9].txt",
			path:     "/api/file5.txt",
			expected: false,
		},
		{
			name:     "exact path without wildcards",
			pattern:  "/api/health",
			path:     "/api/health",
			expected: true,
		},
		{
			name:     "exact path no match",
			pattern:  "/api/health",
			path:     "/api/status",
			expected: false,
		},
		{
			name:     "empty path",
			pattern:  "/api/**",
			path:     "",
			expected: false,
		},
		{
			name:     "root with double star",
			pattern:  "/**",
			path:     "/api/health",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewMinimatchMatcher(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMinimatchMatcher(%q) error = %v, wantErr %v", tt.pattern, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			result := matcher.Match(tt.path)
			if result != tt.expected {
				t.Errorf("MinimatchMatcher.Match(%q) with pattern %q = %v, want %v",
					tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestMinimatchMatcher_InvalidPattern tests error handling for invalid glob patterns
func TestMinimatchMatcher_InvalidPattern(t *testing.T) {
	invalidPatterns := []string{
		"[invalid", // unclosed bracket
	}

	for _, pattern := range invalidPatterns {
		t.Run(pattern, func(t *testing.T) {
			_, err := NewMinimatchMatcher(pattern)
			if err == nil {
				t.Errorf("NewMinimatchMatcher(%q) expected error for invalid pattern, got nil", pattern)
			}
		})
	}
}

// TestAllMatcher tests the AllMatcher implementation
func TestAllMatcher(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "matches any path",
			path:     "/api/health",
			expected: true,
		},
		{
			name:     "matches root",
			path:     "/",
			expected: true,
		},
		{
			name:     "matches empty path",
			path:     "",
			expected: true,
		},
		{
			name:     "matches long path",
			path:     "/api/v1/users/123/profile/settings",
			expected: true,
		},
		{
			name:     "matches with query string",
			path:     "/api/health?check=true",
			expected: true,
		},
		{
			name:     "matches special characters",
			path:     "/api/特殊文字/test",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := &AllMatcher{}
			result := matcher.Match(tt.path)
			if result != tt.expected {
				t.Errorf("AllMatcher.Match(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestMatcherInterface ensures all matchers implement the Matcher interface correctly
func TestMatcherInterface(t *testing.T) {
	// This test verifies that all matcher types satisfy the Matcher interface
	var _ Matcher = &ExactMatcher{}
	var _ Matcher = &PrefixMatcher{}
	var _ Matcher = (*RegexMatcher)(nil)
	var _ Matcher = (*MinimatchMatcher)(nil)
	var _ Matcher = &AllMatcher{}
}

// BenchmarkExactMatcher benchmarks the ExactMatcher performance
func BenchmarkExactMatcher(b *testing.B) {
	matcher := NewExactMatcher("/api/health")
	path := "/api/health"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match(path)
	}
}

// BenchmarkPrefixMatcher benchmarks the PrefixMatcher performance
func BenchmarkPrefixMatcher(b *testing.B) {
	matcher := NewPrefixMatcher("/api/")
	path := "/api/health"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match(path)
	}
}

// BenchmarkRegexMatcher benchmarks the RegexMatcher performance
func BenchmarkRegexMatcher(b *testing.B) {
	matcher, _ := NewRegexMatcher("^/api/.*$")
	path := "/api/health"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match(path)
	}
}

// BenchmarkMinimatchMatcher benchmarks the MinimatchMatcher performance
func BenchmarkMinimatchMatcher(b *testing.B) {
	matcher, _ := NewMinimatchMatcher("/api/**/*.json")
	path := "/api/v1/data.json"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match(path)
	}
}

// BenchmarkAllMatcher benchmarks the AllMatcher performance
func BenchmarkAllMatcher(b *testing.B) {
	matcher := &AllMatcher{}
	path := "/api/health"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match(path)
	}
}
