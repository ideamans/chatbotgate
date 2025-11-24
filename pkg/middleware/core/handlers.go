package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/assets"
	"github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
	"github.com/ideamans/chatbotgate/pkg/middleware/forwarding"
	"github.com/ideamans/chatbotgate/pkg/middleware/session"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
)

// handleLogin displays the login page using html/template
func (m *Middleware) handleLogin(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }
	prefix := m.config.Server.GetAuthPathPrefix()

	// Build common page data
	pageData := m.buildPageData(lang, theme, "login.title")

	// Build provider data
	var providerDataList []ProviderData
	providers := m.oauthManager.GetProviders()
	for _, p := range providers {
		providerName := p.Name()

		// Find icon URL from config
		var iconPath string
		for _, providerCfg := range m.config.OAuth2.Providers {
			if providerCfg.Type == providerName && providerCfg.IconURL != "" {
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

		providerDataList = append(providerDataList, ProviderData{
			Name:     providerName,
			IconPath: iconPath,
			URL:      joinAuthPath(prefix, "/oauth2/start/"+providerName),
			Label:    fmt.Sprintf(t("login.oauth2.continue"), providerName),
		})
	}

	// Build login page data
	data := LoginPageData{
		PageData:        pageData,
		Providers:       providerDataList,
		EmailEnabled:    m.emailHandler != nil,
		PasswordEnabled: m.passwordHandler != nil,
		EmailSendPath:   joinAuthPath(prefix, "/email/send"),
		EmailIconPath:   joinAuthPath(prefix, "/assets/icons/email.svg"),
		Translations: LoginTranslations{
			Or:          t("login.or"),
			EmailLabel:  t("login.email.label"),
			EmailSave:   t("login.email.save"),
			EmailSubmit: t("login.email.submit"),
			ThemeAuto:   t("ui.theme.auto"),
			ThemeLight:  t("ui.theme.light"),
			ThemeDark:   t("ui.theme.dark"),
			LanguageEn:  t("ui.language.en"),
			LanguageJa:  t("ui.language.ja"),
		},
	}

	// Add password form HTML if enabled
	if m.passwordHandler != nil {
		data.PasswordFormHTML = template.HTML(m.passwordHandler.RenderPasswordForm(lang))
	}

	// Render template
	if err := renderTemplate(w, m.templates.login, data, m); err != nil {
		m.logger.Error("Failed to render login template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handleLogout logs out the user using html/template
func (m *Middleware) handleLogout(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }
	prefix := m.config.Server.GetAuthPathPrefix()

	// Get session cookie
	cookie, err := r.Cookie(m.config.Session.Cookie.Name)
	if err == nil {
		// Delete session (ignore error, proceed with logout anyway)
		_ = session.Delete(m.sessionStore, cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     m.config.Session.Cookie.Name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Build page data
	pageData := m.buildPageData(lang, theme, "logout.title")
	pageData.Subtitle = t("logout.heading")

	data := LogoutPageData{
		PageData:   pageData,
		Message:    t("logout.message"),
		LoginURL:   joinAuthPath(prefix, "/login"),
		LoginLabel: t("logout.login"),
	}

	// Render template
	if err := renderTemplate(w, m.templates.logout, data, m); err != nil {
		m.logger.Error("Failed to render logout template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handleEmailSent shows the email sent confirmation page using html/template
func (m *Middleware) handleEmailSent(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }
	prefix := m.config.Server.GetAuthPathPrefix()

	// Build page data
	pageData := m.buildPageData(lang, theme, "email.sent.title")
	pageData.Subtitle = t("email.sent.heading")

	data := EmailSentPageData{
		PageData:       pageData,
		Message:        t("email.sent.message"),
		Detail:         t("email.sent.detail"),
		OTPLabel:       t("email.sent.otp_label"),
		OTPPlaceholder: t("email.sent.otp_placeholder"),
		VerifyButton:   t("email.sent.verify_button"),
		BackLabel:      t("email.sent.back"),
		LoginURL:       joinAuthPath(prefix, "/login"),
		VerifyOTPPath:  joinAuthPath(prefix, "/email/verify-otp"),
	}

	// Render template
	if err := renderTemplate(w, m.templates.emailSent, data, m); err != nil {
		m.logger.Error("Failed to render email sent template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handleForbidden displays the access denied page using html/template
func (m *Middleware) handleForbidden(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }
	prefix := m.config.Server.GetAuthPathPrefix()

	// Build page data
	pageData := m.buildPageData(lang, theme, "error.forbidden.title")
	pageData.Subtitle = t("error.forbidden.heading")

	data := ErrorPageData{
		PageData:    pageData,
		Message:     t("error.forbidden.message"),
		ActionURL:   joinAuthPath(prefix, "/login"),
		ActionLabel: t("login.back"),
	}

	// Render template
	if err := renderErrorTemplate(w, m.templates.forbidden, data, http.StatusForbidden, m); err != nil {
		m.logger.Error("Failed to render forbidden template", "error", err)
		http.Error(w, "Access Denied", http.StatusForbidden)
		return
	}
}

// handleEmailFetchError displays an error page when OAuth2 provider fails to provide email
func (m *Middleware) handleEmailFetchError(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }
	prefix := m.config.Server.GetAuthPathPrefix()

	// Build page data
	pageData := m.buildPageData(lang, theme, "error.email_required.title")
	pageData.Subtitle = t("error.email_required.heading")

	data := ErrorPageData{
		PageData:    pageData,
		Message:     t("error.email_required.message"),
		ActionURL:   joinAuthPath(prefix, "/login"),
		ActionLabel: t("login.back"),
	}

	// Render template
	if err := renderErrorTemplate(w, m.templates.emailReq, data, http.StatusBadRequest, m); err != nil {
		m.logger.Error("Failed to render email required template", "error", err)
		http.Error(w, "Email required", http.StatusBadRequest)
		return
	}
}

// handle404 displays the 404 Not Found page using html/template
func (m *Middleware) handle404(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	// Build page data
	pageData := m.buildPageData(lang, theme, "error.notfound.title")
	pageData.Subtitle = t("error.notfound.heading")

	data := ErrorPageData{
		PageData:    pageData,
		Message:     t("error.notfound.message"),
		ActionURL:   "/",
		ActionLabel: t("error.notfound.home"),
	}

	// Render template
	if err := renderErrorTemplate(w, m.templates.notFound, data, http.StatusNotFound, m); err != nil {
		m.logger.Error("Failed to render 404 template", "error", err)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
}

// handle500 displays the 500 Internal Server Error page with optional error details
func (m *Middleware) handle500(w http.ResponseWriter, r *http.Request, err error) {
	lang := i18n.DetectLanguage(r)
	theme := i18n.DetectTheme(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	// Build page data
	pageData := m.buildPageData(lang, theme, "error.server.title")
	pageData.Subtitle = t("error.server.heading")

	data := ErrorPageData{
		PageData:    pageData,
		Message:     t("error.server.message"),
		ActionURL:   "/",
		ActionLabel: t("error.server.home"),
	}

	// Build error details accordion if error is provided
	if err != nil {
		errorDetailsHTML := `
    <div class="accordion" id="error-accordion">
      <div class="accordion-header" onclick="document.getElementById('error-accordion').classList.toggle('open')">
        <span class="accordion-header-title">` + template.HTMLEscapeString(t("error.details.title")) + `</span>
        <span class="accordion-header-icon"></span>
      </div>
      <div class="accordion-content">
        <div class="accordion-body">` + template.HTMLEscapeString(fmt.Sprintf("%+v", err)) + `</div>
      </div>
    </div>`
		data.ErrorDetails = template.HTML(errorDetailsHTML)
	}

	// Render template
	if renderErr := renderErrorTemplate(w, m.templates.server, data, http.StatusInternalServerError, m); renderErr != nil {
		m.logger.Error("Failed to render 500 template", "error", renderErr)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// HealthResponse represents the JSON response for health check
type HealthResponse struct {
	Status     string `json:"status"`      // Current health status (starting/ready/draining/etc.)
	Live       bool   `json:"live"`        // Process is alive
	Ready      bool   `json:"ready"`       // Ready to accept traffic
	Since      string `json:"since"`       // ISO8601 timestamp of when middleware started
	Detail     string `json:"detail"`      // Human-readable detail message
	RetryAfter *int   `json:"retry_after"` // Retry after N seconds (only present when 503)
}

// Health Check Strategy
// ======================
//
// ChatbotGate uses a unified /_auth/health endpoint for all health checks, supporting both
// readiness and liveness probes through a single URL with minimal complexity.
//
// Endpoints:
//   - /_auth/health           → Readiness probe (default)
//   - /_auth/health?probe=live → Liveness probe
//
// Readiness vs Liveness:
//   - Readiness: Returns 200 when ready to accept traffic, 503 when starting/draining
//   - Liveness:  Always returns 200 if process is alive (no dependency checks)
//
// Health States:
//   - starting   → Initial state after middleware creation
//   - ready      → Middleware is ready (after SetReady() call)
//   - draining   → Graceful shutdown in progress (after SetDraining() call)
//   - warming    → (Reserved for future use, e.g., cache warming)
//   - migrating  → (Reserved for future use, e.g., data migration)
//   - prefilling → (Reserved for future use, e.g., connection pool setup)
//
// Response Format:
//   - 200 OK: Ready to accept traffic (ready=true)
//   - 503 Service Unavailable: Not ready (ready=false) with Retry-After header
//   - Always returns JSON with status details for both success and failure
//
// Lifecycle:
//   1. Middleware created → status="starting", ready=false
//   2. Initialization complete → SetReady() → status="ready", ready=true
//   3. SIGTERM received → SetDraining() → status="draining", ready=false
//   4. Server shutdown → connections drained → process exit
//
// Container Orchestration:
//   - Docker/ECS: Use /_auth/health for health checks
//   - Kubernetes: Use /_auth/health for readinessProbe, /_auth/health?probe=live for livenessProbe
//   - ALB/NLB: Use /_auth/health with 200 as healthy status
//
// Graceful Shutdown:
//   When SIGTERM is received:
//   1. SetDraining() is called → /_auth/health returns 503
//   2. Load balancers detect 503 and stop sending new requests
//   3. Existing requests are allowed to complete
//   4. Server shuts down cleanly
//
// See also:
//   - middleware.go: SetReady(), SetDraining(), health state management
//   - server/middleware_manager.go: Lifecycle management
//   - README.md: Deployment examples and configuration

// handleHealth handles the health check endpoint (/_auth/health)
// Supports both readiness check (default) and liveness check (?probe=live)
// See: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
func (m *Middleware) handleHealth(w http.ResponseWriter, r *http.Request) {
	// IMPORTANT: Health checks must only accept GET and HEAD methods
	// This follows HTTP spec and Kubernetes/Docker health check conventions
	// Health checks are read-only operations and should use safe, idempotent methods
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":  "Method Not Allowed",
			"detail": "Health check endpoint only accepts GET and HEAD methods",
		})
		return
	}

	probe := r.URL.Query().Get("probe")

	// Liveness probe: just check if process is alive (no dependency checks)
	if probe == "live" {
		m.handleLiveness(w, r)
		return
	}

	// Default: Readiness probe (check if ready to accept traffic)
	m.handleReadiness(w, r)
}

// handleLiveness handles liveness probe (/_auth/health?probe=live)
// Returns 200 if the process is alive (no dependency checks)
func (m *Middleware) handleLiveness(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "live",
		Live:   m.healthLive.Load(),
		Ready:  m.healthReady.Load(), // Include ready status for visibility
		Since:  m.healthStarted.Format(time.RFC3339),
		Detail: "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// handleReadiness handles readiness probe (/_auth/health)
// Returns 200 if ready to accept traffic, 503 otherwise
func (m *Middleware) handleReadiness(w http.ResponseWriter, r *http.Request) {
	ready := m.healthReady.Load()
	status := m.GetHealthStatus()
	live := m.healthLive.Load()

	response := HealthResponse{
		Status: string(status),
		Live:   live,
		Ready:  ready,
		Since:  m.healthStarted.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")

	if ready {
		// Ready to accept traffic
		response.Detail = "ok"
		w.WriteHeader(http.StatusOK)
	} else {
		// Not ready yet (starting, warming, draining, etc.)
		retryAfter := 5
		response.Detail = "warming up"
		response.RetryAfter = &retryAfter

		w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	_ = json.NewEncoder(w).Encode(response)
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
		Secure:   m.config.Session.Cookie.Secure,
		SameSite: m.config.Session.Cookie.GetSameSite(),
	})

	// Store provider in cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_provider",
		Value:    providerName,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   m.config.Session.Cookie.Secure,
		SameSite: m.config.Session.Cookie.GetSameSite(),
	})

	// Store redirect URL in cookie for token exchange
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_redirect_url",
		Value:    redirectURL,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   m.config.Session.Cookie.Secure,
		SameSite: m.config.Session.Cookie.GetSameSite(),
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
	if oldCookie, err := r.Cookie(m.config.Session.Cookie.Name); err == nil {
		_ = session.Delete(m.sessionStore, oldCookie.Value)
	}

	// Create session with new session ID
	sessionID, err := generateSessionID()
	if err != nil {
		m.logger.Error("Failed to generate session ID", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	duration, err := m.config.Session.Cookie.GetExpireDuration()
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
		Name:     m.config.Session.Cookie.Name,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(duration.Seconds()),
		HttpOnly: m.config.Session.Cookie.HTTPOnly,
		Secure:   m.config.Session.Cookie.Secure,
		SameSite: m.config.Session.Cookie.GetSameSite(),
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
	n, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	// Verify we got the expected number of random bytes
	if n != 32 {
		return "", fmt.Errorf("insufficient entropy for session ID generation: got %d bytes, expected 32", n)
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

	// Get redirect URL from cookie (where user originally wanted to go)
	redirectURL := "/"
	if cookie, err := r.Cookie(redirectCookieName); err == nil && cookie.Value != "" {
		if isValidRedirectURL(cookie.Value) {
			redirectURL = cookie.Value
		}
	}

	// Send login link with redirect URL embedded in token
	err := m.emailHandler.SendLoginLink(email, redirectURL, lang)
	if err != nil {
		m.logger.Debug("Email send failed", "email", maskEmail(email), "error", err)

		// Check if this is a rate limit error
		if strings.Contains(err.Error(), "rate limit exceeded") {
			m.logger.Warn("Email authentication rate limited", "email", maskEmail(email))
			http.Error(w, t("error.rate_limit"), http.StatusTooManyRequests)
			return
		}

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
func (m *Middleware) handleEmailVerify(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	t := func(key string) string { return m.translator.T(lang, key) }

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, t("error.invalid_request"), http.StatusBadRequest)
		return
	}

	// Verify token and get redirect URL
	email, redirectURL, err := m.emailHandler.VerifyToken(token)
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
	if oldCookie, err := r.Cookie(m.config.Session.Cookie.Name); err == nil {
		_ = session.Delete(m.sessionStore, oldCookie.Value)
	}

	// Create session with new session ID
	sessionID, err := generateSessionID()
	if err != nil {
		m.logger.Error("Failed to generate session ID", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	duration, err := m.config.Session.Cookie.GetExpireDuration()
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
		Name:     m.config.Session.Cookie.Name,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(duration.Seconds()),
		HttpOnly: m.config.Session.Cookie.HTTPOnly,
		Secure:   m.config.Session.Cookie.Secure,
		SameSite: m.config.Session.Cookie.GetSameSite(),
	})

	m.logger.Info("Email authentication successful", "email", maskEmail(email))

	// Use redirect URL from token, or fall back to cookie or home page
	if redirectURL == "" {
		redirectURL = m.getRedirectURL(w, r)
	} else {
		// Still delete the redirect cookie if it exists
		http.SetCookie(w, &http.Cookie{
			Name:   redirectCookieName,
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
	}

	// Validate redirect URL to prevent open redirect attacks
	if !isValidRedirectURL(redirectURL) {
		redirectURL = "/"
	}

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

	// Verify OTP and get redirect URL
	email, redirectURL, err := m.emailHandler.VerifyOTP(otp)
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
	if oldCookie, err := r.Cookie(m.config.Session.Cookie.Name); err == nil {
		_ = session.Delete(m.sessionStore, oldCookie.Value)
	}

	// Create session with new session ID
	sessionID, err := generateSessionID()
	if err != nil {
		m.logger.Error("Failed to generate session ID", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	duration, err := m.config.Session.Cookie.GetExpireDuration()
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
		Name:     m.config.Session.Cookie.Name,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(duration.Seconds()),
		HttpOnly: m.config.Session.Cookie.HTTPOnly,
		Secure:   m.config.Session.Cookie.Secure,
		SameSite: m.config.Session.Cookie.GetSameSite(),
	})

	m.logger.Info("Email authentication successful via OTP", "email", maskEmail(email))

	// Use redirect URL from token, or fall back to cookie or home page
	if redirectURL == "" {
		redirectURL = m.getRedirectURL(w, r)
	} else {
		// Still delete the redirect cookie if it exists
		http.SetCookie(w, &http.Cookie{
			Name:   redirectCookieName,
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
	}

	// Validate redirect URL to prevent open redirect attacks
	if !isValidRedirectURL(redirectURL) {
		redirectURL = "/"
	}

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

// buildStyleLinks generates stylesheet link tags (temporary wrapper for backward compatibility)
func (m *Middleware) buildStyleLinks() string {
	return m.buildStyleLinksHTML()
}

// buildAuthHeader generates auth header HTML (temporary wrapper for backward compatibility)
func (m *Middleware) buildAuthHeader(prefix string) string {
	return m.buildAuthHeaderHTML(prefix)
}

// buildAuthSubtitle generates auth subtitle HTML
func (m *Middleware) buildAuthSubtitle(subtitle string) string {
	if subtitle == "" {
		return ""
	}
	return `<h2 class="auth-subtitle">` + template.HTMLEscapeString(subtitle) + `</h2>`
}
