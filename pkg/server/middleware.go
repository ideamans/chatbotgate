package server

import (
	"net/http"
	"strings"
)

const (
	sessionCookieName  = "session_id"
	contextKeySession  = "session"
	redirectCookieName = "_oauth2_redirect" // Cookie name for storing redirect URL
)

// staticResourcePaths are paths that should not trigger authentication or be saved as redirect URLs
var staticResourcePaths = []string{
	"/favicon.ico",
	"/robots.txt",
	"/apple-touch-icon.png",
	"/apple-touch-icon-precomposed.png",
}

// isStaticResource checks if the request path is for a static resource
func isStaticResource(path string) bool {
	for _, staticPath := range staticResourcePaths {
		if path == staticPath || strings.HasPrefix(path, staticPath) {
			return true
		}
	}
	return false
}

// authMiddleware checks if the user is authenticated
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for static resources
		if isStaticResource(r.URL.Path) {
			http.NotFound(w, r)
			return
		}

		// Get session cookie
		cookie, err := r.Cookie(s.config.Session.CookieName)
		if err != nil {
			// No session cookie, redirect to login
			s.redirectToLogin(w, r)
			return
		}

		// Get session from store
		sess, err := s.sessionStore.Get(cookie.Value)
		if err != nil {
			// Invalid session, redirect to login
			s.redirectToLogin(w, r)
			return
		}

		// Check if session is valid
		if !sess.IsValid() {
			// Expired session, redirect to login
			s.sessionStore.Delete(cookie.Value)
			s.redirectToLogin(w, r)
			return
		}

		// Session is valid, continue to next handler
		next.ServeHTTP(w, r)
	})
}

// redirectToLogin redirects to the login page and saves the original URL
func (s *Server) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	// Save the original URL to a cookie so we can redirect back after authentication
	// But skip saving static resources and only save if there's no existing redirect cookie
	originalURL := r.URL.String()

	// Don't save static resource paths
	if !isStaticResource(r.URL.Path) {
		// Only save if there's no existing redirect cookie (don't overwrite)
		if _, err := r.Cookie(redirectCookieName); err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     redirectCookieName,
				Value:    originalURL,
				Path:     "/",
				MaxAge:   600, // 10 minutes - enough time to complete authentication
				HttpOnly: true,
				Secure:   s.config.Session.CookieSecure,
				SameSite: http.SameSiteLaxMode,
			})
		}
	}

	loginPath := joinAuthPath(s.config.Server.GetAuthPathPrefix(), "/login")
	http.Redirect(w, r, loginPath, http.StatusFound)
}

// getRedirectURL retrieves and deletes the redirect URL from cookie
func (s *Server) getRedirectURL(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie(redirectCookieName)
	if err != nil {
		return "/" // Default to home if no redirect cookie
	}

	// Delete the redirect cookie
	http.SetCookie(w, &http.Cookie{
		Name:   redirectCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	redirectURL := cookie.Value

	// Security check: only allow relative URLs to prevent open redirect
	if redirectURL == "" || strings.HasPrefix(redirectURL, "//") || strings.Contains(redirectURL, "://") {
		return "/"
	}

	return redirectURL
}

func normalizeAuthPrefix(prefix string) string {
	if prefix == "" {
		return "/_auth"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if len(prefix) > 1 && strings.HasSuffix(prefix, "/") {
		prefix = strings.TrimSuffix(prefix, "/")
	}
	return prefix
}

func joinAuthPath(prefix, suffix string) string {
	normalized := normalizeAuthPrefix(prefix)
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	if normalized == "/" {
		return suffix
	}
	return normalized + suffix
}
