package oauth2

import "net/http"

// testTransport is a custom HTTP transport for testing
type testTransport struct {
	baseURL string
	path    string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect all requests to test server
	req.URL.Scheme = "http"
	req.URL.Host = req.URL.Host // Keep the host from server.URL
	if t.baseURL != "" {
		// Parse baseURL to get host
		req.URL, _ = req.URL.Parse(t.baseURL + t.path)
	}
	return http.DefaultTransport.RoundTrip(req)
}
