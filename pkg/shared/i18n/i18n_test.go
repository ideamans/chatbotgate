package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTranslator_T(t *testing.T) {
	translator := NewTranslator()

	tests := []struct {
		name     string
		lang     Language
		key      string
		expected string
	}{
		{
			name:     "English translation exists",
			lang:     English,
			key:      "login.title",
			expected: "Login",
		},
		{
			name:     "Japanese translation exists",
			lang:     Japanese,
			key:      "login.title",
			expected: "ログイン",
		},
		{
			name:     "Fallback to English",
			lang:     "fr",
			key:      "login.title",
			expected: "Login",
		},
		{
			name:     "Key not found - return key",
			lang:     English,
			key:      "nonexistent.key",
			expected: "nonexistent.key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translator.T(tt.lang, tt.key)
			if result != tt.expected {
				t.Errorf("T(%s, %s) = %s, want %s", tt.lang, tt.key, result, tt.expected)
			}
		})
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name           string
		queryParam     string
		cookieValue    string
		acceptLanguage string
		expected       Language
	}{
		{
			name:       "Query parameter takes precedence",
			queryParam: "ja",
			expected:   Japanese,
		},
		{
			name:        "Cookie when no query param",
			cookieValue: "ja",
			expected:    Japanese,
		},
		{
			name:           "Accept-Language header",
			acceptLanguage: "ja-JP,ja;q=0.9,en;q=0.8",
			expected:       Japanese,
		},
		{
			name:           "Accept-Language with English",
			acceptLanguage: "en-US,en;q=0.9",
			expected:       English,
		},
		{
			name:     "Default when nothing specified",
			expected: DefaultLanguage,
		},
		{
			name:       "Normalize language code",
			queryParam: "JA",
			expected:   Japanese,
		},
		{
			name:       "Unknown language defaults to English",
			queryParam: "fr",
			expected:   English,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			// Set query parameter
			if tt.queryParam != "" {
				q := req.URL.Query()
				q.Set("lang", tt.queryParam)
				req.URL.RawQuery = q.Encode()
			}

			// Set cookie
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  "lang",
					Value: tt.cookieValue,
				})
			}

			// Set Accept-Language header
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}

			result := DetectLanguage(req)
			if result != tt.expected {
				t.Errorf("DetectLanguage() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected Language
	}{
		{"en", English},
		{"EN", English},
		{"en-US", English},
		{"en-GB", English},
		{"ja", Japanese},
		{"JA", Japanese},
		{"ja-JP", Japanese},
		{"fr", English}, // Unknown defaults to English
		{"de", English},
		{"", English},
		{"   en   ", English},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeLanguage(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeLanguage(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAllTranslationsExist(t *testing.T) {
	translator := NewTranslator()

	// Get all keys from English (reference language)
	englishKeys := make(map[string]bool)
	for key := range translator.translations[English] {
		englishKeys[key] = true
	}

	// Check that all English keys have Japanese translations
	for key := range englishKeys {
		if _, ok := translator.translations[Japanese][key]; !ok {
			t.Errorf("Missing Japanese translation for key: %s", key)
		}
	}

	// Check that all Japanese keys exist in English (to detect orphaned translations)
	for key := range translator.translations[Japanese] {
		if _, ok := englishKeys[key]; !ok {
			t.Errorf("Japanese translation exists but English translation missing for key: %s", key)
		}
	}
}
