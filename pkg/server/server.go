package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ideamans/multi-oauth2-proxy/pkg/auth/email"
	"github.com/ideamans/multi-oauth2-proxy/pkg/auth/oauth2"
	"github.com/ideamans/multi-oauth2-proxy/pkg/authz"
	"github.com/ideamans/multi-oauth2-proxy/pkg/config"
	"github.com/ideamans/multi-oauth2-proxy/pkg/i18n"
	"github.com/ideamans/multi-oauth2-proxy/pkg/logging"
	"github.com/ideamans/multi-oauth2-proxy/pkg/proxy"
	"github.com/ideamans/multi-oauth2-proxy/pkg/session"
)

// Server represents the HTTP server
type Server struct {
	config       *config.Config
	router       *chi.Mux
	sessionStore session.Store
	oauthManager *oauth2.Manager
	emailHandler *email.Handler
	authzChecker authz.Checker
	proxyHandler *proxy.Handler
	translator   *i18n.Translator
	logger       logging.Logger
	httpServer   *http.Server
}

// New creates a new server instance
func New(
	cfg *config.Config,
	sessionStore session.Store,
	oauthManager *oauth2.Manager,
	emailHandler *email.Handler,
	authzChecker authz.Checker,
	proxyHandler *proxy.Handler,
	logger logging.Logger,
) *Server {
	s := &Server{
		config:       cfg,
		sessionStore: sessionStore,
		oauthManager: oauthManager,
		emailHandler: emailHandler,
		authzChecker: authzChecker,
		proxyHandler: proxyHandler,
		translator:   i18n.NewTranslator(),
		logger:       logger.WithModule("server"),
	}

	s.setupRouter()
	return s
}

// setupRouter configures the HTTP router
func (s *Server) setupRouter() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Get authentication path prefix
	prefix := normalizeAuthPrefix(s.config.Server.GetAuthPathPrefix())

	// Health check endpoints (no authentication required)
	r.Get("/health", s.handleHealth)
	r.Get("/ready", s.handleReady)

	// Authentication endpoints (no authentication required)
	authRouter := chi.NewRouter()
	authRouter.Get("/login", s.handleLogin)
	authRouter.Get("/logout", s.handleLogout)
	authRouter.Post("/logout", s.handleLogout)
	authRouter.Get("/oauth2/start/{provider}", s.handleOAuth2Start)
	authRouter.Get("/oauth2/callback", s.handleOAuth2Callback)

	if s.emailHandler != nil {
		authRouter.Post("/email/send", s.handleEmailSend)
		authRouter.Get("/email/verify", s.handleEmailVerify)
	}

	// Static assets endpoint
	authRouter.Get("/assets/styles.css", s.handleStylesCSS)
	authRouter.Get("/assets/icons/{icon}", s.handleIcon)

	if prefix == "/" {
		r.Mount("/", authRouter)
	} else {
		r.Mount(prefix, authRouter)
	}

	// Protected routes - all other routes require authentication and proxy to upstream
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.HandleFunc("/*", s.handleProxy)
	})

	s.router = r
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	s.logger.Info("Starting server", "address", addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server")
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Router returns the chi router (for testing)
func (s *Server) Router() *chi.Mux {
	return s.router
}
