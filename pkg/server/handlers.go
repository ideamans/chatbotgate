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

// handleLogin displays the login page
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return s.translator.T(lang, key) }

	// Determine CSS file based on theme
	cssFile := "https://cdn.jsdelivr.net/npm/water.css@2/out/water.css"
	if theme == i18n.ThemeLight {
		cssFile = "https://cdn.jsdelivr.net/npm/water.css@2/out/light.css"
	} else if theme == i18n.ThemeDark {
		cssFile = "https://cdn.jsdelivr.net/npm/water.css@2/out/dark.css"
	}

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("login.title") + ` - ` + s.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssFile + `">
<style>
.ui-controls { margin: 20px 0; padding: 15px; background: rgba(0,0,0,0.05); border-radius: 5px; }
.ui-controls label { margin-right: 10px; }
.ui-controls select { margin-right: 15px; }
</style>
</head>
<body>
<div class="ui-controls">
	<label>` + t("ui.theme") + `: <select id="theme-select" onchange="changeTheme(this.value)">
		<option value="auto" ` + map[bool]string{true: `selected`, false: ``}[theme == i18n.ThemeAuto] + `>` + t("ui.theme.auto") + `</option>
		<option value="light" ` + map[bool]string{true: `selected`, false: ``}[theme == i18n.ThemeLight] + `>` + t("ui.theme.light") + `</option>
		<option value="dark" ` + map[bool]string{true: `selected`, false: ``}[theme == i18n.ThemeDark] + `>` + t("ui.theme.dark") + `</option>
	</select></label>

	<label>` + t("ui.language") + `: <select id="lang-select" onchange="changeLanguage(this.value)">
		<option value="en" ` + map[bool]string{true: `selected`, false: ``}[lang == i18n.English] + `>` + t("ui.language.en") + `</option>
		<option value="ja" ` + map[bool]string{true: `selected`, false: ``}[lang == i18n.Japanese] + `>` + t("ui.language.ja") + `</option>
	</select></label>
</div>

<h1>` + s.config.Service.Name + `</h1>
<p>` + s.config.Service.Description + `</p>
<h2>` + t("login.oauth2.heading") + `</h2>
<ul>`

	prefix := s.config.Server.GetAuthPathPrefix()
	providers := s.oauthManager.GetProviders()
	for _, p := range providers {
		html += fmt.Sprintf(`<li><a href="%s/oauth2/start/%s">%s</a></li>`, prefix, p.Name(), p.Name())
	}

	html += `</ul>`

	// Add email authentication form if enabled
	if s.emailHandler != nil {
		emailSendPath := joinAuthPath(s.config.Server.GetAuthPathPrefix(), "/email/send")
		html += `
<h2>` + t("login.email.heading") + `</h2>
<form method="POST" action="` + emailSendPath + `">
  <label for="email">` + t("login.email.label") + `:</label><br>
  <input type="email" id="email" name="email" required><br><br>
  <button type="submit">` + t("login.email.submit") + `</button>
</form>`
	}

	html += `
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
	window.location.reload();
}

function changeLanguage(lang) {
	setCookie("lang", lang, 365);
	window.location.reload();
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

	// Determine CSS file based on theme
	cssFile := "https://cdn.jsdelivr.net/npm/water.css@2/out/water.css"
	if theme == i18n.ThemeLight {
		cssFile = "https://cdn.jsdelivr.net/npm/water.css@2/out/light.css"
	} else if theme == i18n.ThemeDark {
		cssFile = "https://cdn.jsdelivr.net/npm/water.css@2/out/dark.css"
	}

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

	// Show logout page
	loginPath := joinAuthPath(s.config.Server.GetAuthPathPrefix(), "/login")

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("logout.title") + ` - ` + s.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssFile + `">
</head>
<body>
<h1>` + t("logout.heading") + `</h1>
<p>` + t("logout.message") + `</p>
<p><a href="` + loginPath + `">` + t("logout.login") + `</a></p>
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
		http.Error(w, "Access denied", http.StatusForbidden)
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

	// Determine CSS file based on theme
	cssFile := "https://cdn.jsdelivr.net/npm/water.css@2/out/water.css"
	if theme == i18n.ThemeLight {
		cssFile = "https://cdn.jsdelivr.net/npm/water.css@2/out/light.css"
	} else if theme == i18n.ThemeDark {
		cssFile = "https://cdn.jsdelivr.net/npm/water.css@2/out/dark.css"
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, t("error.invalid_request"), http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		http.Error(w, t("error.invalid_email"), http.StatusBadRequest)
		return
	}

	// Send login link
	err := s.emailHandler.SendLoginLink(email)
	if err != nil {
		s.logger.Error("Failed to send login link", "email", email, "error", err)
		// Don't reveal whether the email is authorized or not
		// Always show success message
	}

	// Show success message
	loginPath := joinAuthPath(s.config.Server.GetAuthPathPrefix(), "/login")

	html := `<!DOCTYPE html>
<html lang="` + string(lang) + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("email.sent.title") + ` - ` + s.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssFile + `">
</head>
<body>
<h1>` + t("email.sent.heading") + `</h1>
<p>` + t("email.sent.message") + `</p>
<p>` + t("email.sent.detail") + `</p>
<p><a href="` + loginPath + `">` + t("email.sent.back") + `</a></p>
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

		// Determine CSS file based on theme
		cssFile := "https://cdn.jsdelivr.net/npm/water.css@2/out/water.css"
		if theme == i18n.ThemeLight {
			cssFile = "https://cdn.jsdelivr.net/npm/water.css@2/out/light.css"
		} else if theme == i18n.ThemeDark {
			cssFile = "https://cdn.jsdelivr.net/npm/water.css@2/out/dark.css"
		}

		retryPath := joinAuthPath(s.config.Server.GetAuthPathPrefix(), "/email")

		html := `<!DOCTYPE html>
<html lang="` + string(lang) + `">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + t("email.invalid.title") + ` - ` + s.config.Service.Name + `</title>
<link rel="stylesheet" href="` + cssFile + `">
</head>
<body>
<h1>` + t("email.invalid.heading") + `</h1>
<p>` + t("email.invalid.message") + `</p>
<p><a href="` + retryPath + `">` + t("email.invalid.retry") + `</a></p>
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
