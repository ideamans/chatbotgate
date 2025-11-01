package email

import (
	"fmt"
	"time"

	"github.com/ideamans/multi-oauth2-proxy/pkg/authz"
	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
	"github.com/ideamans/multi-oauth2-proxy/pkg/ratelimit"
)

// Handler manages email authentication
type Handler struct {
	tokenStore     *TokenStore
	sender         Sender
	authzChecker   authz.Checker
	limiter        *ratelimit.Limiter
	fileWriter     *FileWriter
	config         config.EmailAuthConfig
	serviceName    string
	baseURL        string
	authPathPrefix string
}

// NewHandler creates a new email authentication handler
func NewHandler(
	cfg config.EmailAuthConfig,
	serviceName string,
	baseURL string,
	authPathPrefix string,
	authzChecker authz.Checker,
	cookieSecret string,
) (*Handler, error) {
	// Create token store
	tokenStore := NewTokenStore(cookieSecret)

	// Create sender based on configuration
	var sender Sender
	switch cfg.SenderType {
	case "smtp":
		sender = NewSMTPSender(cfg.SMTP)
	case "sendgrid":
		sender = NewSendGridSender(cfg.SendGrid)
	default:
		return nil, fmt.Errorf("unsupported sender type: %s", cfg.SenderType)
	}

	// Create rate limiter (3 emails per minute per address)
	limiter := ratelimit.NewLimiter(3, 1*time.Minute)

	// Create file writer if OTP output file is configured
	var fileWriter *FileWriter
	if cfg.OTPOutputFile != "" {
		fileWriter = NewFileWriter(cfg.OTPOutputFile)
	}

	return &Handler{
		tokenStore:     tokenStore,
		sender:         sender,
		authzChecker:   authzChecker,
		limiter:        limiter,
		fileWriter:     fileWriter,
		config:         cfg,
		serviceName:    serviceName,
		baseURL:        baseURL,
		authPathPrefix: authPathPrefix,
	}, nil
}

// SendLoginLink sends a login link to the specified email address
func (h *Handler) SendLoginLink(email string) error {
	// Check authorization first
	if !h.authzChecker.IsAllowed(email) {
		return fmt.Errorf("email not authorized: %s", email)
	}

	// Check rate limit
	if !h.limiter.Allow(email) {
		return fmt.Errorf("rate limit exceeded for: %s", email)
	}

	// Get token duration
	duration, err := h.config.Token.GetTokenExpireDuration()
	if err != nil {
		duration = 15 * time.Minute // Default
	}

	// Generate token
	token, err := h.tokenStore.GenerateToken(email, duration)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Create login URL
	loginURL := fmt.Sprintf("%s%s/email/verify?token=%s", h.baseURL, h.authPathPrefix, token)

	// If OTP file output is configured, write to file instead of sending email
	if h.fileWriter != nil {
		expiresAt := time.Now().Add(duration)
		if err := h.fileWriter.WriteOTP(email, token, loginURL, expiresAt); err != nil {
			// Clean up token if write fails
			h.tokenStore.DeleteToken(token)
			return fmt.Errorf("failed to write OTP to file: %w", err)
		}
		return nil
	}

	// Compose email
	subject := fmt.Sprintf("Login Link - %s", h.serviceName)
	body := fmt.Sprintf(`Click the link below to log in to %s.
This link is valid for %d minutes.

%s

If you did not request this email, please ignore it.`,
		h.serviceName,
		int(duration.Minutes()),
		loginURL)

	// Send email
	if err := h.sender.Send(email, subject, body); err != nil {
		// Clean up token if send fails
		h.tokenStore.DeleteToken(token)
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// VerifyToken verifies a login token and returns the associated email
func (h *Handler) VerifyToken(token string) (string, error) {
	return h.tokenStore.VerifyToken(token)
}

// Cleanup removes expired tokens
func (h *Handler) Cleanup() {
	h.tokenStore.CleanupExpired()
}

// SetSender sets the email sender (for testing)
func (h *Handler) SetSender(sender Sender) {
	h.sender = sender
}
