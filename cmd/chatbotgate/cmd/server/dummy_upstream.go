package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// DummyUpstream is a simple HTTP server for testing purposes
// when no upstream is configured
type DummyUpstream struct {
	server   *http.Server
	listener net.Listener
	logger   logging.Logger
}

// NewDummyUpstream creates and starts a new dummy upstream server
// on an automatically selected available port.
// Returns the server instance or nil if startup fails (with a warning logged).
func NewDummyUpstream(logger logging.Logger) *DummyUpstream {
	if logger == nil {
		logger = logging.NewSimpleLogger("dummy-upstream", logging.LevelInfo, true)
	}

	// Find an available port by listening on :0
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		logger.Warn("Failed to start dummy upstream server", "error", err)
		return nil
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<title>Dummy Upstream</title>
<style>
body { font-family: sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
h1 { color: #333; }
p { color: #666; }
code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
</style>
</head>
<body>
<h1>Dummy Upstream Server</h1>
<p>This is a placeholder response from the ChatbotGate dummy upstream server.</p>
<p>Request path: <code>%s</code></p>
<p>Configure a real upstream in your config file to replace this.</p>
</body>
</html>`, r.URL.Path)
	})

	server := &http.Server{
		Handler: handler,
	}

	d := &DummyUpstream{
		server:   server,
		listener: listener,
		logger:   logger,
	}

	// Start server in background
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Warn("Dummy upstream server error", "error", err)
		}
	}()

	logger.Info("Dummy upstream server started", "addr", d.URL())

	return d
}

// URL returns the URL of the dummy upstream server
func (d *DummyUpstream) URL() string {
	return fmt.Sprintf("http://%s", d.listener.Addr().String())
}

// Stop stops the dummy upstream server with a timeout
// This method is non-blocking and will force close after timeout
func (d *DummyUpstream) Stop() {
	if d == nil || d.server == nil {
		return
	}

	d.logger.Info("Stopping dummy upstream server")

	// Use a short timeout for shutdown (not graceful)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try to shutdown, but don't wait too long
	done := make(chan struct{})
	go func() {
		_ = d.server.Shutdown(ctx)
		close(done)
	}()

	select {
	case <-done:
		d.logger.Info("Dummy upstream server stopped")
	case <-time.After(3 * time.Second):
		// Force close if shutdown takes too long
		d.logger.Warn("Force closing dummy upstream server (shutdown timeout)")
		_ = d.listener.Close()
	}
}
