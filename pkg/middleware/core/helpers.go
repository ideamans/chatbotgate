package middleware

import (
	"net/http"
	"strings"
)

const (
	redirectCookieName = "_oauth2_redirect" // Cookie name for storing redirect URL
)

// staticResourcePaths are paths that should not trigger authentication or be saved as redirect URLs
var staticResourcePaths = []string{
	"/favicon.ico",
	"/robots.txt",
	"/apple-touch-icon.png",
	"/apple-touch-icon-precomposed.png",
	"/_auth/assets/",  // Auth page assets (CSS, icons, etc.)
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

// getRedirectURL retrieves and deletes the redirect URL from cookie
func (m *Middleware) getRedirectURL(w http.ResponseWriter, r *http.Request) string {
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

// extractPathParam extracts a path parameter from the URL
// For example, extractPathParam(r.URL.Path, "/_auth/oauth2/start/") returns "google" from "/_auth/oauth2/start/google"
func extractPathParam(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	param := strings.TrimPrefix(path, prefix)
	// Remove any trailing slashes and query parameters
	if idx := strings.Index(param, "?"); idx != -1 {
		param = param[:idx]
	}
	param = strings.TrimSuffix(param, "/")
	return param
}
