package rules

import "fmt"

// Rule represents a compiled rule with a matcher and action
type Rule struct {
	matcher     Matcher
	action      Action
	description string
}

// Evaluator evaluates path access rules
type Evaluator struct {
	rules []*Rule
}

// NewEvaluator creates a new rule evaluator from configuration
func NewEvaluator(config *Config) (*Evaluator, error) {
	// If no rules specified, use default (require auth for all)
	if config == nil || len(config.Rules) == 0 {
		config = GetDefaultConfig()
	}

	// Validate configuration first
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid rules configuration: %w", err)
	}

	// Compile all rules
	rules := make([]*Rule, 0, len(config.Rules))
	for i, ruleConfig := range config.Rules {
		rule, err := compileRule(&ruleConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to compile rule[%d]: %w", i, err)
		}
		rules = append(rules, rule)
	}

	return &Evaluator{rules: rules}, nil
}

// compileRule compiles a rule configuration into an executable rule
func compileRule(config *RuleConfig) (*Rule, error) {
	var matcher Matcher
	var err error

	// Create matcher based on configuration
	switch {
	case config.Exact != "":
		matcher = NewExactMatcher(config.Exact)
	case config.Prefix != "":
		matcher = NewPrefixMatcher(config.Prefix)
	case config.Regex != "":
		matcher, err = NewRegexMatcher(config.Regex)
		if err != nil {
			return nil, fmt.Errorf("failed to compile regex: %w", err)
		}
	case config.Minimatch != "":
		matcher, err = NewMinimatchMatcher(config.Minimatch)
		if err != nil {
			return nil, fmt.Errorf("failed to compile minimatch pattern: %w", err)
		}
	case config.All != nil && *config.All:
		matcher = NewAllMatcher()
	default:
		// This should not happen if Validate() was called
		return nil, fmt.Errorf("no matcher specified")
	}

	return &Rule{
		matcher:     matcher,
		action:      config.Action,
		description: config.Description,
	}, nil
}

// Evaluate evaluates a path against all rules and returns the action
// Rules are evaluated in order, and the first matching rule determines the action
func (e *Evaluator) Evaluate(path string) Action {
	for _, rule := range e.rules {
		if rule.matcher.Match(path) {
			return rule.action
		}
	}

	// If no rules match, default to requiring authentication
	return ActionAuth
}

// ShouldAllow returns true if the path should be allowed without authentication
func (e *Evaluator) ShouldAllow(path string) bool {
	return e.Evaluate(path) == ActionAllow
}

// ShouldAuth returns true if the path requires authentication
func (e *Evaluator) ShouldAuth(path string) bool {
	return e.Evaluate(path) == ActionAuth
}

// ShouldDeny returns true if the path should be denied (403)
func (e *Evaluator) ShouldDeny(path string) bool {
	return e.Evaluate(path) == ActionDeny
}
