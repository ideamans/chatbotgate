package forwarding

import (
	"testing"
)

func TestGetValueByPath(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		path     string
		expected string
	}{
		{
			name: "simple string field",
			data: map[string]interface{}{
				"name": "John Doe",
			},
			path:     "name",
			expected: "John Doe",
		},
		{
			name: "nested string field",
			data: map[string]interface{}{
				"secrets": map[string]interface{}{
					"access_token": "secret-token-123",
				},
			},
			path:     "secrets.access_token",
			expected: "secret-token-123",
		},
		{
			name: "deeply nested field",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"profile": map[string]interface{}{
						"settings": map[string]interface{}{
							"theme": "dark",
						},
					},
				},
			},
			path:     "user.profile.settings.theme",
			expected: "dark",
		},
		{
			name: "numeric value",
			data: map[string]interface{}{
				"count": 42,
			},
			path:     "count",
			expected: "42",
		},
		{
			name: "boolean value - true",
			data: map[string]interface{}{
				"enabled": true,
			},
			path:     "enabled",
			expected: "true",
		},
		{
			name: "boolean value - false",
			data: map[string]interface{}{
				"enabled": false,
			},
			path:     "enabled",
			expected: "false",
		},
		{
			name: "float value",
			data: map[string]interface{}{
				"price": 19.99,
			},
			path:     "price",
			expected: "19.99",
		},
		{
			name: "path not found - missing field",
			data: map[string]interface{}{
				"name": "John",
			},
			path:     "email",
			expected: "",
		},
		{
			name: "path not found - missing nested field",
			data: map[string]interface{}{
				"secrets": map[string]interface{}{
					"token": "abc",
				},
			},
			path:     "secrets.access_token",
			expected: "",
		},
		{
			name: "path not found - intermediate value is not a map",
			data: map[string]interface{}{
				"secrets": "not-a-map",
			},
			path:     "secrets.access_token",
			expected: "",
		},
		{
			name: "empty path",
			data: map[string]interface{}{
				"name": "John",
			},
			path:     "",
			expected: "",
		},
		{
			name: "null value",
			data: map[string]interface{}{
				"value": nil,
			},
			path:     "value",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetValueByPath(tt.data, tt.path)
			if result != tt.expected {
				t.Errorf("GetValueByPath(%v, %q) = %q, want %q", tt.data, tt.path, result, tt.expected)
			}
		})
	}
}
