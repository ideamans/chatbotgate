package email

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ideamans/chatbotgate/pkg/middleware/authz"
	"github.com/ideamans/chatbotgate/pkg/middleware/config"
	"github.com/ideamans/chatbotgate/pkg/middleware/ratelimit"
	"github.com/ideamans/chatbotgate/pkg/shared/i18n"
	"github.com/ideamans/chatbotgate/pkg/shared/kvs"
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
	tokenKVS kvs.Store,
	emailQuotaKVS kvs.Store,
) (*Handler, error) {
	serviceName := serviceCfg.Name
	// Create token store with KVS backend
	tokenStore := NewTokenStore(cookieSecret, tokenKVS)

	// Parse EmailAuthConfig.From for shared sender config
	parentEmail, parentName := cfg.GetFromAddress()

	// Create sender based on configuration
	var sender Sender
	switch cfg.SenderType {
	case "smtp":
		sender = NewSMTPSender(cfg.SMTP, parentEmail, parentName)
	case "sendgrid":
		sender = NewSendGridSender(cfg.SendGrid, parentEmail, parentName)
	case "sendmail":
		sender = NewSendmailSender(cfg.Sendmail, parentEmail, parentName)
	default:
		return nil, fmt.Errorf("unsupported sender type: %s", cfg.SenderType)
	}

	// Create rate limiter with KVS backend using configured limit per minute
	limiter := ratelimit.NewLimiter(cfg.GetLimitPerMinute(), 1*time.Minute, emailQuotaKVS)

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

	// Get OTP from token for email display
	ctx := context.Background()
	tokenData, err := h.tokenStore.kvs.Get(ctx, token)
	if err != nil {
		h.tokenStore.DeleteToken(token)
		return fmt.Errorf("failed to retrieve token data: %w", err)
	}

	var tokenObj Token
	if err := json.Unmarshal(tokenData, &tokenObj); err != nil {
		h.tokenStore.DeleteToken(token)
		return fmt.Errorf("failed to unmarshal token: %w", err)
	}

	// Generate HTML email using Hermes template with OTP
	htmlBody, textBody, err := h.emailTemplate.GenerateLoginEmail(loginURL, tokenObj.OTP, int(duration.Minutes()), lang, h.translator)
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

// VerifyOTP verifies an OTP and returns the associated email
func (h *Handler) VerifyOTP(otp string) (string, error) {
	return h.tokenStore.VerifyOTP(otp)
}

// Cleanup removes expired tokens
func (h *Handler) Cleanup() {
	h.tokenStore.CleanupExpired()
}

// SetSender sets the email sender (for testing)
func (h *Handler) SetSender(sender Sender) {
	h.sender = sender
}
