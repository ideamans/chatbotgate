# multi-oauth2-proxy

A lightweight authentication proxy supporting multiple OAuth2 providers with email-based authorization.

## Features

**Phase 1 (✅ Complete)**
- ✅ Single OAuth2 provider (Google)
- ✅ Reverse proxy to backend application
- ✅ Email and domain-based authorization
- ✅ Session management (in-memory)
- ✅ Simple web UI for authentication
- ✅ YAML configuration
- ✅ Test coverage 56.2%

**Phase 2 (✅ Complete)**
- ✅ Email authentication (passwordless magic links)
- ✅ SMTP and SendGrid support
- ✅ One-time token management (HMAC-SHA256)
- ✅ Rate limiting (token bucket algorithm)
- ✅ CSRF protection with state parameter
- ✅ Test coverage 63.1%

**Phase 3 (✅ Complete - Current)**
- ✅ Multiple OAuth2 providers (Google, GitHub, Microsoft)
- ✅ Host-based routing (multi-tenant support)
- ✅ Configuration auto-reload (fsnotify)
- ✅ Internationalization (Japanese/English)
- ✅ Language detection (Accept-Language, Cookie, Query)
- ✅ Colored logging with TTY detection
- ✅ Test coverage 69.9%

**Upcoming Phases**
- Phase 4: Redis sessions, Prometheus metrics, structured logging
- Phase 5: SSL/TLS automation (Let's Encrypt), MFA, WebAuthn

## Quick Start

### Prerequisites

- Go 1.21 or higher
- OAuth2 credentials for your chosen provider(s):
  - [Google OAuth2](https://console.cloud.google.com/apis/credentials)
  - [GitHub OAuth Apps](https://github.com/settings/developers)
  - [Microsoft Azure AD](https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade)
- SMTP server or SendGrid API key (optional, for email authentication)

### Installation

```bash
# Clone the repository
git clone https://github.com/ideamans/multi-oauth2-proxy.git
cd multi-oauth2-proxy

# Build
go build -o multi-oauth2-proxy ./cmd/multi-oauth2-proxy

# Or install
go install ./cmd/multi-oauth2-proxy
```

### Configuration

1. Copy the example configuration:

```bash
cp config.example.yaml config.yaml
```

2. Edit `config.yaml` and configure:
   - **cookie_secret**: Generate with `openssl rand -base64 32`
   - **OAuth2 credentials**: Add client ID and secret for your chosen provider(s)
   - **upstream**: Set your backend application URL
   - **authorization**: Configure allowed emails/domains

Example configuration:

```yaml
service:
  name: "My Application"
  description: "Secure authentication for My Application"

server:
  host: "0.0.0.0"
  port: 4180

proxy:
  upstream: "http://localhost:8080"

session:
  cookie_secret: "your-random-secret-here"
  cookie_expire: "168h"

oauth2:
  providers:
    - name: "google"
      client_id: "your-google-client-id"
      client_secret: "your-google-client-secret"
      enabled: true

authorization:
  allowed_emails:
    - "user@example.com"
  allowed_domains:
    - "@yourcompany.com"
```

### Running

```bash
# Start the proxy
./multi-oauth2-proxy -config config.yaml

# Or with default config path
./multi-oauth2-proxy
```

The proxy will start on `http://localhost:4180` by default.

### Usage

1. Navigate to `http://localhost:4180`
2. You'll be redirected to the login page
3. Choose your authentication method:
   - Click on your preferred OAuth2 provider (Google, GitHub, Microsoft)
   - Or use email authentication (if enabled)
4. Authorize the application
5. If your email is authorized, you'll be redirected to your backend application
6. All subsequent requests will be proxied with authentication headers:
   - `X-Forwarded-User`: User's email
   - `X-Forwarded-Email`: User's email
   - `X-Auth-Provider`: Authentication provider (e.g., "google", "github", "microsoft", "email")

## Development

### Project Structure

```
multi-oauth2-proxy/
├── cmd/
│   └── multi-oauth2-proxy/   # Main application entry point
│       └── main.go
├── pkg/
│   ├── auth/
│   │   ├── email/            # Email authentication (magic links)
│   │   └── oauth2/           # OAuth2 authentication (Google, GitHub, Microsoft)
│   ├── authz/                # Authorization (email checking)
│   ├── config/               # Configuration management with auto-reload
│   ├── i18n/                 # Internationalization (Japanese/English)
│   ├── logging/              # Colored logging with TTY detection
│   ├── proxy/                # Reverse proxy with host-based routing
│   ├── ratelimit/            # Rate limiting (token bucket)
│   ├── server/               # HTTP server & routing
│   │   └── static/           # Embedded CSS and assets
│   └── session/              # Session management (in-memory)
├── web/                      # Frontend design system (Tailwind CSS 4)
│   ├── src/                  # Source files
│   │   ├── styles/           # CSS with design tokens
│   │   └── index.html        # Design system catalog
│   ├── dist/                 # Built assets (committed for Go embed)
│   └── package.json          # Node.js dependencies
├── config.example.yaml       # Example configuration
├── Makefile                  # Build automation
├── PLAN.md                   # Detailed design document
└── README.md
```

### Design System

This project includes a modern design system built with Tailwind CSS 4, providing:
- **Light/Dark theme support** with automatic detection
- **Responsive components** optimized for authentication UI
- **CSS custom properties** for easy customization
- **Component catalog** for development

#### Viewing the Design System

```bash
# Install web dependencies
make install-web

# Start the design system catalog
make dev-web
```

Visit `http://localhost:3000` to see all available components, colors, and page examples.

#### Building the Design System

```bash
# Build CSS and copy to Go embed directory
make build-web

# Or build everything (web + go)
make build
```

The built CSS is embedded in the Go binary via `//go:embed`, so no external assets are needed at runtime.

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Building

```bash
# Build for current platform
go build -o multi-oauth2-proxy ./cmd/multi-oauth2-proxy

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o multi-oauth2-proxy ./cmd/multi-oauth2-proxy

# Build with version
go build -ldflags "-X main.version=1.0.0" -o multi-oauth2-proxy ./cmd/multi-oauth2-proxy
```

## Configuration Reference

### Service

- `name`: Application name (displayed in UI and emails)
- `description`: Application description (displayed in UI)

### Server

- `host`: Listen address (default: `0.0.0.0`)
- `port`: Listen port (default: `4180`)

### Proxy

- `upstream`: Default backend application URL (required)
- `hosts`: (Optional) Host-based routing for multi-tenant support

Example with host-based routing:

```yaml
proxy:
  upstream: "http://default-backend:8080"  # Default backend
  hosts:
    app1.example.com: "http://backend1:8080"
    app2.example.com: "http://backend2:8080"
```

### Session

- `cookie_name`: Session cookie name (default: `_oauth2_proxy`)
- `cookie_secret`: Secret for cookie encryption (required, min 32 chars)
- `cookie_expire`: Session expiration duration (default: `168h` = 7 days)
- `cookie_secure`: Use secure cookies (set to `true` for HTTPS)
- `cookie_httponly`: HttpOnly flag (default: `true`)
- `cookie_samesite`: SameSite policy (default: `lax`)

### OAuth2

Configure one or more OAuth2 providers:

```yaml
oauth2:
  providers:
    - name: "google"              # Provider identifier
      display_name: "Google"      # Display name in UI
      client_id: "..."            # OAuth2 client ID
      client_secret: "..."        # OAuth2 client secret
      enabled: true               # Enable/disable provider
```

### Email Authentication

(Optional) Enable passwordless email authentication:

```yaml
email_auth:
  enabled: true
  sender_type: "smtp"  # or "sendgrid"

  # SMTP configuration
  smtp:
    host: "smtp.gmail.com"
    port: 587
    username: "your-email@gmail.com"
    password: "your-app-password"
    from: "noreply@yourdomain.com"

  # SendGrid configuration (alternative)
  sendgrid:
    api_key: "your-sendgrid-api-key"
    from: "noreply@yourdomain.com"

  # Token configuration
  token:
    expire: "15m"  # Token expiration time

  # Rate limiting
  rate_limit:
    requests: 3        # Max requests
    window: "1m"       # Per time window
```

### Authorization

Control who can access your application:

```yaml
authorization:
  # Allow specific email addresses
  allowed_emails:
    - "user@example.com"
    - "admin@company.com"

  # Allow all users from specific domains
  allowed_domains:
    - "@yourcompany.com"
    - "@trusted-partner.org"
```

### Logging

- `level`: Main log level (`debug`, `info`, `warn`, `error`)
- `module_level`: Sub-module log level
- `color`: Enable colored output (auto-detects TTY)

## OAuth2 Provider Setup

### Google OAuth2

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google+ API
4. Go to Credentials → Create Credentials → OAuth 2.0 Client ID
5. Configure consent screen
6. Add authorized redirect URIs:
   - `http://localhost:4180/oauth2/callback` (development)
   - `https://yourdomain.com/oauth2/callback` (production)
7. Copy Client ID and Client Secret to your `config.yaml`

### GitHub OAuth Apps

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click "New OAuth App"
3. Fill in the application details:
   - **Application name**: Your app name
   - **Homepage URL**: `https://yourdomain.com`
   - **Authorization callback URL**: `http://localhost:4180/oauth2/callback` (or your production URL)
4. Click "Register application"
5. Copy Client ID and generate a Client Secret
6. Add to your `config.yaml`:
   ```yaml
   oauth2:
     providers:
       - name: "github"
         client_id: "your-github-client-id"
         client_secret: "your-github-client-secret"
         enabled: true
   ```

### Microsoft Azure AD

1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to Azure Active Directory → App registrations
3. Click "New registration"
4. Fill in the application details:
   - **Name**: Your app name
   - **Supported account types**: Choose appropriate option
   - **Redirect URI**: Select "Web" and enter `http://localhost:4180/oauth2/callback`
5. After creation, note the "Application (client) ID"
6. Go to "Certificates & secrets" → "New client secret"
7. Copy the secret value immediately (it won't be shown again)
8. Go to "API permissions" → "Add a permission" → "Microsoft Graph"
   - Add `User.Read`, `email`, `profile`, `openid` permissions
9. Add to your `config.yaml`:
   ```yaml
   oauth2:
     providers:
       - name: "microsoft"
         client_id: "your-azure-client-id"
         client_secret: "your-azure-client-secret"
         enabled: true
   ```

## Internationalization

The UI supports Japanese and English. Language is detected in the following order:
1. Query parameter: `?lang=ja` or `?lang=en`
2. Cookie: `lang` cookie value
3. HTTP header: `Accept-Language` header

To set a default language preference, the application sets a cookie after detection.

## Endpoints

**Authentication:**
- `/login` - Login page (displays all available authentication methods)
- `/logout` - Logout and clear session
- `/oauth2/start/{provider}` - Initiate OAuth2 flow (provider: google, github, microsoft)
- `/oauth2/callback` - OAuth2 callback handler
- `/auth/email` - Email authentication login form
- `/auth/email/send` - Send magic link to email (POST)
- `/auth/email/verify` - Verify email token and create session (GET)

**Health Checks:**
- `/health` - Health check endpoint
- `/ready` - Readiness check endpoint

**Protected Routes:**
- `/*` - All other routes (requires authentication, proxied to backend)

## Security Considerations

1. **Cookie Secret**: Use a strong random secret (32+ characters) generated with `openssl rand -base64 32`
2. **HTTPS**: Always use `cookie_secure: true` in production with HTTPS
3. **Email Verification**: OAuth2 providers verify email addresses automatically
4. **Session Expiration**: Configure appropriate session timeouts (default: 7 days)
5. **Authorization**: Use domain restrictions when possible (e.g., `@yourcompany.com`)
6. **Rate Limiting**: Email authentication includes built-in rate limiting (3 requests/minute by default)
7. **CSRF Protection**: OAuth2 flows use state parameter for CSRF protection
8. **Token Security**: Email magic links use HMAC-SHA256 tokens with 15-minute expiration
9. **SMTP Credentials**: Store SMTP/SendGrid credentials securely, never commit to version control
10. **OAuth2 Secrets**: Keep client secrets secure and rotate them periodically

## Contributing

See [PLAN.md](PLAN.md) for the detailed design and development roadmap.

### Development Phases

- **Phase 1** (✅ Complete): Core foundation with single OAuth2 provider - 56.2% coverage
- **Phase 2** (✅ Complete): Email authentication and security enhancements - 63.1% coverage
- **Phase 3** (✅ Complete): Multi-provider support and advanced routing - 69.9% coverage
- **Phase 4** (Planned): Production features (Redis sessions, Prometheus metrics, structured logging)
- **Phase 5** (Planned): SSL/TLS automation (Let's Encrypt), MFA, WebAuthn

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Inspired by [oauth2-proxy](https://oauth2-proxy.github.io/oauth2-proxy/)
- Built with [chi router](https://github.com/go-chi/chi)
- Uses [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2) for OAuth2 implementation
