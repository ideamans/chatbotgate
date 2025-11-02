package email

import (
	"fmt"
	"time"

	"github.com/ideamans/multi-oauth2-proxy/pkg/authz"
	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
	"github.com/ideamans/multi-oauth2-proxy/pkg/i18n"
	"github.com/ideamans/multi-oauth2-proxy/pkg/ratelimit"
)

// Handler manages email authentication
type Handler struct {
	tokenStore     *TokenStore
	sender         Sender
	authzChecker   authz.Checker
	limiter        *ratelimit.Limiter
	emailTemplate  *EmailTemplate
	translator     *i18n.Translator
	config         config.EmailAuthConfig
	serviceName    string
	baseURL        string
	authPathPrefix string
}

// NewHandler creates a new email authentication handler
func NewHandler(
	cfg config.EmailAuthConfig,
	serviceCfg config.ServiceConfig,
	baseURL string,
	authPathPrefix string,
	authzChecker authz.Checker,
	translator *i18n.Translator,
	cookieSecret string,
) (*Handler, error) {
	serviceName := serviceCfg.Name
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

	// Create email template
	logoWidth := serviceCfg.LogoWidth
	if logoWidth == "" {
		logoWidth = "200px" // Default logo width
	}
	emailTemplate := NewEmailTemplate(
		serviceName,
		serviceCfg.LogoURL,
		logoWidth,
		serviceCfg.IconURL,
		baseURL,
	)

	return &Handler{
		tokenStore:     tokenStore,
		sender:         sender,
		authzChecker:   authzChecker,
		limiter:        limiter,
		emailTemplate:  emailTemplate,
		translator:     translator,
		config:         cfg,
		serviceName:    serviceName,
		baseURL:        baseURL,
		authPathPrefix: authPathPrefix,
	}, nil
}

// SendLoginLink sends a login link to the specified email address
func (h *Handler) SendLoginLink(email string, lang i18n.Language) error {
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

	// Generate HTML email using Hermes template
	htmlBody, textBody, err := h.emailTemplate.GenerateLoginEmail(loginURL, int(duration.Minutes()), lang, h.translator)
	if err != nil {
		// Clean up token if generation fails
		h.tokenStore.DeleteToken(token)
		return fmt.Errorf("failed to generate email: %w", err)
	}

	// Send HTML email
	subject := fmt.Sprintf(h.translator.T(lang, "email.login.subject"), h.serviceName)
	if err := h.sender.SendHTML(email, subject, htmlBody, textBody); err != nil {
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
