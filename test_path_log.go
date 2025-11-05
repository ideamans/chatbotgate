package main

import (
	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

func main() {
	logger := logging.NewSimpleLogger("main", logging.LevelDebug, true)
	
	// Without path
	logger.Info("Server started", "port", 4180, "host", "localhost")
	
	// With path
	logger.Info("Authentication successful", "path", "/_auth/login", "email", "user@example.com", "provider", "google")
	
	// Path only
	logger.Debug("Handling request", "path", "/api/users")
	
	// Nested component with path
	middlewareLogger := logger.WithModule("middleware")
	middlewareLogger.Info("Session created", "path", "/_auth/oauth2/callback", "session_id", "abc123")
	
	// Error with path
	middlewareLogger.Error("Authentication failed", "path", "/_auth/login", "error", "invalid credentials")
}
