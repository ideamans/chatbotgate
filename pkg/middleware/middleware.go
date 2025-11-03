package middleware

import (
	"net/http"

	"github.com/ideamans/chatbotgate/pkg/auth/email"
	"github.com/ideamans/chatbotgate/pkg/auth/oauth2"
	"github.com/ideamans/chatbotgate/pkg/authz"
	"github.com/ideamans/chatbotgate/pkg/config"
	"github.com/ideamans/chatbotgate/pkg/forwarding"
	"github.com/ideamans/chatbotgate/pkg/i18n"
	"github.com/ideamans/chatbotgate/pkg/logging"
	"github.com/ideamans/chatbotgate/pkg/session"
)

// Middleware is the core authentication middleware
// It implements http.Handler and can wrap any http.Handler
type Middleware struct {
	config        *config.Config
	sessionStore  session.Store
	oauthManager  *oauth2.Manager
	emailHandler  *email.Handler
	authzChecker  authz.Checker
	forwarder     *forwarding.Forwarder
	translator    *i18n.Translator
	logger        logging.Logger
	next          http.Handler // The next handler to call after auth succeeds
}

// New creates a new authentication middleware
func New(
	cfg *config.Config,
	sessionStore session.Store,
	oauthManager *oauth2.Manager,
	emailHandler *email.Handler,
	authzChecker authz.Checker,
	forwarder *forwarding.Forwarder,
	translator *i18n.Translator,
	logger logging.Logger,
) *Middleware {
	return &Middleware{
		config:       cfg,
		sessionStore: sessionStore,
		oauthManager: oauthManager,
		emailHandler: emailHandler,
		authzChecker: authzChecker,
		forwarder:    forwarder,
		translator:   translator,
		logger:       logger,
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
	case matchPath(r.URL.Path, prefix, "/email/verify"):
		m.handleEmailVerify(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/assets/styles.css"):
		m.handleStylesCSS(w, r)
		return
	case matchPath(r.URL.Path, prefix, "/assets/icons/"):
		m.handleIcon(w, r)
		return
	case r.URL.Path == "/health":
		m.handleHealth(w, r)
		return
	case r.URL.Path == "/ready":
		m.handleReady(w, r)
		return
	}

	// Check authentication for all other paths
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
	sess, err := m.sessionStore.Get(cookie.Value)
	if err != nil || sess == nil {
		// Session not found, redirect to login
		m.redirectToLogin(w, r)
		return
	}

	// Check if session is valid
	if !sess.IsValid() {
		// Session expired or invalid, delete and redirect
		m.sessionStore.Delete(cookie.Value)
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
		w.Write([]byte("Authenticated"))
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
			http.SetCookie(w, &http.Cookie{
				Name:     redirectCookieName,
				Value:    originalURL,
				Path:     "/",
				MaxAge:   600, // 10 minutes - enough time to complete authentication
				HttpOnly: true,
				Secure:   m.config.Session.CookieSecure,
				SameSite: http.SameSiteLaxMode,
			})
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
		}

		// Add headers using forwarder (handles X-ChatbotGate-User, X-ChatbotGate-Email)
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
