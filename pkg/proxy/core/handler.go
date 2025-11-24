package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Handler is a reverse proxy handler
type Handler struct {
	upstream *url.URL
	proxy    *httputil.ReverseProxy
	secret   SecretConfig
}

// NewHandler creates a new proxy handler with a default upstream
// Deprecated: Use NewHandlerWithConfig instead
func NewHandler(upstreamURL string) (*Handler, error) {
	upstreamConfig := UpstreamConfig{
		URL: upstreamURL,
	}
	return NewHandlerWithConfig(upstreamConfig)
}

// NewHandlerWithConfig creates a new proxy handler with upstream configuration
func NewHandlerWithConfig(upstreamConfig UpstreamConfig) (*Handler, error) {
	// Parse upstream
	upstream, err := url.Parse(upstreamConfig.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL: %w", err)
	}

	proxy := createReverseProxy(upstream, upstreamConfig.Secret)

	return &Handler{
		upstream: upstream,
		proxy:    proxy,
		secret:   upstreamConfig.Secret,
	}, nil
}

// createReverseProxy creates a reverse proxy with WebSocket, SSE, and streaming support
func createReverseProxy(target *url.URL, secret SecretConfig) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Preserve the original Director
	originalDirector := proxy.Director

	// Custom Director to handle headers and protocol upgrades
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Add secret header if configured
		if secret.Header != "" && secret.Value != "" {
			req.Header.Set(secret.Header, secret.Value)
		}

		// Add X-Forwarded-* headers for backend to know original request details
		if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
			// X-Real-IP: Original client IP
			if prior := req.Header.Get("X-Real-IP"); prior == "" {
				req.Header.Set("X-Real-IP", clientIP)
			}

			// X-Forwarded-For: Chain of proxies
			if prior := req.Header.Get("X-Forwarded-For"); prior != "" {
				clientIP = prior + ", " + clientIP
			}
			req.Header.Set("X-Forwarded-For", clientIP)
		}

		// X-Forwarded-Proto: Original protocol (http/https)
		if req.Header.Get("X-Forwarded-Proto") == "" {
			proto := "http"
			if req.TLS != nil {
				proto = "https"
			}
			req.Header.Set("X-Forwarded-Proto", proto)
		}

		// X-Forwarded-Host: Original host header
		if req.Header.Get("X-Forwarded-Host") == "" {
			req.Header.Set("X-Forwarded-Host", req.Host)
		}

		// Preserve WebSocket upgrade headers
		if strings.ToLower(req.Header.Get("Upgrade")) == "websocket" {
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Upgrade", "websocket")
		}
	}

	// Enable streaming responses (SSE, video streaming, large downloads)
	// FlushInterval causes the ReverseProxy to flush to the client
	// while copying the response body. This enables Server-Sent Events (SSE)
	// and streaming responses to work properly.
	proxy.FlushInterval = 100 * time.Millisecond

	// BufferPool reduces memory allocations for large file transfers
	// by reusing byte slices between requests
	proxy.BufferPool = newBufferPool()

	return proxy
}

// bufferPool implements httputil.BufferPool for memory-efficient copying
// Uses sync.Pool to reuse buffers and reduce GC pressure
type bufferPool struct {
	pool *sync.Pool
}

// newBufferPool creates a new buffer pool with 32KB buffers
func newBufferPool() *bufferPool {
	return &bufferPool{
		pool: &sync.Pool{
			New: func() interface{} {
				// Allocate 32KB buffer for efficient file copying
				b := make([]byte, 32*1024)
				return &b
			},
		},
	}
}

func (bp *bufferPool) Get() []byte {
	// Get a buffer from the pool
	bufPtr := bp.pool.Get().(*[]byte)
	return *bufPtr
}

func (bp *bufferPool) Put(b []byte) {
	// Only pool buffers of expected size to prevent memory bloat
	if cap(b) != 32*1024 {
		return
	}
	// Reset slice to full capacity before returning to pool
	b = b[:cap(b)]
	bp.pool.Put(&b)
}

// ServeHTTP handles the proxy request
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.proxy.ServeHTTP(w, r)
}

// AddAuthHeaders adds authentication headers to the request
// Deprecated: Use the Forwarder from pkg/forwarding for more flexible header management
func AddAuthHeaders(r *http.Request, email, provider string) {
	r.Header.Set("X-ChatbotGate-User", email)
	r.Header.Set("X-ChatbotGate-Email", email)
	r.Header.Set("X-Auth-Provider", provider)
}
