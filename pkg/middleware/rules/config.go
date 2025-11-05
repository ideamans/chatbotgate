package rules

import (
	"fmt"
	"regexp"

	"github.com/gobwas/glob"
)

// Action represents the action to take when a rule matches
type Action string

const (
	ActionAllow Action = "allow" // Allow access without authentication
	ActionAuth  Action = "auth"  // Require authentication
	ActionDeny  Action = "deny"  // Deny access (403)
)

// RuleConfig represents a single rule in the configuration
type RuleConfig struct {
	// Matchers (only one should be specified)
	Exact     string `yaml:"exact,omitempty"`     // Exact path match
	Prefix    string `yaml:"prefix,omitempty"`    // Prefix match
	Regex     string `yaml:"regex,omitempty"`     // Regular expression match
	Minimatch string `yaml:"minimatch,omitempty"` // Glob/minimatch pattern
	All       *bool  `yaml:"all,omitempty"`       // Match all paths (must be true if specified)

	// Action to take when matched
	Action Action `yaml:"action"`

	// Optional description for documentation
	Description string `yaml:"description,omitempty"`
}

// Config represents the rules configuration
type Config struct {
	Rules []RuleConfig `yaml:"rules,omitempty"`
}

// Validate validates the rules configuration
func (c *Config) Validate() error {
	if len(c.Rules) == 0 {
		// No rules specified = default to require auth for all
		return nil
	}

	for i, rule := range c.Rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("rule[%d]: %w", i, err)
		}
	}

	return nil
}

// Validate validates a single rule configuration
func (r *RuleConfig) Validate() error {
	// Check if all: false is explicitly specified (error)
	if r.All != nil && !*r.All {
		return fmt.Errorf("all: false is not allowed (omit the field or use all: true)")
	}

	// Count how many matchers are specified
	matcherCount := 0
	if r.Exact != "" {
		matcherCount++
	}
	if r.Prefix != "" {
		matcherCount++
	}
	if r.Regex != "" {
		matcherCount++
	}
	if r.Minimatch != "" {
		matcherCount++
	}
	if r.All != nil && *r.All {
		matcherCount++
	}

	// Exactly one matcher must be specified
	if matcherCount == 0 {
		return fmt.Errorf("no matcher specified (must specify one of: exact, prefix, regex, minimatch, all)")
	}
	if matcherCount > 1 {
		return fmt.Errorf("multiple matchers specified (only one of exact, prefix, regex, minimatch, all is allowed)")
	}

	// Validate action
	switch r.Action {
	case ActionAllow, ActionAuth, ActionDeny:
		// Valid action
	default:
		return fmt.Errorf("invalid action %q (must be one of: allow, auth, deny)", r.Action)
	}

	// Validate regex syntax if specified
	if r.Regex != "" {
		if _, err := regexp.Compile(r.Regex); err != nil {
			return fmt.Errorf("invalid regex pattern %q: %w", r.Regex, err)
		}
	}

	// Validate minimatch pattern if specified
	if r.Minimatch != "" {
		if _, err := glob.Compile(r.Minimatch); err != nil {
			return fmt.Errorf("invalid minimatch pattern %q: %w", r.Minimatch, err)
		}
	}

	return nil
}

// GetDefaultConfig returns the default rules configuration (require auth for all)
func GetDefaultConfig() *Config {
	allTrue := true
	return &Config{
		Rules: []RuleConfig{
			{
				All:         &allTrue,
				Action:      ActionAuth,
				Description: "Default: require authentication for all paths",
			},
		},
	}
}
