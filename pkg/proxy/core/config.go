package proxy

// UpstreamConfig represents upstream server configuration with optional secret header
type UpstreamConfig struct {
	URL    string       `yaml:"url" json:"url"`       // Upstream URL (required)
	Secret SecretConfig `yaml:"secret" json:"secret"` // Secret header configuration (optional)
}

// SecretConfig represents secret header configuration for upstream authentication
type SecretConfig struct {
	Header string `yaml:"header" json:"header"` // HTTP header name (e.g., "X-Chatbotgate-Secret")
	Value  string `yaml:"value" json:"value"`   // Secret value to send
}
