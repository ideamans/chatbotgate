# Multi OAuth2 Proxy

A flexible authentication proxy supporting multiple OAuth2 providers and email authentication. Can be used as a **standalone proxy**, **authentication middleware**, or **Go library**.

## Overview

Multi OAuth2 Proxy provides authentication as a service with three flexible deployment modes:

1. **🔧 Middleware Library** - Embed authentication in your Go application
2. **⚙️ Programmatic Server** - Configure everything in Go code
3. **📄 Configuration File Server** - Use YAML or JSON configuration files

### Key Features

- ✅ **Multiple OAuth2 Providers**: Google, GitHub, Microsoft, and custom OIDC
- ✅ **Email Authentication**: Passwordless magic links via SMTP or SendGrid
- ✅ **Flexible Storage**: Unified KVS abstraction (Memory, LevelDB, Redis) for sessions, tokens, and rate limiting
- ✅ **Flexible Architecture**: Use as library, middleware, or standalone proxy
- ✅ **Multi-tenant Support**: Host-based routing for different backends
- ✅ **Configuration Formats**: YAML, JSON, or programmatic Go configuration
- ✅ **WebSocket Support**: Full support for WebSocket and SSE connections
- ✅ **Internationalization**: Japanese and English UI
- ✅ **Security**: CSRF protection, rate limiting, HMAC-SHA256 tokens
- ✅ **Modern Design**: Tailwind CSS 4 with light/dark themes
- ✅ **Zero Dependencies at Runtime**: All assets embedded in binary

## Quick Start

### Prerequisites

- Go 1.21 or higher
- OAuth2 credentials from your provider(s):
  - [Google OAuth2](https://console.cloud.google.com/apis/credentials)
  - [GitHub OAuth Apps](https://github.com/settings/developers)
  - [Microsoft Azure AD](https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade)
- SMTP server or SendGrid API key (optional, for email authentication)

### Installation

```bash
# Clone the repository
git clone https://github.com/ideamans/multi-oauth2-proxy.git
cd multi-oauth2-proxy

# Build the binary
go build -o multi-oauth2-proxy ./cmd/multi-oauth2-proxy

# Or install directly
go install github.com/ideamans/multi-oauth2-proxy/cmd/multi-oauth2-proxy@latest
```

### Quick Start with Config File

1. **Create configuration file** (YAML or JSON):

```bash
# Copy example configuration
cp config.example.yaml config.yaml

# Or use JSON
cp config.example.json config.json
```

2. **Edit configuration**:

```yaml
service:
  name: "My Application"

server:
  host: "0.0.0.0"
  port: 4180

proxy:
  upstream: "http://localhost:8080"  # Your backend app

session:
  cookie_secret: "your-random-32-char-secret-here"  # Generate with: openssl rand -base64 32

# KVS storage (sessions, tokens, rate limiting)
kvs:
  default:
    type: "memory"  # or "leveldb" / "redis" for persistence

oauth2:
  providers:
    - name: "google"
      client_id: "YOUR-GOOGLE-CLIENT-ID"
      client_secret: "YOUR-GOOGLE-CLIENT-SECRET"
      enabled: true

authorization:
  allowed:
    - "user@example.com"
    - "@yourcompany.com"
```

3. **Run the proxy**:

```bash
./multi-oauth2-proxy -config config.yaml
# or
./multi-oauth2-proxy -config config.json
```

4. **Access your application**:
   - Navigate to `http://localhost:4180`
   - Choose authentication method (OAuth2 or Email)
   - Access your protected application

## Three Ways to Use

### 1. As a Middleware Library

Embed authentication directly in your Go application:

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
        // Configure programmatically
    }

    // Create authentication middleware
    authMiddleware := middleware.New(cfg, sessionStore, oauthManager, ...)

    // Your backend handler
    myHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := r.Header.Get("X-Forwarded-User")
        // Your application logic
    })

    // Wrap and serve
    http.ListenAndServe(":4180", authMiddleware.Wrap(myHandler))
}
```

See [`examples/middleware_only.go`](examples/middleware_only.go) for a complete example.

### 2. As a Programmatic Server

Configure the full proxy server in Go code without configuration files:

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

    srv := server.New(cfg, sessionStore, oauthManager, ...)
    srv.Start()
}
```

See [`examples/programmatic_server.go`](examples/programmatic_server.go) for a complete example.

### 3. As a Configuration File Server

Standard deployment using YAML or JSON configuration:

```bash
# With YAML
./multi-oauth2-proxy -config config.yaml

# With JSON
./multi-oauth2-proxy -config config.json
```

The format is automatically detected from the file extension (`.yaml`, `.yml`, or `.json`).

See [`examples/config_file_server.go`](examples/config_file_server.go) and [`examples/README.md`](examples/README.md) for more details.

## Architecture

### Middleware-First Design

```
┌─────────────────────────────────────────────────────┐
│                  Your Application                    │
│                                                      │
│  ┌────────────────────────────────────────────┐    │
│  │   Auth Middleware (pkg/middleware)         │    │
│  │   • OAuth2 Authentication                  │    │
│  │   • Email Authentication                   │    │
│  │   • Session Management                     │    │
│  │   • Authorization Checks                   │    │
│  └────────────────────────────────────────────┘    │
│                       ↓                              │
│  ┌────────────────────────────────────────────┐    │
│  │   Your Backend Handler / Reverse Proxy     │    │
│  │   • Receives authenticated requests        │    │
│  │   • Headers: X-Forwarded-User, etc.        │    │
│  └────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘

                        ↕
        ┌───────────────────────────────┐
        │  Unified KVS Abstraction      │
        │  ┌─────────────────────────┐  │
        │  │ session:* (Sessions)    │  │
        │  │ token:* (OTP Tokens)    │  │
        │  │ ratelimit:* (Buckets)   │  │
        │  └─────────────────────────┘  │
        │                               │
        │  Memory / LevelDB / Redis     │
        └───────────────────────────────┘
```

### Project Structure

```
multi-oauth2-proxy/
├── cmd/
│   └── multi-oauth2-proxy/     # CLI application (config file mode)
├── pkg/
│   ├── middleware/             # 🆕 Core authentication middleware
│   │   ├── middleware.go       #     Main middleware logic
│   │   ├── handlers.go         #     Auth handlers (login, OAuth2, email)
│   │   └── helpers.go          #     Helper functions
│   ├── assets/                 # 🆕 Embedded static assets
│   │   ├── assets.go           #     Go embed directives
│   │   └── static/             #     CSS, icons, HTML
│   ├── server/                 # Simplified server wrapper
│   │   ├── server.go           #     HTTP server (wraps middleware)
│   │   └── server_test.go      #     E2E tests
│   ├── auth/
│   │   ├── oauth2/             # OAuth2 providers
│   │   └── email/              # Email authentication
│   ├── authz/                  # Authorization checks
│   ├── config/                 # Configuration (YAML/JSON)
│   ├── kvs/                    # 🆕 Unified KVS abstraction
│   │   ├── kvs.go              #     Store interface
│   │   ├── memory.go           #     In-memory store
│   │   ├── leveldb.go          #     LevelDB persistent store
│   │   ├── redis.go            #     Redis distributed store
│   │   └── namespaced.go       #     Namespace wrapper
│   ├── proxy/                  # Reverse proxy with WebSocket support
│   ├── session/                # Session management (uses KVS)
│   ├── i18n/                   # Internationalization
│   ├── logging/                # Logging
│   └── ratelimit/              # Rate limiting (uses KVS)
├── examples/                   # 🆕 Usage examples
│   ├── README.md               #     Documentation
│   ├── middleware_only.go      #     Middleware library example
│   ├── programmatic_server.go  #     Programmatic config example
│   └── config_file_server.go   #     Config file example
├── web/                        # Frontend (Tailwind CSS 4)
├── config.example.yaml         # YAML configuration example
├── config.example.json         # 🆕 JSON configuration example
├── PLAN.md                     # Detailed design document
└── README.md
```

## Configuration

### Supported Formats

Both YAML and JSON are supported. The format is automatically detected from the file extension:

- `.yaml` or `.yml` → YAML format
- `.json` → JSON format

### Configuration Reference

See [`config.example.yaml`](config.example.yaml) or [`config.example.json`](config.example.json) for complete examples.

#### Service Configuration

```yaml
service:
  name: "My Application"           # Application name (shown in UI)
  description: "Authentication"    # Description (shown in UI)
  icon_url: ""                     # Optional: 48px icon URL
  logo_url: ""                     # Optional: Logo image URL
  logo_width: "200px"              # Optional: Logo width
```

#### Server Configuration

```yaml
server:
  host: "0.0.0.0"                  # Listen address
  port: 4180                       # Listen port
  auth_path_prefix: "/_auth"       # Auth endpoints prefix (default: /_auth)
```

#### Proxy Configuration

```yaml
proxy:
  upstream: "http://localhost:8080"  # Default upstream backend

  # Optional: Host-based routing for multi-tenant
  routes:
    - host: "app1.example.com"
      upstream: "http://backend1:8080"
    - host: "app2.example.com"
      upstream: "http://backend2:8080"
```

#### Session Configuration

```yaml
session:
  cookie_name: "_oauth2_proxy"     # Cookie name
  cookie_secret: "SECRET"          # Required: 32+ char secret (generate with: openssl rand -base64 32)
  cookie_expire: "168h"            # Expiration (168h = 7 days)
  cookie_secure: false             # Set true for HTTPS
  cookie_httponly: true            # HttpOnly flag
  cookie_samesite: "lax"           # SameSite policy
```

#### KVS (Key-Value Store) Configuration

Multi OAuth2 Proxy uses a unified KVS abstraction for sessions, OTP tokens, and rate limiting. You can use a single shared backend or dedicated backends for each purpose:

```yaml
kvs:
  # Default KVS (shared by all use cases with namespace isolation)
  default:
    type: "memory"  # "memory", "leveldb", or "redis"

    # Memory-specific config
    memory:
      cleanup_interval: "5m"

    # LevelDB-specific config (persistent, single-server)
    # leveldb:
    #   path: "/var/lib/multi-oauth2-proxy/kvs"  # Empty = OS cache/temp dir
    #   sync_writes: false
    #   cleanup_interval: "5m"

    # Redis-specific config (distributed, multi-server)
    # redis:
    #   addr: "localhost:6379"
    #   password: ""
    #   db: 0
    #   pool_size: 0  # 0 = default (10 * CPU cores)

  # Namespace prefixes (optional, defaults shown)
  namespaces:
    session: "session:"      # Session keys
    token: "token:"          # OTP token keys
    ratelimit: "ratelimit:"  # Rate limit bucket keys

  # Optional: Override with dedicated backends
  # session:
  #   type: "redis"
  #   redis:
  #     addr: "localhost:6379"
  #     db: 1

  # token:
  #   type: "memory"
  #   memory:
  #     cleanup_interval: "1m"

  # ratelimit:
  #   type: "leveldb"
  #   leveldb:
  #     path: "/var/lib/multi-oauth2-proxy/ratelimit"
```

**KVS Design Benefits:**

- 🔄 **Single Backend**: One Redis/LevelDB connection serves all purposes
- 🏷️ **Namespace Isolation**: Logical separation with key prefixes
- ⚡ **Efficient Cleanup**: Each namespace scans only its own keys
- 🎯 **Flexibility**: Override specific use cases with dedicated backends
- 📊 **Scalability**: Shared backend or independent scaling per use case

#### OAuth2 Providers

```yaml
oauth2:
  providers:
    - name: "google"               # Provider ID
      display_name: "Google"       # Display name in UI
      client_id: "YOUR-CLIENT-ID"
      client_secret: "YOUR-SECRET"
      enabled: true

    - name: "github"
      display_name: "GitHub"
      client_id: "YOUR-CLIENT-ID"
      client_secret: "YOUR-SECRET"
      enabled: true

    - name: "microsoft"
      display_name: "Microsoft"
      client_id: "YOUR-CLIENT-ID"
      client_secret: "YOUR-SECRET"
      enabled: false

    # Custom OIDC provider
    - name: "custom"
      display_name: "Custom OIDC"
      client_id: "YOUR-CLIENT-ID"
      client_secret: "YOUR-SECRET"
      auth_url: "https://provider.example.com/oauth2/authorize"
      token_url: "https://provider.example.com/oauth2/token"
      userinfo_url: "https://provider.example.com/oauth2/userinfo"
      enabled: false
```

#### Email Authentication

```yaml
email_auth:
  enabled: true
  sender_type: "smtp"              # "smtp", "sendgrid", or "file"

  # SMTP configuration
  smtp:
    host: "smtp.gmail.com"
    port: 587
    username: "your-email@gmail.com"
    password: "your-app-password"
    tls: false
    starttls: true

  # SendGrid configuration (alternative)
  sendgrid:
    api_key: "SG.xxxxxxxxxx"

  from_email: "noreply@example.com"
  from_name: "My Application"

  token:
    expire: "15m"                  # Token expiration
```

#### Authorization

```yaml
authorization:
  # Specific email addresses
  allowed_emails:
    - "user@example.com"
    - "admin@company.com"

  # Domain wildcards
  allowed_domains:
    - "@yourcompany.com"
    - "@trusted-partner.org"
```

#### Logging

```yaml
logging:
  level: "info"                    # debug, info, warn, error
  module_level: "debug"            # Sub-module log level
  format: "text"                   # text or json
  color: true                      # Colored output (auto-detects TTY)
```

## Authentication Flow

All three usage modes follow the same authentication flow:

```
1. User accesses protected resource
   ↓
2. Middleware checks session
   ↓
3. If not authenticated → Redirect to /_auth/login
   ↓
4. User selects authentication method:
   • OAuth2 provider (Google, GitHub, Microsoft)
   • Email magic link
   ↓
5. Authentication succeeds
   ↓
6. Session created, redirect to original resource
   ↓
7. All subsequent requests include headers:
   • X-Forwarded-User: user@example.com
   • X-Forwarded-Email: user@example.com
   • X-Auth-Provider: google|github|microsoft|email
```

## OAuth2 Provider Setup

### Google OAuth2

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create/select project → APIs & Services → Credentials
3. Create OAuth 2.0 Client ID
4. Add authorized redirect URIs:
   - `http://localhost:4180/_auth/oauth2/callback` (development)
   - `https://yourdomain.com/_auth/oauth2/callback` (production)
5. Copy Client ID and Secret to config

### GitHub OAuth Apps

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. New OAuth App
3. Authorization callback URL: `http://localhost:4180/_auth/oauth2/callback`
4. Copy Client ID and Secret to config

### Microsoft Azure AD

1. Go to [Azure Portal](https://portal.azure.com/)
2. Azure AD → App registrations → New registration
3. Redirect URI: `http://localhost:4180/_auth/oauth2/callback`
4. Certificates & secrets → New client secret
5. API permissions → Add `User.Read`, `email`, `profile`, `openid`
6. Copy Application ID and Secret to config

## Advanced Features

### Host-Based Routing (Multi-Tenant)

Route different hosts to different backends:

```yaml
proxy:
  upstream: "http://default-backend:8080"
  routes:
    - host: "app1.example.com"
      upstream: "http://backend1:8080"
    - host: "app2.example.com"
      upstream: "http://backend2:8080"
```

### WebSocket Support

WebSocket connections are automatically detected and proxied correctly. The proxy preserves:
- `Upgrade: websocket` headers
- `Connection: Upgrade` headers
- All WebSocket frames

### Server-Sent Events (SSE)

SSE streaming is supported with `FlushInterval: 100ms` for real-time updates.

### Custom Auth Path Prefix

Change the authentication endpoint prefix:

```yaml
server:
  auth_path_prefix: "/_oauth2_proxy"  # Default: /_auth
```

Authentication endpoints become:
- `/_oauth2_proxy/login`
- `/_oauth2_proxy/logout`
- `/_oauth2_proxy/oauth2/start/{provider}`
- etc.

## API Endpoints

### Authentication Endpoints

All authentication endpoints are prefixed with `auth_path_prefix` (default: `/_auth`):

- `GET /_auth/login` - Login page (displays all providers)
- `GET /_auth/logout` - Logout and clear session
- `POST /_auth/logout` - Logout via POST
- `GET /_auth/oauth2/start/{provider}` - Start OAuth2 flow
- `GET /_auth/oauth2/callback` - OAuth2 callback handler
- `POST /_auth/email/send` - Send magic link email
- `GET /_auth/email/verify` - Verify email token
- `GET /_auth/assets/styles.css` - Embedded CSS
- `GET /_auth/assets/icons/{icon}` - Embedded icons

### Health Check Endpoints

- `GET /health` - Health check (always returns 200 OK)
- `GET /ready` - Readiness check (always returns 200 OK)

### Protected Routes

- `/*` - All other routes (require authentication, proxied to backend)

## Development

### Building

```bash
# Build for current platform
go build -o multi-oauth2-proxy ./cmd/multi-oauth2-proxy

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o multi-oauth2-proxy ./cmd/multi-oauth2-proxy

# Build with version
go build -ldflags "-X main.version=1.0.0" -o multi-oauth2-proxy ./cmd/multi-oauth2-proxy
```

### Testing

#### Unit Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./pkg/middleware/... -v
```

#### E2E Tests

The project includes a comprehensive E2E test environment with Docker Compose. See [E2E.md](./E2E.md) for details.

```bash
# Start E2E environment (includes Mailpit for email preview)
docker compose -f e2e/docker/docker-compose.e2e.yaml up --build

# Access the test environment:
# - Proxy: http://localhost:4180
# - Mailpit WebUI: http://localhost:8025 (email preview)
# - Target App: http://localhost:3000
# - OAuth2 Stub: http://localhost:3001

# Run Playwright tests
docker compose -f e2e/docker/docker-compose.e2e.yaml \
               -f e2e/docker/docker-compose.e2e.playwright.yaml \
               up --build playwright-runner
```

**Mailpit Email Preview**: The E2E environment includes [Mailpit](https://github.com/axllent/mailpit), a lightweight email server that allows you to:
- Preview emails in a web UI (http://localhost:8025)
- Click login links directly from emails
- Test email authentication flows end-to-end
- Use REST API to verify email content in tests

### Design System

The project includes a modern design system built with Tailwind CSS 4:

```bash
# Install web dependencies
cd web && npm install

# Start design system catalog
npm run dev

# Build CSS
npm run build
```

Visit `http://localhost:3000` to see the design system catalog.

## Security Considerations

1. **Cookie Secret**: Use strong random secret (32+ chars): `openssl rand -base64 32`
2. **HTTPS**: Always use `cookie_secure: true` in production
3. **Session Expiration**: Configure appropriate timeouts (default: 7 days)
4. **Authorization**: Use domain restrictions (`@company.com`) when possible
5. **Rate Limiting**: Email auth includes rate limiting (3 req/min default)
6. **CSRF Protection**: OAuth2 flows use state parameter for CSRF
7. **Token Security**: Magic links use HMAC-SHA256 with 15min expiration
8. **Credentials**: Never commit secrets to version control
9. **OAuth2 Secrets**: Rotate client secrets periodically
10. **Headers**: Authentication headers are passed securely to backend

## Development Phases

- **Phase 1** ✅ (Complete): Core OAuth2 + Reverse Proxy - 56.2% coverage
- **Phase 2** ✅ (Complete): Email auth + Security - 63.1% coverage
- **Phase 3** ✅ (Complete): Multi-provider + i18n - 69.9% coverage
- **Phase 4** ✅ (Complete): Middleware architecture + JSON config
- **Phase 5** ✅ (Complete): Unified KVS abstraction (Memory/LevelDB/Redis)
- **Phase 6** (Planned): Prometheus metrics, structured logging
- **Phase 7** (Planned): SSL/TLS automation, MFA, WebAuthn

See [PLAN.md](PLAN.md) for detailed design documentation.

## Comparison with oauth2-proxy

| Feature | multi-oauth2-proxy | oauth2-proxy |
|---------|-------------------|--------------|
| Multiple OAuth2 providers | ✅ | ✅ |
| Email authentication | ✅ | ❌ |
| Middleware library | ✅ | ❌ |
| Programmatic config | ✅ | ❌ |
| JSON config support | ✅ | ❌ |
| WebSocket support | ✅ | ✅ |
| Host-based routing | ✅ | Limited |
| Modern UI (Tailwind) | ✅ | Basic |
| Internationalization | ✅ (ja/en) | ❌ |
| Embedded assets | ✅ | ❌ |

## Contributing

Contributions are welcome! Please:

1. Check existing issues or create a new one
2. Fork the repository
3. Create a feature branch
4. Write tests for your changes
5. Ensure all tests pass: `go test ./...`
6. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [oauth2-proxy](https://oauth2-proxy.github.io/oauth2-proxy/)
- Built with [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2)
- Design system powered by [Tailwind CSS 4](https://tailwindcss.com/)

## Support

- **Issues**: [GitHub Issues](https://github.com/ideamans/multi-oauth2-proxy/issues)
- **Documentation**: [PLAN.md](PLAN.md) for detailed design
- **Examples**: [examples/](examples/) directory
