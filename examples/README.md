# Multi OAuth2 Proxy - Usage Examples

This directory contains examples showing three different ways to use multi-oauth2-proxy.

## Overview

multi-oauth2-proxy can be used in three different ways:

### 1. Middleware Library (Programming Interface)
Use only the authentication middleware in your Go application. You provide your own backend handler.

**Use case:** You have an existing Go web application and want to add OAuth2 authentication.

**Example:** `middleware_only.go`

```bash
go run middleware_only.go
```

### 2. Programmatic Server (Programming Interface)
Configure and run the full proxy server entirely from Go code, without configuration files.

**Use case:** You want full control over configuration in Go code, or you want to generate configuration dynamically.

**Example:** `programmatic_server.go`

```bash
# Set environment variables for OAuth2 providers
export GOOGLE_CLIENT_ID="your-client-id"
export GOOGLE_CLIENT_SECRET="your-client-secret"

go run programmatic_server.go
```

### 3. Configuration File Server (CLI Interface)
Load configuration from YAML or JSON files and start the proxy server.

**Use case:** Standard deployment with configuration files.

**Example:** `config_file_server.go`

```bash
# With YAML configuration
go run config_file_server.go -config ../config.example.yaml

# With JSON configuration
go run config_file_server.go -config ../config.example.json
```

## Configuration Formats

### YAML Configuration
See `../config.example.yaml` for a full example.

```yaml
service:
  name: "My OAuth2 Proxy"
  description: "Authentication proxy"

server:
  host: "0.0.0.0"
  port: 4180

proxy:
  upstream: "http://localhost:8080"

session:
  cookie_name: "_oauth2_proxy"
  cookie_secret: "your-random-secret-key-32-chars"
  cookie_expire: "168h"

oauth2:
  providers:
    - name: "google"
      client_id: "YOUR-CLIENT-ID"
      client_secret: "YOUR-CLIENT-SECRET"
      enabled: true
```

### JSON Configuration
See `../config.example.json` for a full example.

```json
{
  "service": {
    "name": "My OAuth2 Proxy",
    "description": "Authentication proxy"
  },
  "server": {
    "host": "0.0.0.0",
    "port": 4180
  },
  "proxy": {
    "upstream": "http://localhost:8080"
  },
  "session": {
    "cookie_name": "_oauth2_proxy",
    "cookie_secret": "your-random-secret-key-32-chars",
    "cookie_expire": "168h"
  },
  "oauth2": {
    "providers": [
      {
        "name": "google",
        "client_id": "YOUR-CLIENT-ID",
        "client_secret": "YOUR-CLIENT-SECRET",
        "enabled": true
      }
    ]
  }
}
```

## Comparison of Usage Modes

| Feature | Middleware Only | Programmatic Server | Config File Server |
|---------|----------------|---------------------|-------------------|
| **Configuration** | Go code | Go code | YAML/JSON file |
| **Reverse Proxy** | You provide | Built-in | Built-in |
| **OAuth2 Auth** | ✓ | ✓ | ✓ |
| **Email Auth** | ✓ | ✓ | ✓ |
| **Flexibility** | Highest | High | Medium |
| **Ease of Use** | Medium | Medium | Easiest |
| **Best For** | Embedding in apps | Dynamic config | Standard deployment |

## Quick Start Guide

### 1. Middleware Only

```go
package main

import (
    "net/http"
    "github.com/ideamans/multi-oauth2-proxy/pkg/middleware"
    "github.com/ideamans/multi-oauth2-proxy/pkg/config"
    // ... other imports
)

func main() {
    cfg := &config.Config{
        // Configure here
    }

    // Create middleware
    authMiddleware := middleware.New(cfg, ...)

    // Your backend handler
    myHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := r.Header.Get("X-Forwarded-User")
        // Your application logic
    })

    // Wrap and serve
    http.ListenAndServe(":4180", authMiddleware.Wrap(myHandler))
}
```

### 2. Programmatic Server

```go
package main

import (
    "github.com/ideamans/multi-oauth2-proxy/pkg/server"
    "github.com/ideamans/multi-oauth2-proxy/pkg/config"
    // ... other imports
)

func main() {
    cfg := &config.Config{
        Server: config.ServerConfig{
            Host: "0.0.0.0",
            Port: 4180,
        },
        Proxy: config.ProxyConfig{
            Upstream: "http://localhost:8080",
        },
        // ... full configuration
    }

    srv := server.New(cfg, ...)
    srv.Start()
}
```

### 3. Configuration File Server

```bash
# Create config.yaml
cat > config.yaml << EOF
service:
  name: "My OAuth2 Proxy"
server:
  host: "0.0.0.0"
  port: 4180
proxy:
  upstream: "http://localhost:8080"
# ... rest of configuration
EOF

# Run the proxy
./multi-oauth2-proxy -config config.yaml
```

## Authentication Flow

All three usage modes follow the same authentication flow:

1. User accesses a protected resource
2. Redirected to `/_auth/login` (or custom prefix)
3. User selects authentication method (OAuth2 or Email)
4. After successful authentication, redirected back to original resource
5. All subsequent requests include authentication headers:
   - `X-Forwarded-User`: User's email
   - `X-Forwarded-Email`: User's email
   - `X-Auth-Provider`: Authentication provider (google, github, email, etc.)

## Next Steps

- See the main [README.md](../README.md) for detailed configuration options
- Check [PLAN.md](../PLAN.md) for architecture details
- Run the examples to see each mode in action
- Customize the configuration for your use case

## Support

For questions or issues:
- GitHub Issues: https://github.com/ideamans/multi-oauth2-proxy/issues
- Documentation: See README.md and PLAN.md
