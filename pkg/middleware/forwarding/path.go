package forwarding

import (
	"fmt"
	"strings"
)

// GetValueByPath retrieves a value from a nested map using a dot-separated path
// For example, "secrets.access_token" will retrieve data["secrets"]["access_token"]
// Returns the value as a string if found, empty string otherwise
func GetValueByPath(data map[string]interface{}, path string) string {
	if path == "" {
		return ""
	}

	parts := strings.Split(path, ".")
	current := data

	// Navigate through the nested structure
	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return ""
		}

		// If this is the last part, convert to string and return
		if i == len(parts)-1 {
			return toString(value)
		}

		// Otherwise, expect a nested map
		nestedMap, ok := value.(map[string]interface{})
		if !ok {
			return ""
		}
		current = nestedMap
	}

	return ""
}

// toString converts various types to string representation
func toString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return ""
	default:
		// Use fmt.Sprintf for numeric and other types
		return fmt.Sprintf("%v", v)
	}
}
