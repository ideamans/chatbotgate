package middleware

import (
	"net/http"

	"github.com/ideamans/chatbotgate/pkg/middleware/auth/email"
	"github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
	"github.com/ideamans/chatbotgate/pkg/middleware/authz"
	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/middleware/forwarding"
	"github.com/ideamans/chatbotgate/pkg/middleware/rules"
	"github.com/ideamans/chatbotgate/pkg/middleware/session"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// Middleware is the core authentication middleware
// It implements http.Handler and can wrap any http.Handler
type Middleware struct {
	config         *config.Config
	sessionStore   kvs.Store
	oauthManager   *oauth2.Manager
	emailHandler   *email.Handler
	authzChecker   authz.Checker
	forwarder      forwarding.Forwarder // Interface type
	rulesEvaluator *rules.Evaluator     // Rules-based access control
	translator     *i18n.Translator
	logger         logging.Logger
	next           http.Handler // The next handler to call after auth succeeds
}

// New creates a new authentication middleware
func New(
	cfg *config.Config,
	sessionStore kvs.Store,
	oauthManager *oauth2.Manager,
	emailHandler *email.Handler,
	authzChecker authz.Checker,
	forwarder forwarding.Forwarder, // Interface type
	rulesEvaluator *rules.Evaluator, // Rules evaluator
	translator *i18n.Translator,
	logger logging.Logger,
) *Middleware {
	return &Middleware{
		config:         cfg,
		sessionStore:   sessionStore,
		oauthManager:   oauthManager,
		emailHandler:   emailHandler,
		authzChecker:   authzChecker,
		forwarder:      forwarder,
		rulesEvaluator: rulesEvaluator,
		translator:     translator,
		logger:         logger,
	}
}

// Wrap wraps a http.Handler with authentication
// This is the main entry point for using the middleware
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	m.next = next
	return m
}

// ServeHTTP implements http.Handler
// This is where all requests pass through
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prefix := m.config.Server.GetAuthPathPrefix()

	// Handle authentication endpoints
	switch {
	case matchPath(r.URL.Path, prefix, "/login"):
		m.handleLogin(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/logout"):
		m.handleLogout(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/oauth2/start/"):
		m.handleOAuth2Start(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/oauth2/callback"):
		m.handleOAuth2Callback(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/email/send"):
		m.handleEmailSend(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/email/sent"):
		m.handleEmailSent(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/email/verify"):
		m.handleEmailVerify(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/email/verify-otp"):
		m.handleEmailVerifyOTP(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/assets/main.css"):
		m.handleMainCSS(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/assets/dify.css"):
		m.handleDifyCSS(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/assets/icons/"):
		m.handleIcon(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/404"):
		m.handle404(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/500"):
		m.handle500(w, r, nil)
		return
	case r.URL.Path == "/health":
		m.handleHealth(w, r)
		return
	case r.URL.Path == "/ready":
		m.handleReady(w, r)
		return
	}

	// Evaluate access rules for the path
	if m.rulesEvaluator != nil {
		action := m.rulesEvaluator.Evaluate(r.URL.Path)
		switch action {
		case rules.ActionAllow:
			// Allow access without authentication
			m.logger.Debug("Rules: allowing without authentication", "path", r.URL.Path, "action", action)
			if m.next != nil {
				m.next.ServeHTTP(w, r)
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("Allowed"))
			}
			return

		case rules.ActionDeny:
			// Deny access (403)
			m.logger.Debug("Rules: denying access", "path", r.URL.Path, "action", action)
			http.Error(w, "Access Denied", http.StatusForbidden)
			return

		case rules.ActionAuth:
			// Require authentication (default behavior)
			m.logger.Debug("Rules: requiring authentication", "path", r.URL.Path, "action", action)
			m.requireAuth(w, r)
			return
		}
	}

	// If no rules evaluator, default to requiring authentication
	m.requireAuth(w, r)
}

// requireAuth checks if the user is authenticated
// If yes, calls the next handler
// If no, redirects to login
func (m *Middleware) requireAuth(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	cookie, err := r.Cookie(m.config.Session.CookieName)
	if err != nil {
		// No session cookie, redirect to login
		m.redirectToLogin(w, r)
		return
	}

	// Get session from store
	sess, err := session.Get(m.sessionStore, cookie.Value)
	if err != nil || sess == nil {
		// Session not found, redirect to login
		m.redirectToLogin(w, r)
		return
	}

	// Check if session is valid
	if !sess.IsValid() {
		// Session expired or invalid, delete and redirect
		_ = session.Delete(m.sessionStore, cookie.Value)
		m.redirectToLogin(w, r)
		return
	}

	// Session is valid, add auth headers and call next handler
	m.addAuthHeaders(r, sess)

	if m.next != nil {
		m.next.ServeHTTP(w, r)
	} else {
		// If no next handler, return 200 OK (useful for testing)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Authenticated"))
	}
}

// redirectToLogin redirects to the login page with the original URL
func (m *Middleware) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	prefix := m.config.Server.GetAuthPathPrefix()
	loginPath := joinAuthPath(prefix, "/login")

	// Store original URL in cookie for redirect after authentication
	// Don't save static resource paths
	originalURL := r.URL.RequestURI()
	if !isStaticResource(r.URL.Path) && originalURL != "" && originalURL != "/" {
		// Only save if there's no existing redirect cookie (don't overwrite)
		if _, err := r.Cookie(redirectCookieName); err != nil {
			// Validate redirect URL to prevent open redirect attacks
			if isValidRedirectURL(originalURL) {
				http.SetCookie(w, &http.Cookie{
					Name:     redirectCookieName,
					Value:    originalURL,
					Path:     "/",
					MaxAge:   600, // 10 minutes - enough time to complete authentication
					HttpOnly: true,
					Secure:   m.config.Session.CookieSecure,
					SameSite: m.config.Session.GetCookieSameSite(),
				})
			}
		}
	}

	http.Redirect(w, r, loginPath, http.StatusFound)
}

// addAuthHeaders adds authentication headers to the request
func (m *Middleware) addAuthHeaders(r *http.Request, sess *session.Session) {
	// Add authentication status headers
	r.Header.Set("X-Authenticated", "true")
	r.Header.Set("X-Auth-Provider", sess.Provider)

	// Add forwarding headers (X-Forwarded-*) only if configured
	if m.forwarder != nil {
		userInfo := &forwarding.UserInfo{
			Username: sess.Name, // For email auth, this will be empty
			Email:    sess.Email,
			Extra:    sess.Extra,    // Additional OAuth2 data for custom forwarding
			Provider: sess.Provider, // Provider name for provider-specific forwarding
		}

		// Add headers using forwarder (handles X-ChatbotGate-User, X-ChatbotGate-Email, and custom fields)
		// Can be plain text or encrypted depending on configuration
		r.Header = m.forwarder.AddToHeaders(r.Header, userInfo)
	}
}

// matchPath checks if the request path matches the auth endpoint
func matchPath(requestPath, prefix, endpoint string) bool {
	fullPath := joinAuthPath(prefix, endpoint)
	if endpoint[len(endpoint)-1] == '/' {
		// Prefix match for endpoints like "/oauth2/start/"
		return len(requestPath) >= len(fullPath) && requestPath[:len(fullPath)] == fullPath
	}
	// Exact match
	return requestPath == fullPath
}

// joinAuthPath joins auth prefix and endpoint path
func joinAuthPath(prefix, endpoint string) string {
	// Normalize prefix
	if prefix == "" {
		prefix = "/_auth"
	}
	if prefix[len(prefix)-1] == '/' {
		prefix = prefix[:len(prefix)-1]
	}
	if prefix[0] != '/' {
		prefix = "/" + prefix
	}

	// Normalize endpoint
	if endpoint[0] != '/' {
		endpoint = "/" + endpoint
	}

	return prefix + endpoint
}
