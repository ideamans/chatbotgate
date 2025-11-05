package rules

import (
	"regexp"
	"strings"

	"github.com/gobwas/glob"
)

// Matcher is an interface for path matching
type Matcher interface {
	// Match returns true if the path matches
	Match(path string) bool
}

// ExactMatcher matches exact paths
type ExactMatcher struct {
	path string
}

func NewExactMatcher(path string) *ExactMatcher {
	return &ExactMatcher{path: path}
}

func (m *ExactMatcher) Match(path string) bool {
	return path == m.path
}

// PrefixMatcher matches path prefixes
type PrefixMatcher struct {
	prefix string
}

func NewPrefixMatcher(prefix string) *PrefixMatcher {
	return &PrefixMatcher{prefix: prefix}
}

func (m *PrefixMatcher) Match(path string) bool {
	return strings.HasPrefix(path, m.prefix)
}

// RegexMatcher matches paths using regular expressions
type RegexMatcher struct {
	pattern *regexp.Regexp
}

func NewRegexMatcher(pattern string) (*RegexMatcher, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexMatcher{pattern: re}, nil
}

func (m *RegexMatcher) Match(path string) bool {
	return m.pattern.MatchString(path)
}

// MinimatchMatcher matches paths using glob/minimatch patterns
type MinimatchMatcher struct {
	glob glob.Glob
}

func NewMinimatchMatcher(pattern string) (*MinimatchMatcher, error) {
	g, err := glob.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &MinimatchMatcher{glob: g}, nil
}

func (m *MinimatchMatcher) Match(path string) bool {
	return m.glob.Match(path)
}

// AllMatcher matches all paths
type AllMatcher struct{}

func NewAllMatcher() *AllMatcher {
	return &AllMatcher{}
}

func (m *AllMatcher) Match(path string) bool {
	return true
}
