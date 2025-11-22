# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## ChatbotGate - Authentication Reverse Proxy

A lightweight, flexible authentication reverse proxy providing unified authentication through multiple OAuth2 providers and passwordless email authentication.

### Development Commands

```bash
# Build everything (web assets + Go binary)
make build

# Build only web assets (CSS, JS from web/)
make build-web

# Build only Go binary
make build-go

# Run all unit tests
make test

# Run tests with coverage report
make test-coverage

# Format Go code (gofmt)
make fmt

# Check code formatting (CI-style, exits on error)
make fmt-check

# Run linters (golangci-lint)
make lint

# Run all CI checks (format check + lint + test)
make ci

# Build and run server with example config
make run

# Install web dependencies
make install-web

# Run web dev server (design system catalog)
make dev-web

# Clean build artifacts
make clean
```

### Key Testing Commands

```bash
# Run specific package tests
go test ./pkg/middleware/auth/oauth2
go test ./pkg/middleware/session

# Run tests with verbose output
go test -v ./pkg/...

# Run tests with race detector
go test -race ./...

# Run e2e tests (requires Docker)
cd e2e && make test
```

### Architecture Overview

**Layered Middleware Design:**
```
HTTP Request → Auth Check → Authorization → Rules → Forwarding → Proxy → Upstream
```

**Project Structure:**
```
chatbotgate/
├── cmd/chatbotgate/          # Main CLI entry point
├── pkg/
│   ├── middleware/           # Authentication & authorization middleware
│   │   ├── auth/
│   │   │   ├── oauth2/       # OAuth2 providers (Google, GitHub, Microsoft, Custom)
│   │   │   └── email/        # Passwordless email authentication
│   │   ├── authz/            # Authorization (email/domain whitelisting)
│   │   ├── session/          # Session management with multiple backends
│   │   ├── rules/            # Path-based access control (allow/auth/deny)
│   │   ├── forwarding/       # User info forwarding to upstream
│   │   ├── config/           # Middleware configuration
│   │   ├── core/             # Core middleware logic
│   │   └── factory/          # Middleware factory pattern
│   ├── proxy/                # Reverse proxy with WebSocket support
│   │   ├── core/             # Proxy implementation
│   │   └── config/           # Proxy configuration
│   └── shared/               # Reusable components
│       ├── kvs/              # Key-Value Store abstraction (Memory/LevelDB/Redis)
│       ├── i18n/             # Internationalization (en/ja)
│       ├── logging/          # Structured logging
│       ├── config/           # Config utilities with live reload
│       └── filewatcher/      # File watching for config hot-reload
├── web/                      # Web UI assets (HTML, CSS, TypeScript)
│   ├── src/                  # TypeScript source
│   ├── public/               # Static assets
│   └── vite.config.ts        # Vite build configuration
├── e2e/                      # End-to-end tests with Playwright
│   ├── src/                  # E2E test scenarios
│   ├── config/               # Test configurations
│   └── docker/               # Docker setup for testing
├── email/                    # Email templates (HTML/text)
└── config.example.yaml       # Comprehensive config example
```

### Core Concepts

**1. Provider Pattern:**
All authentication providers (OAuth2, email) implement common interfaces, enabling easy extension:
- `oauth2.Provider`: OAuth2/OIDC provider interface
- `email.Sender`: Email sender interface (SMTP/SendGrid)

**2. KVS Abstraction:**
Unified Key-Value Store interface supporting multiple backends:
- Memory: Fast, ephemeral (development)
- LevelDB: Persistent, embedded (single server)
- Redis: Distributed, scalable (production)

All use the same `kvs.Store` interface with namespace isolation for sessions, tokens, and rate limits.

**3. Middleware Composition:**
Standard Go middleware pattern (`func(http.Handler) http.Handler`) allows flexible composition:
```go
handler := authMiddleware(
    authzMiddleware(
        rulesMiddleware(
            forwardingMiddleware(
                proxyHandler
            )
        )
    )
)
```

**4. Configuration-Driven:**
All components configured via YAML with live reload support (fsnotify).
Changes to config.yaml are detected and applied without restart (where supported).

**5. Authentication Path Prefix:**
All auth endpoints use a configurable prefix (default: `/_auth`) to avoid conflicts:
- `/_auth/login` - Login page
- `/_auth/oauth2/start/{provider}` - OAuth2 start
- `/_auth/oauth2/callback` - OAuth2 callback
- `/_auth/email` - Email login
- `/_auth/email/send` - Send magic link
- `/_auth/email/verify` - Verify token
- `/_auth/logout` - Logout

**6. Standardized OAuth2 Fields:**
All OAuth2 providers populate standardized fields in `UserInfo.Extra`:
- `_email`: User email address (common across all providers)
- `_username`: User display name (with provider-specific fallbacks)
- `_avatar_url`: Profile picture URL (when available)

These fields enable consistent user info forwarding regardless of provider.

**7. Health Check System:**
The middleware maintains internal state for health monitoring:
- `/_auth/health` - Readiness probe (returns 200 when ready, 503 when starting/draining)
- `/_auth/health?probe=live` - Liveness probe (always returns 200 if process is alive)

Health states:
- `starting` - Initial state after creation
- `ready` - Middleware is ready (after `SetReady()` call)
- `draining` - Graceful shutdown in progress (after `SetDraining()` call)

The `MiddlewareManager` is responsible for calling `SetReady()` after initialization and `SetDraining()` on SIGTERM.

### Web Development

The `web/` directory contains the authentication UI built with:
- TypeScript + Vite for building
- Water.css for styling (classless CSS framework)
- Theme support (Auto/Light/Dark)
- Multi-language (English/Japanese)

**Build Process:**
1. `cd web && yarn build` - Builds CSS/JS to `web/dist/`
2. `web/copy-to-pkg.js` - Copies built assets to `pkg/middleware/assets/`
3. Assets are embedded in Go binary via `//go:embed`

### Testing Strategy

**Unit Tests:**
- Each package has `*_test.go` files
- Mock implementations for interfaces (Provider, Store, Sender)
- Table-driven tests preferred
- Target: 80%+ coverage

**Integration Tests:**
- Test full middleware chain with mock providers
- Session flow testing
- Rule evaluation testing

**E2E Tests (e2e/):**
- Playwright-based browser automation
- Tests full OAuth2 flow with mock providers
- Email authentication flow testing
- Docker Compose for test environment

### OAuth2 Provider Implementation

**Default Scopes:**
Each provider has defaults when `scopes` config is empty:
- Google: `openid`, `userinfo.email`, `userinfo.profile`
- GitHub: `user:email`, `read:user`
- Microsoft: `openid`, `profile`, `email`, `User.Read`
- Custom: `openid`, `email`, `profile`

**Important:** When custom scopes are specified in config, defaults are NOT added automatically. Explicitly include required scopes for user information.

**Creating Custom Providers:**
Implement the `oauth2.Provider` interface:
```go
type Provider interface {
    Name() string
    DisplayName() string
    IconURL() string
    GetAuthURL(state string) string
    Exchange(ctx context.Context, code string) (*UserInfo, error)
}
```

### Configuration Hot Reload

**Reloadable:**
- Service name and branding
- OAuth2 provider settings
- Authorization rules
- Logging levels
- Access control rules

**Requires Restart:**
- Server host/port
- Session cookie secret
- KVS backend type

### User Information Forwarding

Forwards authenticated user data to upstream apps via:
- Query parameters
- HTTP headers
- Encryption support (AES-256-GCM)
- Compression support (gzip)

**Available Paths:**
- `email`, `username`, `provider` - Basic user fields
- `_email`, `_username`, `_avatar_url` - Standardized OAuth2 fields
- `extra.{field}` - Provider-specific fields
- `extra.secrets.access_token` - OAuth2 tokens
- `.` - Entire user object as JSON

**Filters:**
- `encrypt` - AES-256-GCM encryption
- `zip` - gzip compression
- `base64` - Base64 encoding (auto for binary)

### Access Control Rules

Path-based rules with first-match-wins evaluation:

**Rule Types:**
- `exact`: Exact path match
- `prefix`: Path prefix match
- `regex`: Regular expression match
- `minimatch`: Glob pattern match (supports `**/*.js`)
- `all`: Catch-all rule

**Actions:**
- `allow`: Allow without authentication
- `auth`: Require authentication
- `deny`: Deny access (403)

### Docker & Deployment

**Official Images:**
- Docker Hub: `ideamans/chatbotgate`
- Multi-arch: AMD64 + ARM64
- Tags: `latest`, `v1.0.0`, `v1.0.0-amd64`, `v1.0.0-arm64`

**Production Recommendations:**
- Use Redis KVS for sessions
- Enable `cookie_secure: true` with HTTPS
- Pin Docker image versions (not `latest`)
- Configure resource limits (0.5-1 CPU, 256-512MB memory)
- Use health checks: `GET /_auth/health`

### CI/CD

GitHub Actions workflows:
- **CI** (`ci.yml`): Lint, format check, unit tests, e2e tests
- **Release** (`release.yml`): GoReleaser + Docker Hub publish on tags

### Common Patterns

**Context Propagation:**
Always pass `context.Context` through the stack for cancellation and timeouts.

**Error Wrapping:**
Use `fmt.Errorf("context: %w", err)` to maintain error chains.

**Interface Design:**
All major components use interfaces for testability and extensibility.

**Namespace Isolation:**
KVS uses namespace prefixes to separate concerns:
- `session:` - Session data
- `token:` - Email auth tokens
- `email_quota:` - Email send quota/rate limiting (configurable via `email_auth.limit_per_minute`, default: 5/min)

### Pre-Commit Checklist

**IMPORTANT:** Before creating any git commit, always run the following checks in order:

1. **Check code formatting:**
   ```bash
   make fmt-check
   ```
   If formatting issues are found, fix them with:
   ```bash
   make fmt
   ```

2. **Run linters:**
   ```bash
   make lint
   ```
   If linter issues are found, fix them before proceeding.

3. **Run tests:**
   ```bash
   make test
   ```
   If test failures occur, fix them before proceeding.

4. **Evaluate fixes:**
   - If fixes are minor and straightforward, proceed with the commit
   - If fixes require significant changes or are complex, explain the issues to the user and abort the commit process

**Quick check:** You can also run all checks at once:
```bash
make ci
```

This ensures code quality and prevents CI failures.

### Important Files

- `config.example.yaml` - Comprehensive configuration reference with comments
- `README.md` - User-facing documentation and quick start
- `GUIDE.md` - Detailed deployment and configuration guide
- `MODULE.md` - Developer guide for using as Go module
- `Dockerfile` - Multi-stage build with minimal runtime image
- `docker-compose.yml` - Development setup with Redis
