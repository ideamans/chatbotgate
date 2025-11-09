package middleware

import (
	"net/http"
)

// handlePasswordLogin handles password authentication
func (m *Middleware) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	if m.passwordHandler == nil {
		http.Error(w, "Password authentication not configured", http.StatusNotFound)
		return
	}

	m.passwordHandler.HandleLogin(w, r)
}
