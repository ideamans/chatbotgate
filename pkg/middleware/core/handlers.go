package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/assets"
	"github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
	"github.com/ideamans/chatbotgate/pkg/middleware/forwarding"
	"github.com/ideamans/chatbotgate/pkg/middleware/session"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
)

// handleHealth handles the health check endpoint
func (m *Middleware) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// handleReady handles the readiness check endpoint
func (m *Middleware) handleReady(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("READY"))
}

// handleMainCSS serves the embedded CSS
func (m *Middleware) handleMainCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(assets.GetEmbeddedCSS()))
}

// handleDifyCSS serves the embedded Dify CSS for iframe optimizations
func (m *Middleware) handleDifyCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(assets.GetEmbeddedDifyCSS()))
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
	_, _ = w.Write(data)
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

// buildStyleLinks generates stylesheet link tags based on configuration
func (m *Middleware) buildStyleLinks() string {
	prefix := m.config.Server.GetAuthPathPrefix()
	cssPath := joinAuthPath(prefix, "/assets/main.css")
	links := `<link rel="stylesheet" href="` + cssPath + `">`

	// Add dify.css if optimization is enabled
	if m.config.Assets.Optimization.Dify {
		difyCSSPath := joinAuthPath(prefix, "/assets/dify.css")
		links += `
<link rel="stylesheet" href="` + difyCSSPath + `">`
	}

	return links
}

// handleLogin displays the login page
func (m *Middleware) handleLogin(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	// Use embedded CSS
	themeClass := ""
	switch theme {
	case i18n.ThemeDark:
		themeClass = "dark"
	case i18n.ThemeLight:
		themeClass = "light"
	default:
		// ThemeAuto: no class, let CSS media query handle it
	}

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("login.title") + ` - ` + m.config.Service.Name + `</title>
` + m.buildStyleLinks() + `
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
	<div style="width: 100%; max-width: 28rem;">
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

	prefix = normalizeAuthPrefix(m.config.Server.GetAuthPathPrefix())
	iconPath := joinAuthPath(prefix, "/assets/icons/chatbotgate.svg")

	html += `
		</div>
		<a href="https://github.com/ideamans/chatbotgate" class="auth-credit">
			<img src="` + iconPath + `" alt="ChatbotGate Logo">
			Protected by ChatbotGate
		</a>
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

	m.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

// handleLogout logs out the user
func (m *Middleware) handleLogout(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	// Get session cookie
	cookie, err := r.Cookie(m.config.Session.CookieName)
	if err == nil {
		// Delete session (ignore error, proceed with logout anyway)
		_ = session.Delete(m.sessionStore, cookie.Value)
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
	loginPath := joinAuthPath(prefix, "/login")

	themeClass := ""
	switch theme {
	case i18n.ThemeDark:
		themeClass = "dark"
	case i18n.ThemeLight:
		themeClass = "light"
	default:
		// ThemeAuto: no class
	}

	iconPath := joinAuthPath(prefix, "/assets/icons/chatbotgate.svg")

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("logout.title") + ` - ` + m.config.Service.Name + `</title>
` + m.buildStyleLinks() + `
</head>
<body>
<div class="auth-container">
	<div style="width: 100%; max-width: 28rem;">
		<div class="card auth-card">
			` + m.buildAuthHeader(prefix) + `
			` + m.buildAuthSubtitle(t("logout.heading")) + `
			<div class="alert alert-success" style="text-align: left; margin-bottom: var(--spacing-md);">` + t("logout.message") + `</div>
			<a href="` + loginPath + `" class="btn btn-primary" style="width: 100%; margin-top: var(--spacing-md);">` + t("logout.login") + `</a>
		</div>
		<a href="https://github.com/ideamans/chatbotgate" class="auth-credit">
			<img src="` + iconPath + `" alt="ChatbotGate Logo">
			Protected by ChatbotGate
		</a>
	</div>
</div>
</body>
</html>`

	m.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
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

	// Determine the base URL for OAuth2 callback
	// Priority: 1. proxyserver.base_url, 2. request Host header
	var authURL, redirectURL string
	if m.config.Server.BaseURL != "" {
		// Use configured base URL
		authURL, redirectURL, err = m.oauthManager.GetAuthURLWithRedirect(providerName, state, m.config.Server.BaseURL, prefix)
		m.logger.Debug("Generated OAuth2 auth URL", "provider", providerName, "base_url", m.config.Server.BaseURL, "redirect_url", redirectURL)
	} else {
		// Use request host (dynamic)
		requestHost := r.Host
		authURL, redirectURL, err = m.oauthManager.GetAuthURLWithRedirect(providerName, state, requestHost, prefix)
		m.logger.Debug("Generated OAuth2 auth URL", "provider", providerName, "request_host", requestHost, "redirect_url", redirectURL)
	}
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
		SameSite: m.config.Session.GetCookieSameSite(),
	})

	// Store provider in cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_provider",
		Value:    providerName,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   m.config.Session.CookieSecure,
		SameSite: m.config.Session.GetCookieSameSite(),
	})

	// Store redirect URL in cookie for token exchange
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_redirect_url",
		Value:    redirectURL,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   m.config.Session.CookieSecure,
		SameSite: m.config.Session.GetCookieSameSite(),
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

	// Get redirect URL from cookie
	redirectURLCookie, err := r.Cookie("oauth_redirect_url")
	if err != nil {
		m.logger.Error("Redirect URL cookie not found")
		http.Error(w, "Invalid redirect URL", http.StatusBadRequest)
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
	oauthRedirectURL := redirectURLCookie.Value

	// Exchange code for token using the same redirect URL
	token, err := m.oauthManager.ExchangeWithRedirect(r.Context(), providerName, code, oauthRedirectURL)
	if err != nil {
		m.logger.Error("Failed to exchange code", "error", err, "redirect_url", oauthRedirectURL)
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

		// Check if email was actually provided by the OAuth2 provider
		if email == "" {
			m.logger.Error("OAuth2 authentication failed: email required for authorization but not provided by OAuth2 provider", "provider", providerName)
			m.handleEmailFetchError(w, r)
			return
		}

		// Check authorization
		if !m.authzChecker.IsAllowed(email) {
			m.logger.Info("OAuth2 authentication denied: user not authorized", "email", maskEmail(email), "provider", providerName)
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

	// Delete any existing session to prevent session fixation attacks
	if oldCookie, err := r.Cookie(m.config.Session.CookieName); err == nil {
		_ = session.Delete(m.sessionStore, oldCookie.Value)
	}

	// Create session with new session ID
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

	// Safely extract Extra data, handling nil userInfo case
	var extra map[string]interface{}
	if userInfo != nil && userInfo.Extra != nil {
		extra = userInfo.Extra
	} else {
		extra = make(map[string]interface{})
	}

	sess := &session.Session{
		ID:            sessionID,
		Email:         email,
		Name:          name,
		Provider:      providerName,
		Extra:         extra, // Store additional user data for custom forwarding
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(duration),
		Authenticated: true,
	}

	// Store session
	if err := session.Set(m.sessionStore, sessionID, sess); err != nil {
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
		SameSite: m.config.Session.GetCookieSameSite(),
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
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_redirect_url",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Log success after all session/cookie operations succeed
	m.logger.Info("OAuth2 authentication successful", "email", maskEmail(email), "name", name, "provider", providerName)

	// Get redirect URL
	redirectURL := m.getRedirectURL(w, r)

	// Add user info to query string if forwarding is enabled
	if m.forwarder != nil {
		fwdUserInfo := &forwarding.UserInfo{
			Username: name,
			Email:    email,
			Extra:    userInfo.Extra,
			Provider: providerName,
		}
		if modifiedURL, err := m.forwarder.AddToQueryString(redirectURL, fwdUserInfo); err == nil {
			redirectURL = modifiedURL
		} else {
			m.logger.Warn("Failed to add user info to redirect URL", "error", err)
		}
	}

	// Redirect to original URL or home
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// generateSessionID generates a random session ID
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// extractUserpart extracts the local part (before @) from an email address
func extractUserpart(email string) string {
	at := strings.Index(email, "@")
	if at == -1 {
		return email
	}
	return email[:at]
}

// handleEmailSend sends a login link to the provided email address
func (m *Middleware) handleEmailSend(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
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

	// Validate email address to prevent SMTP injection
	if !isValidEmail(email) {
		m.logger.Warn("Invalid email address format", "email", maskEmail(email))
		http.Error(w, t("error.invalid_email"), http.StatusBadRequest)
		return
	}

	// Check authorization before sending
	if !m.authzChecker.IsAllowed(email) {
		m.logger.Info("Email authentication denied: user not authorized", "email", maskEmail(email))
		m.handleForbidden(w, r)
		return
	}

	// Send login link
	err := m.emailHandler.SendLoginLink(email, lang)
	if err != nil {
		m.logger.Debug("Email send failed", "email", maskEmail(email), "error", err)
		m.logger.Error("Email authentication failed: could not send login link", "email", maskEmail(email))
		http.Error(w, t("error.internal"), http.StatusInternalServerError)
		return
	}
	m.logger.Info("Login link sent", "email", maskEmail(email))

	// Redirect to email sent page
	prefix := m.config.Server.GetAuthPathPrefix()
	emailSentPath := joinAuthPath(prefix, "/email/sent")
	http.Redirect(w, r, emailSentPath, http.StatusSeeOther)
}

// handleEmailSent shows the email sent confirmation page with OTP input
func (m *Middleware) handleEmailSent(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	// Use embedded CSS
	prefix := m.config.Server.GetAuthPathPrefix()
	loginPath := joinAuthPath(prefix, "/login")

	themeClass := ""
	switch theme {
	case i18n.ThemeDark:
		themeClass = "dark"
	case i18n.ThemeLight:
		themeClass = "light"
	default:
		// ThemeAuto: no class
	}

	iconPath := joinAuthPath(prefix, "/assets/icons/chatbotgate.svg")

	verifyOTPPath := joinAuthPath(prefix, "/email/verify-otp")

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("email.sent.title") + ` - ` + m.config.Service.Name + `</title>
` + m.buildStyleLinks() + `
</head>
<body>
<div class="auth-container">
	<div style="width: 100%; max-width: 28rem;">
		<div class="card auth-card">
			` + m.buildAuthHeader(prefix) + `
			` + m.buildAuthSubtitle(t("email.sent.heading")) + `
			<div class="alert alert-success" style="text-align: left; margin-bottom: var(--spacing-md);">` + t("email.sent.message") + ` ` + t("email.sent.detail") + `</div>

			<!-- OTP Input Section -->
			<div style="text-align: center; margin-top: var(--spacing-lg); margin-bottom: var(--spacing-lg);">
				<div style="margin-bottom: var(--spacing-sm);">
					<span style="color: var(--color-text-secondary); font-size: 0.875rem;">` + t("email.sent.otp_label") + `</span>
				</div>
				<form method="POST" action="` + verifyOTPPath + `" style="display: flex; flex-direction: column; align-items: center; gap: var(--spacing-sm);">
					<input
						type="text"
						name="otp"
						id="otp-input"
						class="input"
						placeholder="` + t("email.sent.otp_placeholder") + `"
						maxlength="14"
						autocomplete="off"
						style="text-align: center; font-family: 'Courier New', monospace; font-size: 1.125rem; font-weight: 600; letter-spacing: 0.05em; background-color: var(--color-bg-muted); border: 2px solid var(--color-border-default); max-width: 16rem; transition: border-color 0.2s ease, background-color 0.2s ease;">
					<button type="submit" id="verify-button" class="btn btn-primary" disabled style="max-width: 16rem; width: 100%;">
						` + t("email.sent.verify_button") + `
					</button>
				</form>
			</div>

			<a href="` + loginPath + `" class="btn btn-ghost" style="width: 100%; margin-top: var(--spacing-md);">` + t("email.sent.back") + `</a>
		</div>
		<a href="https://github.com/ideamans/chatbotgate" class="auth-credit">
			<img src="` + iconPath + `" alt="ChatbotGate Logo">
			Protected by ChatbotGate
		</a>
	</div>
</div>
<script>
(function() {
	const otpInput = document.getElementById('otp-input');
	const verifyButton = document.getElementById('verify-button');
	if (!otpInput || !verifyButton) return;

	function validateOTP(value) {
		const cleaned = value.replace(/[^A-Z0-9]/gi, '').toUpperCase();
		return cleaned.length === 12 && /^[A-Z0-9]{12}$/.test(cleaned);
	}

	function updateUI(isValid) {
		if (isValid) {
			// Input: Green border and background
			otpInput.style.borderColor = 'var(--color-success)';
			otpInput.style.backgroundColor = 'color-mix(in srgb, var(--color-success) 10%, var(--color-bg-muted))';

			// Button: Enable (keep btn-primary style)
			verifyButton.disabled = false;
		} else {
			// Input: Default style
			otpInput.style.borderColor = 'var(--color-border-default)';
			otpInput.style.backgroundColor = 'var(--color-bg-muted)';

			// Button: Disable
			verifyButton.disabled = true;
		}
	}

	otpInput.addEventListener('input', function() {
		updateUI(validateOTP(this.value));
	});
})();
</script>
</body>
</html>`

	m.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
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
		loginPath := joinAuthPath(prefix, "/login")

		themeClass := ""
		switch theme {
		case i18n.ThemeDark:
			themeClass = "dark"
		case i18n.ThemeLight:
			themeClass = "light"
		default:
			// ThemeAuto: no class
		}

		iconPath := joinAuthPath(prefix, "/assets/icons/chatbotgate.svg")

		html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("email.invalid.title") + ` - ` + m.config.Service.Name + `</title>
` + m.buildStyleLinks() + `
</head>
<body>
<div class="auth-container">
	<div style="width: 100%; max-width: 28rem;">
		<div class="card auth-card">
			` + m.buildAuthHeader(prefix) + `
			` + m.buildAuthSubtitle(t("email.invalid.heading")) + `
			<div class="alert alert-error" style="text-align: left; margin-bottom: var(--spacing-md);"><strong>Error:</strong> ` + t("email.invalid.message") + ` This link cannot be used to authenticate.</div>
			<a href="` + loginPath + `" class="btn btn-primary" style="width: 100%; margin-top: var(--spacing-md);">` + t("email.invalid.retry") + `</a>
		</div>
		<a href="https://github.com/ideamans/chatbotgate" class="auth-credit">
			<img src="` + iconPath + `" alt="ChatbotGate Logo">
			Protected by ChatbotGate
		</a>
	</div>
</div>
</body>
</html>`
		m.setSecurityHeaders(w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(html))
		return
	}

	// Check authorization if whitelist is configured
	if m.authzChecker.RequiresEmail() {
		if !m.authzChecker.IsAllowed(email) {
			m.logger.Info("Email authentication denied: user not authorized", "email", maskEmail(email))
			m.handleForbidden(w, r)
			return
		}
		m.logger.Debug("User authorized", "email", maskEmail(email))
	} else {
		m.logger.Debug("No whitelist configured, skipping authorization check", "email", maskEmail(email))
	}

	// Delete any existing session to prevent session fixation attacks
	if oldCookie, err := r.Cookie(m.config.Session.CookieName); err == nil {
		_ = session.Delete(m.sessionStore, oldCookie.Value)
	}

	// Create session with new session ID
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

	// Create Extra fields with standardized OAuth2-compatible fields
	userpart := extractUserpart(email)
	extra := make(map[string]interface{})
	extra["_email"] = email
	extra["_username"] = userpart
	extra["_avatar_url"] = ""
	extra["userpart"] = userpart

	sess := &session.Session{
		ID:            sessionID,
		Email:         email,
		Name:          userpart, // Set Name to userpart for consistency with forwarding
		Provider:      "email",
		Extra:         extra,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(duration),
		Authenticated: true,
	}

	// Store session
	if err := session.Set(m.sessionStore, sessionID, sess); err != nil {
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
		SameSite: m.config.Session.GetCookieSameSite(),
	})

	m.logger.Info("Email authentication successful", "email", maskEmail(email))

	// Get redirect URL
	redirectURL := m.getRedirectURL(w, r)

	// Add user info to query string if forwarding is enabled
	if m.forwarder != nil {
		fwdUserInfo := &forwarding.UserInfo{
			Username: userpart,
			Email:    email,
			Extra:    extra,
			Provider: "email",
		}
		if modifiedURL, err := m.forwarder.AddToQueryString(redirectURL, fwdUserInfo); err == nil {
			redirectURL = modifiedURL
		} else {
			m.logger.Warn("Failed to add user info to redirect URL", "error", err)
		}
	}

	// Redirect to original URL or home
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// handleEmailVerifyOTP verifies the OTP and creates a session
func (m *Middleware) handleEmailVerifyOTP(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	if r.Method != http.MethodPost {
		http.Error(w, t("error.invalid_request"), http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, t("error.invalid_request"), http.StatusBadRequest)
		return
	}

	otp := r.FormValue("otp")
	if otp == "" {
		http.Error(w, t("error.invalid_request"), http.StatusBadRequest)
		return
	}

	// Verify OTP
	email, err := m.emailHandler.VerifyOTP(otp)
	if err != nil {
		m.logger.Debug("OTP verification failed", "error", err)
		m.logger.Error("Email authentication failed: invalid or expired OTP")

		// Redirect back to email sent page with error
		prefix := m.config.Server.GetAuthPathPrefix()
		emailSentPath := joinAuthPath(prefix, "/email/sent")
		http.Redirect(w, r, emailSentPath+"?error=invalid_otp", http.StatusFound)
		return
	}

	// Check authorization if whitelist is configured
	if m.authzChecker.RequiresEmail() {
		if !m.authzChecker.IsAllowed(email) {
			m.logger.Info("Email authentication denied: user not authorized", "email", maskEmail(email))
			m.handleForbidden(w, r)
			return
		}
		m.logger.Debug("User authorized", "email", maskEmail(email))
	} else {
		m.logger.Debug("No whitelist configured, skipping authorization check", "email", maskEmail(email))
	}

	// Delete any existing session to prevent session fixation attacks
	if oldCookie, err := r.Cookie(m.config.Session.CookieName); err == nil {
		_ = session.Delete(m.sessionStore, oldCookie.Value)
	}

	// Create session with new session ID
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

	// Create Extra fields with standardized OAuth2-compatible fields
	userpart := extractUserpart(email)
	extra := make(map[string]interface{})
	extra["_email"] = email
	extra["_username"] = userpart
	extra["_avatar_url"] = ""
	extra["userpart"] = userpart

	sess := &session.Session{
		ID:            sessionID,
		Email:         email,
		Name:          userpart, // Set Name to userpart for consistency with forwarding
		Provider:      "email",
		Extra:         extra,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(duration),
		Authenticated: true,
	}

	// Store session
	if err := session.Set(m.sessionStore, sessionID, sess); err != nil {
		m.logger.Error("Failed to store session", "error", err)
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
		SameSite: m.config.Session.GetCookieSameSite(),
	})

	m.logger.Info("Email authentication successful via OTP", "email", maskEmail(email))

	// Get redirect URL
	redirectURL := m.getRedirectURL(w, r)

	// Add user info to query string if forwarding is enabled
	if m.forwarder != nil {
		fwdUserInfo := &forwarding.UserInfo{
			Username: userpart,
			Email:    email,
			Extra:    extra,
			Provider: "email",
		}
		if modifiedURL, err := m.forwarder.AddToQueryString(redirectURL, fwdUserInfo); err == nil {
			redirectURL = modifiedURL
		} else {
			m.logger.Warn("Failed to add user info to redirect URL", "error", err)
		}
	}

	// Redirect to original URL or home
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// handleForbidden displays the access denied page
func (m *Middleware) handleForbidden(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	prefix := normalizeAuthPrefix(m.config.Server.GetAuthPathPrefix())
	loginPath := joinAuthPath(prefix, "/login")

	themeClass := ""
	switch theme {
	case i18n.ThemeDark:
		themeClass = "dark"
	case i18n.ThemeLight:
		themeClass = "light"
	default:
		// ThemeAuto: no class, let CSS media query handle it
	}

	iconPath := joinAuthPath(prefix, "/assets/icons/chatbotgate.svg")

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("error.forbidden.title") + ` - ` + m.config.Service.Name + `</title>
` + m.buildStyleLinks() + `
</head>
<body>
<div class="auth-container">
  <div style="width: 100%; max-width: 28rem;">
    <div class="card auth-card">
      ` + m.buildAuthHeader(prefix) + `
      ` + m.buildAuthSubtitle(t("error.forbidden.heading")) + `
      <div class="alert alert-error" style="text-align: left; margin-bottom: var(--spacing-md);">` + t("error.forbidden.message") + `</div>
      <a href="` + loginPath + `" class="btn btn-ghost" style="width: 100%; margin-top: var(--spacing-md);">` + t("login.back") + `</a>
    </div>
    <a href="https://github.com/ideamans/chatbotgate" class="auth-credit">
      <img src="` + iconPath + `" alt="ChatbotGate Logo">
      Protected by ChatbotGate
    </a>
  </div>
</div>
</body>
</html>`

	m.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(html))
}

// handleEmailFetchError displays an error page when OAuth2 provider fails to provide email
func (m *Middleware) handleEmailFetchError(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	prefix := normalizeAuthPrefix(m.config.Server.GetAuthPathPrefix())
	loginPath := joinAuthPath(prefix, "/login")

	themeClass := ""
	switch theme {
	case i18n.ThemeDark:
		themeClass = "dark"
	case i18n.ThemeLight:
		themeClass = "light"
	default:
		// ThemeAuto: no class
	}

	iconPath := joinAuthPath(prefix, "/assets/icons/chatbotgate.svg")

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("error.email_required.title") + ` - ` + m.config.Service.Name + `</title>
` + m.buildStyleLinks() + `
</head>
<body>
<div class="auth-container">
  <div style="width: 100%; max-width: 28rem;">
    <div class="card auth-card">
      ` + m.buildAuthHeader(prefix) + `
      ` + m.buildAuthSubtitle(t("error.email_required.heading")) + `
      <div class="alert alert-error" style="text-align: left; margin-bottom: var(--spacing-md);">` + t("error.email_required.message") + `</div>
      <a href="` + loginPath + `" class="btn btn-ghost" style="width: 100%; margin-top: var(--spacing-md);">` + t("login.back") + `</a>
    </div>
    <a href="https://github.com/ideamans/chatbotgate" class="auth-credit">
      <img src="` + iconPath + `" alt="ChatbotGate Logo">
      Protected by ChatbotGate
    </a>
  </div>
</div>
</body>
</html>`

	m.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(html))
}

// handle404 displays the 404 Not Found page
func (m *Middleware) handle404(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	prefix := normalizeAuthPrefix(m.config.Server.GetAuthPathPrefix())
	iconPath := joinAuthPath(prefix, "/assets/icons/chatbotgate.svg")

	themeClass := ""
	switch theme {
	case i18n.ThemeDark:
		themeClass = "dark"
	case i18n.ThemeLight:
		themeClass = "light"
	default:
		// ThemeAuto: no class
	}

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("error.notfound.title") + ` - ` + m.config.Service.Name + `</title>
` + m.buildStyleLinks() + `
</head>
<body>
<div class="auth-container">
  <div style="width: 100%; max-width: 28rem;">
    <div class="card auth-card">
      ` + m.buildAuthHeader(prefix) + `
      ` + m.buildAuthSubtitle(t("error.notfound.heading")) + `
      <div class="alert alert-error" style="text-align: left; margin-bottom: var(--spacing-md);">` + t("error.notfound.message") + `</div>
      <a href="/" class="btn btn-primary" style="width: 100%; margin-top: var(--spacing-md);">` + t("error.notfound.home") + `</a>
    </div>
    <a href="https://github.com/ideamans/chatbotgate" class="auth-credit">
      <img src="` + iconPath + `" alt="ChatbotGate Logo">
      Protected by ChatbotGate
    </a>
  </div>
</div>
</body>
</html>`

	m.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(html))
}

// handle500 displays the 500 Internal Server Error page with optional error details
func (m *Middleware) handle500(w http.ResponseWriter, r *http.Request, err error) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	prefix := normalizeAuthPrefix(m.config.Server.GetAuthPathPrefix())
	iconPath := joinAuthPath(prefix, "/assets/icons/chatbotgate.svg")

	themeClass := ""
	switch theme {
	case i18n.ThemeDark:
		themeClass = "dark"
	case i18n.ThemeLight:
		themeClass = "light"
	default:
		// ThemeAuto: no class
	}

	// Build error details accordion if error is provided
	errorDetailsHTML := ""
	if err != nil {
		errorDetailsHTML = `
    <div class="accordion" id="error-accordion">
      <div class="accordion-header" onclick="document.getElementById('error-accordion').classList.toggle('open')">
        <span class="accordion-header-title">` + t("error.details.title") + `</span>
        <span class="accordion-header-icon"></span>
      </div>
      <div class="accordion-content">
        <div class="accordion-body">` + html.EscapeString(fmt.Sprintf("%+v", err)) + `</div>
      </div>
    </div>`
	}

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `" class="` + themeClass + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("error.server.title") + ` - ` + m.config.Service.Name + `</title>
` + m.buildStyleLinks() + `
</head>
<body>
<div class="auth-container">
  <div style="width: 100%; max-width: 28rem;">
    <div class="card auth-card">
      ` + m.buildAuthHeader(prefix) + `
      ` + m.buildAuthSubtitle(t("error.server.heading")) + `
      <div class="alert alert-error" style="text-align: left; margin-bottom: var(--spacing-md);">` + t("error.server.message") + `</div>
      ` + errorDetailsHTML + `
      <a href="/" class="btn btn-ghost" style="width: 100%; margin-top: var(--spacing-md);">` + t("error.server.home") + `</a>
    </div>
    <a href="https://github.com/ideamans/chatbotgate" class="auth-credit">
      <img src="` + iconPath + `" alt="ChatbotGate Logo">
      Protected by ChatbotGate
    </a>
  </div>
</div>
</body>
</html>`

	m.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(html))
}
