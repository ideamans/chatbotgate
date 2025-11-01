package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ideamans/multi-oauth2-proxy/pkg/auth/oauth2"
	"github.com/ideamans/multi-oauth2-proxy/pkg/i18n"
	proxypkg "github.com/ideamans/multi-oauth2-proxy/pkg/proxy"
	"github.com/ideamans/multi-oauth2-proxy/pkg/session"
)

// handleHealth handles the health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleReady handles the readiness check endpoint
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("READY"))
}

// handleStylesCSS serves the embedded CSS
func (s *Server) handleStylesCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(GetEmbeddedCSS()))
}

// handleIcon serves the embedded SVG icons
func (s *Server) handleIcon(w http.ResponseWriter, r *http.Request) {
	iconName := chi.URLParam(r, "icon")
	iconPath := "static/icons/" + iconName

	// Read the icon file from embedded filesystem
	data, err := GetEmbeddedIcons().ReadFile(iconPath)
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
func (s *Server) buildAuthHeader(prefix string) string {
	serviceName := s.config.Service.Name
	iconURL := s.config.Service.IconURL
	logoURL := s.config.Service.LogoURL
	logoWidth := s.config.Service.LogoWidth
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
func (s *Server) buildAuthSubtitle(subtitle string) string {
	if subtitle == "" {
		return ""
	}
	return `<h2 class="auth-subtitle">` + subtitle + `</h2>`
}

// handleLogin displays the login page
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return s.translator.T(lang, key) }

	// Use embedded CSS
	cssPath := joinAuthPath(s.config.Server.GetAuthPathPrefix(), "/assets/styles.css")
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
<title>` + t("login.title") + ` - ` + s.config.Service.Name + `</title>
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
		` + s.buildAuthHeader(s.config.Server.GetAuthPathPrefix()) + `
		<p class="auth-description">` + s.config.Service.Description + `</p>`

	prefix := s.config.Server.GetAuthPathPrefix()
	providers := s.oauthManager.GetProviders()

	if len(providers) > 0 {
		html += `<div style="margin-bottom: var(--spacing-lg);">`
		for _, p := range providers {
			providerName := p.Name()
			// Map provider names to icon files
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
			iconPath := joinAuthPath(prefix, "/assets/icons/"+iconName+".svg")
			html += `<a href="` + prefix + `/oauth2/start/` + providerName + `" class="btn btn-secondary provider-btn">`
			html += `<img src="` + iconPath + `" alt="` + providerName + `">`
			html += fmt.Sprintf(t("login.oauth2.continue"), providerName) + `</a>`
		}
		html += `</div>`
	}

	// Add email authentication form if enabled
	if s.emailHandler != nil {
		emailSendPath := joinAuthPath(s.config.Server.GetAuthPathPrefix(), "/email/send")
		emailIconPath := joinAuthPath(prefix, "/assets/icons/email.svg")

		if len(providers) > 0 {
			html += `<div class="auth-divider"><span>` + t("login.or") + `</span></div>`
		}

		html += `
		<form method="POST" action="` + emailSendPath + `">
			<div class="form-group">
				<label class="label" for="email">` + t("login.email.label") + `</label>
				<input type="email" id="email" name="email" class="input" placeholder="you@example.com" required>
			</div>
			<button type="submit" class="btn btn-primary provider-btn">
				<img src="` + emailIconPath + `" alt="Email">
				` + t("login.email.submit") + `
			</button>
		</form>`
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
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return s.translator.T(lang, key) }

	// Get session cookie
	cookie, err := r.Cookie(s.config.Session.CookieName)
	if err == nil {
		// Delete session
		s.sessionStore.Delete(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.Session.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Use embedded CSS
	prefix := s.config.Server.GetAuthPathPrefix()
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
<title>` + t("logout.title") + ` - ` + s.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssPath + `">
</head>
<body>
<div class="auth-container">
	<div class="card auth-card">
		` + s.buildAuthHeader(prefix) + `
		` + s.buildAuthSubtitle(t("logout.heading")) + `
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
func (s *Server) handleOAuth2Start(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")

	// Generate state for CSRF protection
	state, err := oauth2.GenerateState()
	if err != nil {
		s.logger.Error("Failed to generate state", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store state in session (simplified for now - in production, use a dedicated state store)
	// For now, we'll pass it directly and verify in callback

	// Get auth URL
	authURL, err := s.oauthManager.GetAuthURL(providerName, state)
	if err != nil {
		s.logger.Error("Failed to get auth URL", "provider", providerName, "error", err)
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
		Secure:   s.config.Session.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Store provider in cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_provider",
		Value:    providerName,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   s.config.Session.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to OAuth2 provider
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleOAuth2Callback handles the OAuth2 callback
func (s *Server) handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	// Get state from cookie
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		s.logger.Error("State cookie not found")
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Get provider from cookie
	providerCookie, err := r.Cookie("oauth_provider")
	if err != nil {
		s.logger.Error("Provider cookie not found")
		http.Error(w, "Invalid provider", http.StatusBadRequest)
		return
	}

	// Verify state
	state := r.URL.Query().Get("state")
	if state != stateCookie.Value {
		s.logger.Error("State mismatch")
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		s.logger.Error("Authorization code not found")
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}

	providerName := providerCookie.Value

	// Exchange code for token
	token, err := s.oauthManager.Exchange(r.Context(), providerName, code)
	if err != nil {
		s.logger.Error("Failed to exchange code", "error", err)
		http.Error(w, "Failed to authenticate", http.StatusInternalServerError)
		return
	}

	// Get user email
	email, err := s.oauthManager.GetUserEmail(r.Context(), providerName, token)
	if err != nil {
		s.logger.Error("Failed to get user email", "error", err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Check authorization
	if !s.authzChecker.IsAllowed(email) {
		s.logger.Warn("User not authorized", "email", email)
		s.handleForbidden(w, r)
		return
	}

	// Create session
	sessionID, err := generateSessionID()
	if err != nil {
		s.logger.Error("Failed to generate session ID", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	duration, err := s.config.Session.GetCookieExpireDuration()
	if err != nil {
		duration = 168 * time.Hour // Default 7 days
	}

	sess := &session.Session{
		ID:            sessionID,
		Email:         email,
		Provider:      providerName,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(duration),
		Authenticated: true,
	}

	// Store session
	if err := s.sessionStore.Set(sessionID, sess); err != nil {
		s.logger.Error("Failed to store session", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.Session.CookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(duration.Seconds()),
		HttpOnly: s.config.Session.CookieHTTPOnly,
		Secure:   s.config.Session.CookieSecure,
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

	s.logger.Info("User authenticated", "email", email, "provider", providerName)

	// Redirect to original URL or home
	http.Redirect(w, r, s.getRedirectURL(w, r), http.StatusFound)
}

// handleProxy proxies the request to the backend
func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	// Get session from cookie
	cookie, err := r.Cookie(s.config.Session.CookieName)
	if err != nil {
		// Should not happen due to middleware, but handle it anyway
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sess, err := s.sessionStore.Get(cookie.Value)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Add authentication headers
	proxypkg.AddAuthHeaders(r, sess.Email, sess.Provider)

	// Proxy the request
	s.proxyHandler.ServeHTTP(w, r)
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
func (s *Server) handleEmailSend(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return s.translator.T(lang, key) }

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
	if !s.authzChecker.IsAllowed(email) {
		s.logger.Warn("User not authorized for email login", "email", email)
		s.handleForbidden(w, r)
		return
	}

	// Send login link
	err := s.emailHandler.SendLoginLink(email)
	if err != nil {
		s.logger.Error("Failed to send login link", "email", email, "error", err)
		http.Error(w, t("error.internal"), http.StatusInternalServerError)
		return
	}

	// Use embedded CSS
	prefix := s.config.Server.GetAuthPathPrefix()
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
<title>` + t("email.sent.title") + ` - ` + s.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssPath + `">
</head>
<body>
<div class="auth-container">
	<div class="card auth-card">
		` + s.buildAuthHeader(prefix) + `
		` + s.buildAuthSubtitle(t("email.sent.heading")) + `
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
func (s *Server) handleEmailVerify(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	t := func(key string) string { return s.translator.T(lang, key) }

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, t("error.invalid_request"), http.StatusBadRequest)
		return
	}

	// Verify token
	email, err := s.emailHandler.VerifyToken(token)
	if err != nil {
		s.logger.Error("Failed to verify token", "error", err)
		theme := i18n.DetectTheme(r)

		// Use embedded CSS
		prefix := s.config.Server.GetAuthPathPrefix()
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
<title>` + t("email.invalid.title") + ` - ` + s.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssPath + `">
</head>
<body>
<div class="auth-container">
	<div class="card auth-card">
		` + s.buildAuthHeader(prefix) + `
		` + s.buildAuthSubtitle(t("email.invalid.heading")) + `
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

	// Create session
	sessionID, err := generateSessionID()
	if err != nil {
		s.logger.Error("Failed to generate session ID", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	duration, err := s.config.Session.GetCookieExpireDuration()
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
	if err := s.sessionStore.Set(sessionID, sess); err != nil {
		s.logger.Error("Failed to store session", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.Session.CookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(duration.Seconds()),
		HttpOnly: s.config.Session.CookieHTTPOnly,
		Secure:   s.config.Session.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	s.logger.Info("User authenticated via email", "email", email)

	// Redirect to original URL or home
	http.Redirect(w, r, s.getRedirectURL(w, r), http.StatusFound)
}

// handleForbidden displays the access denied page
func (s *Server) handleForbidden(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return s.translator.T(lang, key) }

	prefix := normalizeAuthPrefix(s.config.Server.GetAuthPathPrefix())
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
<title>` + t("error.forbidden.title") + ` - ` + s.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssPath + `">
</head>
<body>
<div class="auth-container">
  <div class="card auth-card">
    ` + s.buildAuthHeader(prefix) + `
    ` + s.buildAuthSubtitle(t("error.forbidden.heading")) + `
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
