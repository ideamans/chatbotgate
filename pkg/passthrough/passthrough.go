package passthrough

import (
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/ideamans/chatbotgate/pkg/config"
)

// Matcher checks if a path should bypass authentication
type Matcher struct {
	prefixes       []string
	regexPatterns  []*regexp.Regexp
	globPatterns   []string
	compileErrors  []error
}

// NewMatcher creates a new passthrough matcher from configuration
func NewMatcher(cfg *config.PassthroughConfig) *Matcher {
	if cfg == nil {
		return &Matcher{}
	}

	m := &Matcher{
		prefixes:     cfg.Prefix,
		globPatterns: cfg.Minimatch,
	}

	// Compile regex patterns
	for _, pattern := range cfg.Regex {
		re, err := regexp.Compile(pattern)
		if err != nil {
			m.compileErrors = append(m.compileErrors, err)
			continue
		}
		m.regexPatterns = append(m.regexPatterns, re)
	}

	return m
}

// Match checks if the given path matches any passthrough pattern
func (m *Matcher) Match(requestPath string) bool {
	if m == nil {
		return false
	}

	// Check prefix matches
	for _, prefix := range m.prefixes {
		if strings.HasPrefix(requestPath, prefix) {
			return true
		}
	}

	// Check regex patterns
	for _, re := range m.regexPatterns {
		if re.MatchString(requestPath) {
			return true
		}
	}

	// Check minimatch/glob patterns
	for _, pattern := range m.globPatterns {
		if matchGlob(pattern, requestPath) {
			return true
		}
	}

	return false
}

// HasErrors returns true if there were any compilation errors
func (m *Matcher) HasErrors() bool {
	return len(m.compileErrors) > 0
}

// Errors returns all compilation errors
func (m *Matcher) Errors() []error {
	return m.compileErrors
}

// matchGlob implements glob pattern matching using doublestar library
// Supports:
// - * matches any sequence of non-separator characters
// - ** matches any sequence of characters including separators
// - ? matches any single non-separator character
// - {a,b} matches either a or b (brace expansion)
func matchGlob(pattern, str string) bool {
	matched, err := doublestar.Match(pattern, str)
	return err == nil && matched
}
