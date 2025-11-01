package i18n

import (
	"net/http"
	"strings"
)

// Language represents a supported language
type Language string

const (
	// English is the English language
	English Language = "en"
	// Japanese is the Japanese language
	Japanese Language = "ja"
)

// DefaultLanguage is the fallback language
const DefaultLanguage = English

// Theme represents a UI theme
type Theme string

const (
	// ThemeAuto uses system preference
	ThemeAuto Theme = "auto"
	// ThemeLight uses light theme
	ThemeLight Theme = "light"
	// ThemeDark uses dark theme
	ThemeDark Theme = "dark"
)

// DefaultTheme is the fallback theme
const DefaultTheme = ThemeAuto

// Translation represents a translation map
type Translation map[string]string

// Translations holds all language translations
type Translations map[Language]Translation

// Translator provides translation functionality
type Translator struct {
	translations Translations
}

// NewTranslator creates a new translator
func NewTranslator() *Translator {
	return &Translator{
		translations: defaultTranslations,
	}
}

// T translates a key for the given language
func (t *Translator) T(lang Language, key string) string {
	// Try the requested language
	if trans, ok := t.translations[lang]; ok {
		if text, ok := trans[key]; ok {
			return text
		}
	}

	// Fallback to default language
	if trans, ok := t.translations[DefaultLanguage]; ok {
		if text, ok := trans[key]; ok {
			return text
		}
	}

	// Return the key itself as fallback
	return key
}

// DetectLanguage detects the preferred language from HTTP request
func DetectLanguage(r *http.Request) Language {
	// Check query parameter
	if lang := r.URL.Query().Get("lang"); lang != "" {
		return normalizeLanguage(lang)
	}

	// Check cookie
	if cookie, err := r.Cookie("lang"); err == nil {
		return normalizeLanguage(cookie.Value)
	}

	// Check Accept-Language header
	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang != "" {
		// Parse Accept-Language header (simplified)
		langs := strings.Split(acceptLang, ",")
		if len(langs) > 0 {
			// Get first language
			lang := strings.TrimSpace(strings.Split(langs[0], ";")[0])
			return normalizeLanguage(lang)
		}
	}

	return DefaultLanguage
}

// normalizeLanguage normalizes a language code
func normalizeLanguage(lang string) Language {
	lang = strings.ToLower(strings.TrimSpace(lang))

	// Handle language with region (e.g., en-US, ja-JP)
	if len(lang) > 2 {
		lang = lang[:2]
	}

	switch lang {
	case "ja":
		return Japanese
	case "en":
		return English
	default:
		return DefaultLanguage
	}
}

// DetectTheme detects the preferred theme from HTTP request
func DetectTheme(r *http.Request) Theme {
	// Check query parameter
	if theme := r.URL.Query().Get("theme"); theme != "" {
		return normalizeTheme(theme)
	}

	// Check cookie
	if cookie, err := r.Cookie("theme"); err == nil {
		return normalizeTheme(cookie.Value)
	}

	return DefaultTheme
}

// normalizeTheme normalizes a theme string
func normalizeTheme(theme string) Theme {
	theme = strings.ToLower(strings.TrimSpace(theme))

	switch theme {
	case "light":
		return ThemeLight
	case "dark":
		return ThemeDark
	case "auto":
		return ThemeAuto
	default:
		return DefaultTheme
	}
}

// defaultTranslations contains all default translations
var defaultTranslations = Translations{
	English: Translation{
		// Service
		"service.name":        "Multi OAuth2 Proxy",
		"service.description": "Authentication proxy for multiple OAuth2 providers",

		// Login page
		"login.title":           "Login",
		"login.heading":         "Sign In",
		"login.oauth2.heading":  "Login with OAuth2",
		"login.email.link":      "Or login with Email",
		"login.email.heading":   "Login with Email",
		"login.email.label":     "Email Address",
		"login.email.submit":    "Send Login Link",
		"login.back":            "Back to login options",

		// Email auth
		"email.sent.title":   "Check Your Email",
		"email.sent.heading": "Check Your Email",
		"email.sent.message": "If your email address is authorized, you will receive a login link shortly.",
		"email.sent.detail":  "Please check your inbox and click the link to log in.",
		"email.sent.back":    "Back to login",

		"email.invalid.title":   "Invalid Token",
		"email.invalid.heading": "Invalid or Expired Token",
		"email.invalid.message": "The login link is invalid or has already been used.",
		"email.invalid.retry":   "Request a new login link",

		// Logout
		"logout.title":   "Logged Out",
		"logout.heading": "Logged Out",
		"logout.message": "You have been successfully logged out.",
		"logout.login":   "Login again",

		// Errors
		"error.unauthorized":    "Unauthorized",
		"error.forbidden":       "Access Denied",
		"error.internal":        "Internal Server Error",
		"error.invalid_request": "Invalid Request",
		"error.invalid_email":   "Email is required",

		// Theme and Language
		"ui.theme":         "Theme",
		"ui.theme.auto":    "Auto",
		"ui.theme.light":   "Light ☀️",
		"ui.theme.dark":    "Dark 🌙",
		"ui.language":      "Language",
		"ui.language.en":   "English",
		"ui.language.ja":   "日本語",
	},

	Japanese: Translation{
		// Service
		"service.name":        "Multi OAuth2 Proxy",
		"service.description": "複数のOAuth2プロバイダーに対応した認証プロキシ",

		// Login page
		"login.title":           "ログイン",
		"login.heading":         "サインイン",
		"login.oauth2.heading":  "OAuth2でログイン",
		"login.email.link":      "またはメールでログイン",
		"login.email.heading":   "メールでログイン",
		"login.email.label":     "メールアドレス",
		"login.email.submit":    "ログインリンクを送信",
		"login.back":            "ログイン方法の選択に戻る",

		// Email auth
		"email.sent.title":   "メールを確認してください",
		"email.sent.heading": "メールを確認してください",
		"email.sent.message": "メールアドレスが承認されている場合、まもなくログインリンクが届きます。",
		"email.sent.detail":  "受信箱を確認し、リンクをクリックしてログインしてください。",
		"email.sent.back":    "ログインに戻る",

		"email.invalid.title":   "無効なトークン",
		"email.invalid.heading": "無効または期限切れのトークン",
		"email.invalid.message": "ログインリンクが無効であるか、すでに使用されています。",
		"email.invalid.retry":   "新しいログインリンクをリクエスト",

		// Logout
		"logout.title":   "ログアウトしました",
		"logout.heading": "ログアウトしました",
		"logout.message": "正常にログアウトしました。",
		"logout.login":   "再度ログイン",

		// Errors
		"error.unauthorized":    "未認証",
		"error.forbidden":       "アクセス拒否",
		"error.internal":        "内部サーバーエラー",
		"error.invalid_request": "不正なリクエスト",
		"error.invalid_email":   "メールアドレスが必要です",

		// Theme and Language
		"ui.theme":         "テーマ",
		"ui.theme.auto":    "自動",
		"ui.theme.light":   "ライト ☀️",
		"ui.theme.dark":    "ダーク 🌙",
		"ui.language":      "言語",
		"ui.language.en":   "English",
		"ui.language.ja":   "日本語",
	},
}
