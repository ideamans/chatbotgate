package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Handler is a reverse proxy handler
type Handler struct {
	defaultUpstream *url.URL
	defaultProxy    *httputil.ReverseProxy
	hostProxies     map[string]*httputil.ReverseProxy
}

// NewHandler creates a new proxy handler with a default upstream
func NewHandler(upstreamURL string) (*Handler, error) {
	upstream, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(upstream)

	return &Handler{
		defaultUpstream: upstream,
		defaultProxy:    proxy,
		hostProxies:     make(map[string]*httputil.ReverseProxy),
	}, nil
}

// NewHandlerWithHosts creates a new proxy handler with host-based routing
func NewHandlerWithHosts(defaultUpstream string, hosts map[string]string) (*Handler, error) {
	// Parse default upstream
	upstream, err := url.Parse(defaultUpstream)
	if err != nil {
		return nil, fmt.Errorf("invalid default upstream URL: %w", err)
	}

	defaultProxy := httputil.NewSingleHostReverseProxy(upstream)

	// Parse host-specific upstreams
	hostProxies := make(map[string]*httputil.ReverseProxy)
	for host, upstreamURL := range hosts {
		hostUpstream, err := url.Parse(upstreamURL)
		if err != nil {
			return nil, fmt.Errorf("invalid upstream URL for host %s: %w", host, err)
		}
		hostProxies[host] = httputil.NewSingleHostReverseProxy(hostUpstream)
	}

	return &Handler{
		defaultUpstream: upstream,
		defaultProxy:    defaultProxy,
		hostProxies:     hostProxies,
	}, nil
}

// ServeHTTP handles the proxy request
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check for host-specific proxy
	if proxy, ok := h.hostProxies[r.Host]; ok {
		proxy.ServeHTTP(w, r)
		return
	}

	// Fall back to default proxy
	h.defaultProxy.ServeHTTP(w, r)
}

// AddAuthHeaders adds authentication headers to the request
func AddAuthHeaders(r *http.Request, email, provider string) {
	r.Header.Set("X-Forwarded-User", email)
	r.Header.Set("X-Forwarded-Email", email)
	r.Header.Set("X-Auth-Provider", provider)
}
