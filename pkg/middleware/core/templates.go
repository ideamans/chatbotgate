package middleware

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
)

// PageData contains common data for all pages
type PageData struct {
	Lang               i18n.Language
	Theme              i18n.Theme
	ServiceName        string
	ServiceDescription string
	Title              string
	Subtitle           string
	Header             template.HTML // Pre-rendered header HTML
	StyleLinks         template.HTML // Pre-rendered style links
	CreditIcon         string
}

// LoginPageData contains data for the login page
type LoginPageData struct {
	PageData
	Providers        []ProviderData
	EmailEnabled     bool
	PasswordEnabled  bool
	EmailSendPath    string
	EmailIconPath    string
	PasswordFormHTML template.HTML
	Translations     LoginTranslations
}

// ProviderData contains OAuth2 provider display data
type ProviderData struct {
	Name     string
	IconPath string
	URL      string
	Label    string
}

// LoginTranslations contains translated strings for login page
type LoginTranslations struct {
	Or          string
	EmailLabel  string
	EmailSave   string
	EmailSubmit string
	ThemeAuto   string
	ThemeLight  string
	ThemeDark   string
	LanguageEn  string
	LanguageJa  string
}

// LogoutPageData contains data for the logout page
type LogoutPageData struct {
	PageData
	Message    string
	LoginURL   string
	LoginLabel string
}

// EmailSentPageData contains data for the email sent page
type EmailSentPageData struct {
	PageData
	Message        string
	Detail         string
	OTPLabel       string
	OTPPlaceholder string
	VerifyButton   string
	BackLabel      string
	LoginURL       string
	VerifyOTPPath  string
}

// ErrorPageData contains data for error pages
type ErrorPageData struct {
	PageData
	Message      string
	Detail       string
	ErrorDetails template.HTML // For 500 error accordion
	ActionURL    string
	ActionLabel  string
}

// Templates holds all parsed templates
type Templates struct {
	login     *template.Template
	logout    *template.Template
	emailSent *template.Template
	forbidden *template.Template
	emailReq  *template.Template
	notFound  *template.Template
	server    *template.Template
}

// newTemplates creates and parses all templates
func newTemplates() (*Templates, error) {
	t := &Templates{}

	var err error

	// Parse login template
	t.login, err = template.New("login").Parse(loginTemplate)
	if err != nil {
		return nil, err
	}

	// Parse logout template
	t.logout, err = template.New("logout").Parse(logoutTemplate)
	if err != nil {
		return nil, err
	}

	// Parse email sent template
	t.emailSent, err = template.New("emailSent").Parse(emailSentTemplate)
	if err != nil {
		return nil, err
	}

	// Parse forbidden template
	t.forbidden, err = template.New("forbidden").Parse(forbiddenTemplate)
	if err != nil {
		return nil, err
	}

	// Parse email required template
	t.emailReq, err = template.New("emailReq").Parse(emailRequiredTemplate)
	if err != nil {
		return nil, err
	}

	// Parse 404 template
	t.notFound, err = template.New("notFound").Parse(notFoundTemplate)
	if err != nil {
		return nil, err
	}

	// Parse 500 template
	t.server, err = template.New("server").Parse(serverErrorTemplate)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// renderTemplate renders a template to the response writer
func renderTemplate(w http.ResponseWriter, tmpl *template.Template, data interface{}, m *Middleware) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	m.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(buf.Bytes())
	return err
}

// renderErrorTemplate renders an error template with a specific status code
func renderErrorTemplate(w http.ResponseWriter, tmpl *template.Template, data interface{}, statusCode int, m *Middleware) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	m.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, err := w.Write(buf.Bytes())
	return err
}

// buildPageData builds common page data
func (m *Middleware) buildPageData(lang i18n.Language, theme i18n.Theme, titleKey string) PageData {
	t := func(key string) string { return m.translator.T(lang, key) }
	prefix := m.config.Server.GetAuthPathPrefix()

	return PageData{
		Lang:               lang,
		Theme:              theme,
		ServiceName:        m.config.Service.Name,
		ServiceDescription: m.config.Service.Description,
		Title:              t(titleKey),
		Header:             template.HTML(m.buildAuthHeaderHTML(prefix)),
		StyleLinks:         template.HTML(m.buildStyleLinksHTML()),
		CreditIcon:         joinAuthPath(normalizeAuthPrefix(prefix), "/assets/icons/chatbotgate.svg"),
	}
}

// buildAuthHeaderHTML generates the auth header HTML
func (m *Middleware) buildAuthHeaderHTML(prefix string) string {
	serviceName := m.config.Service.Name
	iconURL := m.config.Service.IconURL
	logoURL := m.config.Service.LogoURL
	logoWidth := m.config.Service.LogoWidth
	if logoWidth == "" {
		logoWidth = "200px"
	}

	// Pattern 3: Logo image (if configured)
	if logoURL != "" {
		return `<img src="` + template.HTMLEscapeString(logoURL) + `" alt="Logo" class="auth-logo" style="--auth-logo-width: ` + template.HTMLEscapeString(logoWidth) + `;">
<h1 class="auth-title">` + template.HTMLEscapeString(serviceName) + `</h1>`
	}

	// Pattern 2: Icon + System name (if configured)
	if iconURL != "" {
		return `<div class="auth-header">
<img src="` + template.HTMLEscapeString(iconURL) + `" alt="Icon" class="auth-icon">
<h1 class="auth-title">` + template.HTMLEscapeString(serviceName) + `</h1>
</div>`
	}

	// Pattern 1: Text only (default)
	return `<h1 class="auth-title">` + template.HTMLEscapeString(serviceName) + `</h1>`
}

// buildStyleLinksHTML generates stylesheet link tags
func (m *Middleware) buildStyleLinksHTML() string {
	prefix := m.config.Server.GetAuthPathPrefix()
	cssPath := joinAuthPath(prefix, "/assets/main.css")
	links := `<link rel="stylesheet" href="` + template.HTMLEscapeString(cssPath) + `">`

	// Add dify.css if optimization is enabled
	if m.config.Assets.Optimization.Dify {
		difyCSSPath := joinAuthPath(prefix, "/assets/dify.css")
		links += `
<link rel="stylesheet" href="` + template.HTMLEscapeString(difyCSSPath) + `">`
	}

	return links
}
