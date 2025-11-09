package password

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/middleware/session"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// Handler handles password authentication
type Handler struct {
	config         config.PasswordAuthConfig
	sessionStore   kvs.Store
	cookieConfig   config.CookieConfig
	authPathPrefix string
	translator     *i18n.Translator
	logger         logging.Logger
}

// NewHandler creates a new password authentication handler
func NewHandler(cfg config.PasswordAuthConfig, sessionStore kvs.Store, cookieConfig config.CookieConfig, authPathPrefix string, translator *i18n.Translator, logger logging.Logger) *Handler {
	return &Handler{
		config:         cfg,
		sessionStore:   sessionStore,
		cookieConfig:   cookieConfig,
		authPathPrefix: authPathPrefix,
		translator:     translator,
		logger:         logger,
	}
}

// HandleLogin handles the password login
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON body
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse password request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Check if password field is empty
	if req.Password == "" {
		h.logger.Warn("Empty password field")
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	// Validate password
	if req.Password != h.config.Password {
		h.logger.Warn("Invalid password attempt")
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	// Create session
	sessionID := generateSessionID()
	sess := &session.Session{
		ID:       sessionID,
		Email:    "password@localhost", // Fixed email for password auth
		Name:     "Password User",
		Provider: "password",
		Extra: map[string]interface{}{
			"_email":      "password@localhost",
			"_username":   "Password User",
			"_avatar_url": "",
			"auth_time":   time.Now().Format(time.RFC3339),
		},
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(7 * 24 * time.Hour), // 7 days default
		Authenticated: true,
	}

	// Save session
	if err := session.Set(h.sessionStore, sessionID, sess); err != nil {
		h.logger.Error("Failed to save session", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set cookie
	expireDuration, _ := h.cookieConfig.GetExpireDuration()
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieConfig.Name,
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(expireDuration),
		Secure:   h.cookieConfig.Secure,
		HttpOnly: h.cookieConfig.HTTPOnly,
		SameSite: h.cookieConfig.GetSameSite(),
	})

	h.logger.Info("Password authentication successful, session created", "session_id", sessionID)

	// Return redirect URL
	redirectURL := r.URL.Query().Get("redirect")
	if redirectURL == "" {
		redirectURL = "/"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"redirect_url": redirectURL,
	}); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	return fmt.Sprintf("pwd_%d", time.Now().UnixNano())
}

// RenderPasswordForm renders the password form HTML
func (h *Handler) RenderPasswordForm(lang i18n.Language) string {
	passwordLabel := h.translator.T(lang, "password.label")
	if passwordLabel == "password.label" {
		passwordLabel = "Password"
	}

	buttonText := h.translator.T(lang, "password.button")
	if buttonText == "password.button" {
		buttonText = "Sign In"
	}

	// Normalize auth path prefix
	prefix := h.authPathPrefix
	if prefix == "" {
		prefix = "/_auth"
	}
	if prefix[0] != '/' {
		prefix = "/" + prefix
	}
	iconPath := prefix + "/assets/icons/password.svg"

	return fmt.Sprintf(`
<form id="password-form">
	<div class="form-group">
		<label class="label" for="password-input">%s</label>
		<input type="password" id="password-input" name="password" class="input" placeholder="Enter password" required />
	</div>
	<button type="submit" id="password-button" class="btn btn-primary provider-btn">
		<img src="%s" alt="Password">
		%s
	</button>
</form>

<script>
(function() {
	const form = document.getElementById('password-form');
	const button = document.getElementById('password-button');
	const input = document.getElementById('password-input');

	form.addEventListener('submit', async function(e) {
		e.preventDefault();

		const password = input.value;
		if (!password) {
			return;
		}

		button.disabled = true;
		button.textContent = 'Processing...';

		try {
			const response = await fetch(window.location.pathname.replace(/\/login$/, '') + '/password/login', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({ password: password })
			});

			if (!response.ok) {
				throw new Error('Authentication failed');
			}

			const data = await response.json();
			window.location.href = data.redirect_url || '/';
		} catch (error) {
			console.error('Password authentication error:', error);
			alert('Invalid password. Please try again.');
			button.disabled = false;
			button.innerHTML = '<img src="%s" alt="Password">%s';
			input.value = '';
			input.focus();
		}
	});
})();
</script>`, passwordLabel, iconPath, buttonText, iconPath, buttonText)
}
