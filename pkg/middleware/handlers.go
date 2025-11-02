package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/ideamans/multi-oauth2-proxy/pkg/assets"
	"github.com/ideamans/multi-oauth2-proxy/pkg/auth/oauth2"
	"github.com/ideamans/multi-oauth2-proxy/pkg/i18n"
	"github.com/ideamans/multi-oauth2-proxy/pkg/session"
)

// handleHealth handles the health check endpoint
func (m *Middleware) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleReady handles the readiness check endpoint
func (m *Middleware) handleReady(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("READY"))
}

// handleStylesCSS serves the embedded CSS
func (m *Middleware) handleStylesCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(assets.GetEmbeddedCSS()))
}

// handleIcon serves the embedded SVG icons
func (m *Middleware) handleIcon(w http.ResponseWriter, r *http.Request) {
	// Extract icon name from URL path
	prefix := m.config.Server.GetAuthPathPrefix()
	fullPrefix := joinAuthPath(prefix, "/assets/icons/")
	iconName := extractPathParam(r.URL.Path, fullPrefix)
	iconPath := "static/icons/" + iconName

	// Read the icon file from embedded filesystem
	data, err := assets.GetEmbeddedIcons().ReadFile(iconPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// buildAuthHeader generates the auth header HTML based on configuration
func (m *Middleware) buildAuthHeader(prefix string) string {
	serviceName := m.config.Service.Name
	iconURL := m.config.Service.IconURL
	logoURL := m.config.Service.LogoURL
	logoWidth := m.config.Service.LogoWidth
	if logoWidth == "" {
		logoWidth = "200px"
	}

	// Pattern 3: Logo image (if configured)
	if logoURL != "" {
		return `<img src="` + logoURL + `" alt="Logo" class="auth-logo" style="--auth-logo-width: ` + logoWidth + `;">
<h1 class="auth-title">` + serviceName + `</h1>`
	}

	// Pattern 2: Icon + System name (if configured)
	if iconURL != "" {
		return `<div class="auth-header">
<img src="` + iconURL + `" alt="Icon" class="auth-icon">
<h1 class="auth-title">` + serviceName + `</h1>
</div>`
	}

	// Pattern 1: Text only (default)
	return `<h1 class="auth-title">` + serviceName + `</h1>`
}

// buildAuthSubtitle generates subtitle HTML if subtitle is provided
func (m *Middleware) buildAuthSubtitle(subtitle string) string {
	if subtitle == "" {
		return ""
	}
	return `<h2 class="auth-subtitle">` + subtitle + `</h2>`
}

// handleLogin displays the login page
func (m *Middleware) handleLogin(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	// Use embedded CSS
	cssPath := joinAuthPath(m.config.Server.GetAuthPathPrefix(), "/assets/styles.css")
	themeClass := ""
	if theme == i18n.ThemeDark {
		themeClass = "dark"
	} else if theme == i18n.ThemeLight {
		themeClass = "light"
	}
	// ThemeAuto: no class, let CSS media query handle it

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("login.title") + ` - ` + m.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssPath + `">
<style>
.settings-toggle {
	position: fixed;
	top: var(--spacing-md);
	right: var(--spacing-md);
	display: flex;
	flex-direction: row;
	gap: var(--spacing-md);
	align-items: center;
	z-index: 100;
	background-color: var(--color-bg-elevated);
	padding: var(--spacing-xs) var(--spacing-md);
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border-default);
}
.settings-toggle select {
	padding: var(--spacing-xs) var(--spacing-sm);
	border: none;
	background: transparent;
	color: var(--color-text-secondary);
	font-size: 0.875rem;
	cursor: pointer;
	appearance: none;
	-webkit-appearance: none;
	-moz-appearance: none;
}
.settings-toggle select:hover {
	color: var(--color-text-primary);
}
.settings-toggle select:focus {
	outline: none;
	color: var(--color-text-primary);
}
</style>
</head>
<body>
<div class="settings-toggle">
	<select id="theme-select" onchange="changeTheme(this.value)">
		<option value="auto" ` + map[bool]string{true: `selected`, false: ``}[theme == i18n.ThemeAuto] + `>` + t("ui.theme.auto") + `</option>
		<option value="light" ` + map[bool]string{true: `selected`, false: ``}[theme == i18n.ThemeLight] + `>` + t("ui.theme.light") + `</option>
		<option value="dark" ` + map[bool]string{true: `selected`, false: ``}[theme == i18n.ThemeDark] + `>` + t("ui.theme.dark") + `</option>
	</select>
	<select id="lang-select" onchange="changeLanguage(this.value)">
		<option value="en" ` + map[bool]string{true: `selected`, false: ``}[lang == i18n.English] + `>` + t("ui.language.en") + `</option>
		<option value="ja" ` + map[bool]string{true: `selected`, false: ``}[lang == i18n.Japanese] + `>` + t("ui.language.ja") + `</option>
	</select>
</div>

<div class="auth-container">
	<div class="card auth-card">
		` + m.buildAuthHeader(m.config.Server.GetAuthPathPrefix()) + `
		<p class="auth-description">` + m.config.Service.Description + `</p>`

	prefix := m.config.Server.GetAuthPathPrefix()
	providers := m.oauthManager.GetProviders()

	if len(providers) > 0 {
		html += `<div style="margin-bottom: var(--spacing-lg);">`
		for _, p := range providers {
			providerName := p.Name()

			// Find icon URL from config
			var iconPath string
			for _, providerCfg := range m.config.OAuth2.Providers {
				if providerCfg.Name == providerName && providerCfg.IconURL != "" {
					// Use custom icon URL from config
					iconPath = providerCfg.IconURL
					break
				}
			}

			// If no custom icon URL, use default embedded icon
			if iconPath == "" {
				iconName := providerName
				knownIcons := map[string]bool{
					"google":    true,
					"github":    true,
					"microsoft": true,
					"facebook":  true,
				}
				if !knownIcons[providerName] {
					iconName = "oidc" // Default to OIDC icon for custom providers
				}
				iconPath = joinAuthPath(prefix, "/assets/icons/"+iconName+".svg")
			}

			html += `<a href="` + prefix + `/oauth2/start/` + providerName + `" class="btn btn-secondary provider-btn">`
			html += `<img src="` + iconPath + `" alt="` + providerName + `">`
			html += fmt.Sprintf(t("login.oauth2.continue"), providerName) + `</a>`
		}
		html += `</div>`
	}

	// Add email authentication form if enabled
	if m.emailHandler != nil {
		emailSendPath := joinAuthPath(m.config.Server.GetAuthPathPrefix(), "/email/send")
		emailIconPath := joinAuthPath(prefix, "/assets/icons/email.svg")

		if len(providers) > 0 {
			html += `<div class="auth-divider"><span>` + t("login.or") + `</span></div>`
		}

		html += `
		<form method="POST" action="` + emailSendPath + `" id="email-form">
			<div class="form-group">
				<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--spacing-xs);">
					<label class="label" for="email" style="margin-bottom: 0;">` + t("login.email.label") + `</label>
					<label style="display: flex; align-items: center; gap: 0.25rem; cursor: pointer; font-size: 0.875rem; color: var(--color-text-secondary);">
						<input type="checkbox" id="save-email-checkbox" style="cursor: pointer;">
						<span>` + t("login.email.save") + `</span>
					</label>
				</div>
				<input type="email" id="email" name="email" class="input" placeholder="you@example.com" required>
			</div>
			<button type="submit" class="btn btn-primary provider-btn">
				<img src="` + emailIconPath + `" alt="Email">
				` + t("login.email.submit") + `
			</button>
		</form>
		<script>
		(function() {
			const emailInput = document.getElementById('email');
			const saveCheckbox = document.getElementById('save-email-checkbox');
			const STORAGE_KEY_EMAIL = 'saved_email';
			const STORAGE_KEY_SAVE = 'save_email_enabled';

			// Load saved settings
			const savedEmail = localStorage.getItem(STORAGE_KEY_EMAIL);
			const saveEnabled = localStorage.getItem(STORAGE_KEY_SAVE) === 'true';

			if (savedEmail && saveEnabled) {
				emailInput.value = savedEmail;
				saveCheckbox.checked = true;
			} else if (saveEnabled) {
				saveCheckbox.checked = true;
			}

			// Save email on input change (if checkbox is checked)
			emailInput.addEventListener('input', function() {
				if (saveCheckbox.checked) {
					localStorage.setItem(STORAGE_KEY_EMAIL, emailInput.value);
				}
			});

			// Handle checkbox changes
			saveCheckbox.addEventListener('change', function() {
				if (saveCheckbox.checked) {
					// Save current email value and remember the checkbox state
					localStorage.setItem(STORAGE_KEY_EMAIL, emailInput.value);
					localStorage.setItem(STORAGE_KEY_SAVE, 'true');
				} else {
					// Clear saved email and checkbox state
					localStorage.removeItem(STORAGE_KEY_EMAIL);
					localStorage.removeItem(STORAGE_KEY_SAVE);
				}
			});
		})();
		</script>`
	}

	html += `
	</div>
</div>
<script>
function setCookie(name, value, days) {
	var expires = "";
	if (days) {
		var date = new Date();
		date.setTime(date.getTime() + (days * 24 * 60 * 60 * 1000));
		expires = "; expires=" + date.toUTCString();
	}
	document.cookie = name + "=" + value + expires + "; path=/; SameSite=Lax";
}

function changeTheme(theme) {
	setCookie("theme", theme, 365);

	// Apply theme immediately without reload
	var html = document.documentElement;
	if (theme === "dark") {
		html.classList.add("dark");
	} else if (theme === "light") {
		html.classList.remove("dark");
	} else {
		// Auto - check system preference
		if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
			html.classList.add("dark");
		} else {
			html.classList.remove("dark");
		}
	}
}

function changeLanguage(lang) {
	setCookie("lang", lang, 365);
	window.location.reload();
}

// Listen for system theme changes
if (window.matchMedia) {
	window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function(e) {
		var savedTheme = getCookie("theme");
		if (!savedTheme || savedTheme === "auto") {
			document.documentElement.classList.toggle("dark", e.matches);
		}
	});
}

function getCookie(name) {
	var nameEQ = name + "=";
	var ca = document.cookie.split(';');
	for(var i=0; i < ca.length; i++) {
		var c = ca[i];
		while (c.charAt(0) == ' ') c = c.substring(1, c.length);
		if (c.indexOf(nameEQ) == 0) return c.substring(nameEQ.length, c.length);
	}
	return null;
}
</script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// handleLogout logs out the user
func (m *Middleware) handleLogout(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	// Get session cookie
	cookie, err := r.Cookie(m.config.Session.CookieName)
	if err == nil {
		// Delete session
		m.sessionStore.Delete(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     m.config.Session.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Use embedded CSS
	prefix := m.config.Server.GetAuthPathPrefix()
	cssPath := joinAuthPath(prefix, "/assets/styles.css")
	loginPath := joinAuthPath(prefix, "/login")

	themeClass := ""
	if theme == i18n.ThemeDark {
		themeClass = "dark"
	} else if theme == i18n.ThemeLight {
		themeClass = "light"
	}

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("logout.title") + ` - ` + m.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssPath + `">
</head>
<body>
<div class="auth-container">
	<div class="card auth-card">
		` + m.buildAuthHeader(prefix) + `
		` + m.buildAuthSubtitle(t("logout.heading")) + `
		<div class="alert alert-success" style="text-align: left; margin-bottom: var(--spacing-md);">` + t("logout.message") + `</div>
		<a href="` + loginPath + `" class="btn btn-primary" style="width: 100%; margin-top: var(--spacing-md);">` + t("logout.login") + `</a>
	</div>
</div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// handleOAuth2Start initiates the OAuth2 flow
func (m *Middleware) handleOAuth2Start(w http.ResponseWriter, r *http.Request) {
	// Extract provider name from URL path
	prefix := m.config.Server.GetAuthPathPrefix()
	fullPrefix := joinAuthPath(prefix, "/oauth2/start/")
	providerName := extractPathParam(r.URL.Path, fullPrefix)

	// Generate state for CSRF protection
	state, err := oauth2.GenerateState()
	if err != nil {
		m.logger.Error("Failed to generate state", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store state in session (simplified for now - in production, use a dedicated state store)
	// For now, we'll pass it directly and verify in callback

	// Get auth URL
	authURL, err := m.oauthManager.GetAuthURL(providerName, state)
	if err != nil {
		m.logger.Error("Failed to get auth URL", "provider", providerName, "error", err)
		http.Error(w, "Invalid provider", http.StatusBadRequest)
		return
	}

	// Store state in a cookie for verification
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		Secure:   m.config.Session.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Store provider in cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_provider",
		Value:    providerName,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   m.config.Session.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to OAuth2 provider
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleOAuth2Callback handles the OAuth2 callback
func (m *Middleware) handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	// Get state from cookie
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		m.logger.Error("State cookie not found")
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Get provider from cookie
	providerCookie, err := r.Cookie("oauth_provider")
	if err != nil {
		m.logger.Error("Provider cookie not found")
		http.Error(w, "Invalid provider", http.StatusBadRequest)
		return
	}

	// Verify state
	state := r.URL.Query().Get("state")
	if state != stateCookie.Value {
		m.logger.Debug("State verification failed", "expected", stateCookie.Value, "actual", state)
		m.logger.Error("OAuth2 authentication failed: state mismatch")
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		m.logger.Error("OAuth2 authentication failed: authorization code not found")
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}

	providerName := providerCookie.Value

	// Exchange code for token
	token, err := m.oauthManager.Exchange(r.Context(), providerName, code)
	if err != nil {
		m.logger.Error("Failed to exchange code", "error", err)
		http.Error(w, "Failed to authenticate", http.StatusInternalServerError)
		return
	}

	// Try to get user email from OAuth2 provider
	// We always try to fetch the user info (email and name) for setting in request headers,
	// regardless of whether authorization check is required
	userInfo, err := m.oauthManager.GetUserInfo(r.Context(), providerName, token)

	var email, name string
	if userInfo != nil {
		email = userInfo.Email
		name = userInfo.Name
	}

	// Check if email-based authorization is required
	if m.authzChecker.RequiresEmail() {
		// Whitelist configured - email is required for authorization
		if err != nil {
			m.logger.Debug("Email fetch failed", "error", err, "provider", providerName)
			m.logger.Error("OAuth2 authentication failed: email required for authorization but could not be retrieved", "provider", providerName)
			m.handleEmailFetchError(w, r)
			return
		}

		// Check authorization
		if !m.authzChecker.IsAllowed(email) {
			m.logger.Info("OAuth2 authentication denied: user not authorized", "email", email, "provider", providerName)
			m.handleForbidden(w, r)
			return
		}
	} else {
		// No whitelist configured - authentication alone is sufficient
		// Email is not required for authorization, but we still try to get it for headers
		if err != nil {
			m.logger.Debug("Email fetch failed (not required for authorization)", "error", err, "provider", providerName)
			m.logger.Warn("Proceeding without user email", "provider", providerName)
			email = "" // Clear email if fetch failed when not required
		}
	}

	// Create session
	sessionID, err := generateSessionID()
	if err != nil {
		m.logger.Error("Failed to generate session ID", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	duration, err := m.config.Session.GetCookieExpireDuration()
	if err != nil {
		duration = 168 * time.Hour // Default 7 days
	}

	sess := &session.Session{
		ID:            sessionID,
		Email:         email,
		Name:          name,
		Provider:      providerName,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(duration),
		Authenticated: true,
	}

	// Store session
	if err := m.sessionStore.Set(sessionID, sess); err != nil {
		m.logger.Debug("Session store failed", "error", err)
		m.logger.Error("OAuth2 authentication failed: could not store session")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     m.config.Session.CookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(duration.Seconds()),
		HttpOnly: m.config.Session.CookieHTTPOnly,
		Secure:   m.config.Session.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Clear OAuth cookies
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_provider",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Log success after all session/cookie operations succeed
	m.logger.Info("OAuth2 authentication successful", "email", email, "name", name, "provider", providerName)

	// Redirect to original URL or home
	http.Redirect(w, r, m.getRedirectURL(w, r), http.StatusFound)
}

// generateSessionID generates a random session ID
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// handleEmailSend sends a login link to the provided email address
func (m *Middleware) handleEmailSend(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	if err := r.ParseForm(); err != nil {
		http.Error(w, t("error.invalid_request"), http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		http.Error(w, t("error.invalid_email"), http.StatusBadRequest)
		return
	}

	// Check authorization before sending
	if !m.authzChecker.IsAllowed(email) {
		m.logger.Info("Email authentication denied: user not authorized", "email", email)
		m.handleForbidden(w, r)
		return
	}

	// Send login link
	err := m.emailHandler.SendLoginLink(email, lang)
	if err != nil {
		m.logger.Debug("Email send failed", "email", email, "error", err)
		m.logger.Error("Email authentication failed: could not send login link", "email", email)
		http.Error(w, t("error.internal"), http.StatusInternalServerError)
		return
	}
	m.logger.Info("Login link sent", "email", email)

	// Use embedded CSS
	prefix := m.config.Server.GetAuthPathPrefix()
	cssPath := joinAuthPath(prefix, "/assets/styles.css")
	loginPath := joinAuthPath(prefix, "/login")

	themeClass := ""
	if theme == i18n.ThemeDark {
		themeClass = "dark"
	} else if theme == i18n.ThemeLight {
		themeClass = "light"
	}

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("email.sent.title") + ` - ` + m.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssPath + `">
</head>
<body>
<div class="auth-container">
	<div class="card auth-card">
		` + m.buildAuthHeader(prefix) + `
		` + m.buildAuthSubtitle(t("email.sent.heading")) + `
		<div class="alert alert-success" style="text-align: left; margin-bottom: var(--spacing-md);">` + t("email.sent.message") + ` ` + t("email.sent.detail") + `</div>
		<a href="` + loginPath + `" class="btn btn-ghost" style="width: 100%; margin-top: var(--spacing-md);">` + t("email.sent.back") + `</a>
	</div>
</div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// handleEmailVerify verifies the email token and creates a session
func (m *Middleware) handleEmailVerify(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, t("error.invalid_request"), http.StatusBadRequest)
		return
	}

	// Verify token
	email, err := m.emailHandler.VerifyToken(token)
	if err != nil {
		m.logger.Debug("Token verification failed", "error", err)
		m.logger.Error("Email authentication failed: invalid or expired token")
		theme := i18n.DetectTheme(r)

		// Use embedded CSS
		prefix := m.config.Server.GetAuthPathPrefix()
		cssPath := joinAuthPath(prefix, "/assets/styles.css")
		loginPath := joinAuthPath(prefix, "/login")

		themeClass := ""
		if theme == i18n.ThemeDark {
			themeClass = "dark"
		} else if theme == i18n.ThemeLight {
			themeClass = "light"
		}

		html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("email.invalid.title") + ` - ` + m.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssPath + `">
</head>
<body>
<div class="auth-container">
	<div class="card auth-card">
		` + m.buildAuthHeader(prefix) + `
		` + m.buildAuthSubtitle(t("email.invalid.heading")) + `
		<div class="alert alert-error" style="text-align: left; margin-bottom: var(--spacing-md);"><strong>Error:</strong> ` + t("email.invalid.message") + ` This link cannot be used to authenticate.</div>
		<a href="` + loginPath + `" class="btn btn-primary" style="width: 100%; margin-top: var(--spacing-md);">` + t("email.invalid.retry") + `</a>
	</div>
</div>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(html))
		return
	}

	// Check authorization if whitelist is configured
	if m.authzChecker.RequiresEmail() {
		if !m.authzChecker.IsAllowed(email) {
			m.logger.Info("Email authentication denied: user not authorized", "email", email)
			m.handleForbidden(w, r)
			return
		}
		m.logger.Debug("User authorized", "email", email)
	} else {
		m.logger.Debug("No whitelist configured, skipping authorization check", "email", email)
	}

	// Create session
	sessionID, err := generateSessionID()
	if err != nil {
		m.logger.Error("Failed to generate session ID", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	duration, err := m.config.Session.GetCookieExpireDuration()
	if err != nil {
		duration = 168 * time.Hour // Default 7 days
	}

	sess := &session.Session{
		ID:            sessionID,
		Email:         email,
		Provider:      "email",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(duration),
		Authenticated: true,
	}

	// Store session
	if err := m.sessionStore.Set(sessionID, sess); err != nil {
		m.logger.Debug("Session store failed", "error", err)
		m.logger.Error("Email authentication failed: could not store session")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     m.config.Session.CookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(duration.Seconds()),
		HttpOnly: m.config.Session.CookieHTTPOnly,
		Secure:   m.config.Session.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	m.logger.Info("Email authentication successful", "email", email)

	// Redirect to original URL or home
	http.Redirect(w, r, m.getRedirectURL(w, r), http.StatusFound)
}

// handleForbidden displays the access denied page
func (m *Middleware) handleForbidden(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	prefix := normalizeAuthPrefix(m.config.Server.GetAuthPathPrefix())
	cssPath := joinAuthPath(prefix, "/assets/styles.css")
	loginPath := joinAuthPath(prefix, "/login")

	themeClass := ""
	if theme == i18n.ThemeDark {
		themeClass = "dark"
	} else if theme == i18n.ThemeLight {
		themeClass = "light"
	}
	// ThemeAuto: no class, let CSS media query handle it

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("error.forbidden.title") + ` - ` + m.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssPath + `">
</head>
<body>
<div class="auth-container">
  <div class="card auth-card">
    ` + m.buildAuthHeader(prefix) + `
    ` + m.buildAuthSubtitle(t("error.forbidden.heading")) + `
    <div class="alert alert-error" style="text-align: left; margin-bottom: var(--spacing-md);">` + t("error.forbidden.message") + `</div>
    <a href="` + loginPath + `" class="btn btn-ghost" style="width: 100%; margin-top: var(--spacing-md);">` + t("login.back") + `</a>
  </div>
</div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(html))
}

// handleEmailFetchError displays an error page when OAuth2 provider fails to provide email
func (m *Middleware) handleEmailFetchError(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	prefix := normalizeAuthPrefix(m.config.Server.GetAuthPathPrefix())
	cssPath := joinAuthPath(prefix, "/assets/styles.css")
	loginPath := joinAuthPath(prefix, "/login")

	themeClass := ""
	if theme == i18n.ThemeDark {
		themeClass = "dark"
	} else if theme == i18n.ThemeLight {
		themeClass = "light"
	}

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("error.email_required.title") + ` - ` + m.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssPath + `">
</head>
<body>
<div class="auth-container">
  <div class="card auth-card">
    ` + m.buildAuthHeader(prefix) + `
    ` + m.buildAuthSubtitle(t("error.email_required.heading")) + `
    <div class="alert alert-error" style="text-align: left; margin-bottom: var(--spacing-md);">` + t("error.email_required.message") + `</div>
    <a href="` + loginPath + `" class="btn btn-ghost" style="width: 100%; margin-top: var(--spacing-md);">` + t("login.back") + `</a>
  </div>
</div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(html))
}
