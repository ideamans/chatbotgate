package config

import (
	"os"
	"regexp"
	"strings"
)

// envVarPattern matches ${VAR} or ${VAR:-default}
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-([^}]*))?\}`)

// ExpandEnv replaces environment variable references in the input string
// with their values.
//
// Supported formats:
//   - ${VAR}         - Replaces with the value of VAR, or empty string if not set
//   - ${VAR:-default} - Replaces with the value of VAR, or "default" if VAR is not set or empty
//
// Example:
//
//	input := "host: ${DB_HOST:-localhost}, port: ${DB_PORT:-5432}"
//	output := ExpandEnv(input)
//	// If DB_HOST=mydb.com and DB_PORT is not set:
//	// output = "host: mydb.com, port: 5432"
func ExpandEnv(input string) string {
	return envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name and default value from the match
		parts := envVarPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match // Should not happen, but return original if parsing fails
		}

		varName := parts[1]
		hasDefault := len(parts) >= 4 && parts[2] != ""
		defaultValue := ""
		if hasDefault {
			defaultValue = parts[3]
		}

		// Get environment variable value
		value, exists := os.LookupEnv(varName)

		// Return the value based on existence and default
		if exists && value != "" {
			return value
		}

		// Use default value if provided
		if hasDefault {
			return defaultValue
		}

		// Return empty string if no default and variable doesn't exist
		return ""
	})
}

// ExpandEnvBytes is a convenience wrapper around ExpandEnv for byte slices
// Useful for processing file contents before YAML/JSON unmarshaling
func ExpandEnvBytes(input []byte) []byte {
	return []byte(ExpandEnv(string(input)))
}

// isValidVarName checks if a string is a valid environment variable name
// Variable names must start with a letter or underscore, followed by letters, digits, or underscores
func isValidVarName(name string) bool {
	if len(name) == 0 {
		return false
	}

	// First character must be letter or underscore
	first := name[0]
	if (first < 'A' || first > 'Z') && (first < 'a' || first > 'z') && first != '_' {
		return false
	}

	// Remaining characters must be letter, digit, or underscore
	for i := 1; i < len(name); i++ {
		c := name[i]
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}

	return true
}

// ExtractEnvVars extracts all environment variable names referenced in the input
// This is useful for validation or documentation purposes
func ExtractEnvVars(input string) []string {
	matches := envVarPattern.FindAllStringSubmatch(input, -1)
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, match := range matches {
		if len(match) >= 2 {
			varName := match[1]
			if !seen[varName] {
				seen[varName] = true
				result = append(result, varName)
			}
		}
	}

	return result
}

// ValidateEnvVars checks if all required environment variables are set
// Returns a list of missing variable names
// Variables with default values (${VAR:-default}) are not considered required
func ValidateEnvVars(input string) []string {
	matches := envVarPattern.FindAllStringSubmatch(input, -1)
	seen := make(map[string]bool)
	missing := make([]string, 0)

	for _, match := range matches {
		if len(match) >= 2 {
			varName := match[1]
			hasDefault := len(match) >= 4 && match[2] != ""

			// Skip if already checked or has default value
			if seen[varName] || hasDefault {
				continue
			}

			seen[varName] = true

			// Check if environment variable is set
			value := os.Getenv(varName)
			if value == "" {
				missing = append(missing, varName)
			}
		}
	}

	return missing
}

// ReplaceEnvVarsForDisplay replaces environment variable values with masked strings
// for safe display in logs or error messages
// Example: "password: ${DB_PASSWORD}" -> "password: ***"
func ReplaceEnvVarsForDisplay(input string) string {
	return envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		parts := envVarPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		varName := parts[1]
		value, exists := os.LookupEnv(varName)

		if exists && value != "" {
			// Mask sensitive values
			if isSensitiveVar(varName) {
				return "***"
			}
			return value
		}

		// Show default value if provided
		if len(parts) >= 4 && parts[2] != "" {
			return parts[3]
		}

		return ""
	})
}

// isSensitiveVar checks if a variable name suggests sensitive data
func isSensitiveVar(name string) bool {
	lowerName := strings.ToLower(name)
	sensitiveKeywords := []string{
		"password", "secret", "key", "token", "api_key",
		"apikey", "auth", "credential", "private",
	}

	for _, keyword := range sensitiveKeywords {
		if strings.Contains(lowerName, keyword) {
			return true
		}
	}

	return false
}
