# ChatbotGate User Guide

Complete guide for deploying, configuring, and operating ChatbotGate as an authentication reverse proxy.

## Table of Contents

- [Introduction](#introduction)
- [Installation](#installation)
- [Configuration](#configuration)
  - [Service Settings](#service-settings)
  - [Server Configuration](#server-configuration)
  - [Proxy Configuration](#proxy-configuration)
  - [Session Management](#session-management)
  - [OAuth2 Providers](#oauth2-providers)
  - [Email Authentication](#email-authentication)
  - [Authorization](#authorization)
  - [KVS Backend](#kvs-backend)
  - [User Information Forwarding](#user-information-forwarding)
  - [Access Control Rules](#access-control-rules)
  - [Assets Optimization](#assets-optimization)
  - [Logging](#logging)
- [Running the Server](#running-the-server)
- [Authentication Flow](#authentication-flow)
- [Production Deployment](#production-deployment)
  - [Docker Hub Deployment](#docker-hub-deployment)
  - [Production Configuration Best Practices](#production-configuration-best-practices)
  - [Reverse Proxy Setup (Nginx)](#reverse-proxy-setup-nginx)
  - [Kubernetes Deployment](#kubernetes-deployment)
  - [Monitoring & Observability](#monitoring--observability)
  - [Scaling Considerations](#scaling-considerations)
  - [CI/CD Integration](#cicd-integration)
- [Advanced Topics](#advanced-topics)
- [Troubleshooting](#troubleshooting)

## Introduction

ChatbotGate is an authentication reverse proxy that sits between your users and your upstream application. It intercepts requests, authenticates users through OAuth2 or email, and then proxies authenticated requests to your backend.

### Key Concepts

- **Authentication Path Prefix**: All auth-related endpoints use this prefix (default: `/_auth`)
- **Upstream**: Your backend application that ChatbotGate proxies to
- **Provider**: An OAuth2/OIDC identity provider (Google, GitHub, etc.)
- **Session**: Encrypted cookie storing user authentication state
- **KVS**: Key-Value Store backend for sessions, tokens, and rate limits

## Installation

### From Source

```bash
# Clone repository
git clone https://github.com/ideamans/chatbotgate.git
cd chatbotgate

# Build binary
go build -o chatbotgate ./cmd/chatbotgate

# Verify installation
./chatbotgate --version
```

### Using Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/ideamans/chatbotgate/releases):

```bash
# Example for Linux amd64
wget https://github.com/ideamans/chatbotgate/releases/latest/download/chatbotgate-linux-amd64
chmod +x chatbotgate-linux-amd64
mv chatbotgate-linux-amd64 /usr/local/bin/chatbotgate
```

### Using Docker

ChatbotGate provides official Docker images on [Docker Hub](https://hub.docker.com/r/ideamans/chatbotgate) with multi-architecture support (amd64/arm64).

#### Quick Start with Docker

```bash
# Pull the latest version
docker pull ideamans/chatbotgate:latest

# Run with config file
docker run -d \
  --name chatbotgate \
  -p 4180:4180 \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  ideamans/chatbotgate:latest

# View logs
docker logs -f chatbotgate
```

#### Using Docker Compose

Create a `docker-compose.yml`:

```yaml
version: '3.8'

services:
  chatbotgate:
    image: ideamans/chatbotgate:latest
    container_name: chatbotgate
    restart: unless-stopped
    ports:
      - "4180:4180"
    volumes:
      - ./config.yaml:/app/config/config.yaml:ro
    environment:
      - TZ=Asia/Tokyo
    # Optional: use external network to connect to upstream
    # networks:
    #   - myapp-network

  # Optional: Redis for production session storage
  redis:
    image: redis:7-alpine
    container_name: chatbotgate-redis
    restart: unless-stopped
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes

volumes:
  redis-data:
```

Start the services:

```bash
docker-compose up -d
docker-compose logs -f chatbotgate
```

#### Available Tags

- `latest` - Latest stable release (multi-arch)
- `v1.0.0` - Specific version (multi-arch)
- `v1.0.0-amd64` - AMD64 architecture only
- `v1.0.0-arm64` - ARM64 architecture only
- `sha-abc1234` - Specific commit (for testing)

#### Configuration with Environment Variables

While YAML configuration is recommended, you can also use environment variables:

```bash
docker run -d \
  --name chatbotgate \
  -p 4180:4180 \
  -e CHATBOTGATE_SERVER_PORT=4180 \
  -e CHATBOTGATE_PROXY_UPSTREAM=http://myapp:8080 \
  -e CHATBOTGATE_SESSION_COOKIE_SECRET=your-secret-key \
  ideamans/chatbotgate:latest
```

## Configuration

Configuration is done via a YAML file. See `config.example.yaml` for a complete example.

### Service Settings

Basic service information and branding:

```yaml
service:
  # Service name displayed on login page
  name: "My Application"

  # Optional description
  description: "Secure authentication for My Application"

  # Optional: Icon URL (48px, left of title)
  icon_url: "https://example.com/icon.svg"

  # Optional: Logo URL (larger, above title)
  logo_url: "https://example.com/logo.svg"

  # Optional: Logo width (default: 200px)
  logo_width: "150px"
```

### Server Configuration

HTTP server settings:

```yaml
server:
  # Listen address (0.0.0.0 = all interfaces)
  host: "0.0.0.0"

  # Listen port
  port: 4180

  # Authentication path prefix
  # All auth endpoints will be under this prefix
  auth_path_prefix: "/_auth"

  # Base URL for email links and redirects
  # Set this when behind a reverse proxy
  base_url: "https://auth.example.com"

  # OAuth2 callback URL override
  # Useful when external URL differs from internal
  callback_url: "https://auth.example.com/_auth/oauth2/callback"
```

**CLI Overrides:**

```bash
# Override host
./chatbotgate -config config.yaml --host 127.0.0.1

# Override port
./chatbotgate -config config.yaml -p 8080

# Both
./chatbotgate -config config.yaml --host 0.0.0.0 --port 4180
```

### Proxy Configuration

Configure upstream application:

```yaml
proxy:
  # Main upstream backend
  upstream:
    url: "http://localhost:8080"

    # Optional: Secret header for upstream authentication
    # Protects upstream from direct access
    secret:
      header: "X-Chatbotgate-Secret"
      value: "your-secret-token-here"
```

The secret header is added to all proxied requests, allowing your upstream to verify requests came through ChatbotGate.

### Session Management

Session cookie configuration:

```yaml
session:
  # Cookie name
  cookie_name: "_oauth2_proxy"

  # Cookie secret (REQUIRED, 32+ characters)
  # Generate with: openssl rand -base64 32
  cookie_secret: "CHANGE-THIS-TO-A-RANDOM-SECRET"

  # Session expiration (Go duration format)
  cookie_expire: "168h"  # 7 days

  # Secure flag (HTTPS only, enable in production)
  cookie_secure: true

  # HttpOnly flag (prevent JavaScript access)
  cookie_httponly: true

  # SameSite policy
  cookie_samesite: "lax"  # "strict", "lax", or "none"
```

**Security Best Practices:**

- Generate a strong random secret: `openssl rand -base64 32`
- Set `cookie_secure: true` when using HTTPS
- Keep `cookie_httponly: true` to prevent XSS attacks
- Use `cookie_samesite: "strict"` for maximum CSRF protection

### OAuth2 Providers

Configure OAuth2/OIDC providers:

#### Google

```yaml
oauth2:
  providers:
    - name: "google"
      display_name: "Google"
      client_id: "YOUR-CLIENT-ID.apps.googleusercontent.com"
      client_secret: "YOUR-CLIENT-SECRET"
      disabled: false  # Set to true to hide from login page

      # Optional: Custom icon
      icon_url: "https://example.com/google-icon.svg"

      # Optional: Custom scopes
      # If not specified, uses default scopes (recommended for user info)
      # If specified, ONLY uses these scopes (defaults not added)
      scopes:
        - "openid"  # Must include defaults if customizing
        - "https://www.googleapis.com/auth/userinfo.email"
        - "https://www.googleapis.com/auth/userinfo.profile"
        - "https://www.googleapis.com/auth/calendar.readonly"  # Additional scope
```

**Default Scopes** (when `scopes` not specified):
- `openid` - OIDC authentication
- `https://www.googleapis.com/auth/userinfo.email` - User email address
- `https://www.googleapis.com/auth/userinfo.profile` - User profile (name, picture)

**Standardized Fields** (available in forwarding):
- `_email`: User's email address
- `_username`: User's display name
- `_avatar_url`: User's profile picture URL

**Setup Instructions:**

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable "Google+ API"
4. Create OAuth 2.0 credentials (Web application)
5. Add authorized redirect URI: `https://your-domain.com/_auth/oauth2/callback`
6. Copy Client ID and Client Secret to config

#### GitHub

```yaml
oauth2:
  providers:
    - name: "github"
      display_name: "GitHub"
      client_id: "YOUR-GITHUB-CLIENT-ID"
      client_secret: "YOUR-GITHUB-CLIENT-SECRET"

      # Optional: Custom scopes
      # If not specified, uses default scopes (recommended for user info)
      # If specified, ONLY uses these scopes (defaults not added)
      scopes:
        - "user:email"  # Must include defaults if customizing
        - "read:user"   # User profile access
        - "repo"        # Additional: Repository access
        - "read:org"    # Additional: Organization membership
```

**Default Scopes** (when `scopes` not specified):
- `user:email` - User email addresses (verified)
- `read:user` - User profile data (name, login, avatar)

**Standardized Fields** (available in forwarding):
- `_email`: User's verified email address
- `_username`: User's display name (fallback to login name if not set)
- `_avatar_url`: User's profile picture URL

**Setup Instructions:**

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click "New OAuth App"
3. Set Authorization callback URL: `https://your-domain.com/_auth/oauth2/callback`
4. Copy Client ID and Client Secret to config

#### Microsoft (Azure AD)

```yaml
oauth2:
  providers:
    - name: "microsoft"
      display_name: "Microsoft"
      client_id: "YOUR-AZURE-APP-ID"
      client_secret: "YOUR-AZURE-CLIENT-SECRET"

      # Optional: Custom scopes
      # If not specified, uses default scopes (recommended for user info)
      # If specified, ONLY uses these scopes (defaults not added)
      scopes:
        - "openid"          # Must include defaults if customizing
        - "profile"         # User profile data
        - "email"           # User email address
        - "User.Read"       # Microsoft Graph user info
        - "Calendars.Read"  # Additional: Calendar access
        - "Mail.Read"       # Additional: Email access
```

**Default Scopes** (when `scopes` not specified):
- `openid` - OIDC authentication
- `profile` - User profile (displayName)
- `email` - User email address
- `User.Read` - Microsoft Graph user information

**Standardized Fields** (available in forwarding):
- `_email`: User's email address
- `_username`: User's display name
- `_avatar_url`: Empty (Microsoft requires separate photo endpoint)

**Setup Instructions:**

1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to "Azure Active Directory" → "App registrations"
3. Create new registration
4. Add redirect URI: `https://your-domain.com/_auth/oauth2/callback`
5. Generate client secret in "Certificates & secrets"
6. Copy Application ID and Client Secret to config

#### Custom OIDC Provider

```yaml
oauth2:
  providers:
    - name: "custom-oidc"
      type: "custom"
      display_name: "My Identity Provider"
      client_id: "YOUR-CLIENT-ID"
      client_secret: "YOUR-CLIENT-SECRET"
      icon_url: "https://your-idp.com/logo.svg"

      # OIDC endpoints
      auth_url: "https://your-idp.com/oauth/authorize"
      token_url: "https://your-idp.com/oauth/token"
      userinfo_url: "https://your-idp.com/oauth/userinfo"

      # Optional: JWKS URL for token validation
      jwks_url: "https://your-idp.com/.well-known/jwks.json"

      # Optional: Skip TLS verification (dev only!)
      insecure_skip_verify: false
```

### Email Authentication

Passwordless email authentication via magic links:

```yaml
email_auth:
  enabled: true

  # Sender type: "smtp" or "sendgrid"
  sender_type: "smtp"

  # Token expiration
  token:
    expire: "15m"  # 15 minutes
```

#### SMTP Configuration

```yaml
email_auth:
  sender_type: "smtp"
  smtp:
    host: "smtp.gmail.com"
    port: 587
    username: "your-email@gmail.com"
    password: "your-app-password"
    from: "noreply@example.com"
    from_name: "My Application"
    tls: false        # Direct TLS (port 465)
    starttls: true    # STARTTLS (port 587)
```

**Gmail Setup:**

1. Enable 2-Factor Authentication on your Google account
2. Generate an [App Password](https://myaccount.google.com/apppasswords)
3. Use the app password in `password` field

#### SendGrid Configuration

```yaml
email_auth:
  sender_type: "sendgrid"
  sendgrid:
    api_key: "SG.xxxxxxxxxxxxxxxxxxxx"
    from: "noreply@example.com"
    from_name: "My Application"

    # Optional: Custom endpoint (for proxies)
    endpoint_url: "https://sendgrid-proxy.example.com"
```

**SendGrid Setup:**

1. Sign up at [SendGrid](https://sendgrid.com/)
2. Create an API key with "Mail Send" permissions
3. Verify sender email address or domain
4. Copy API key to config

**User Information Fields:**

Email authentication provides the same standardized fields as OAuth2 for consistent forwarding:

- `email`: User email address
- `username`: Email local part (before @)
- `provider`: "email"
- `_email`: User email address (standardized field)
- `_username`: Email local part (standardized field)
- `_avatar_url`: Empty string (standardized field)
- `userpart`: Email local part (same as `_username`)

These fields can be used in [User Information Forwarding](#user-information-forwarding) configuration.

### Authorization

Control who can access your application:

```yaml
authorization:
  # Allowed email addresses and domains
  # Entries starting with @ are domain wildcards
  # Empty list [] allows ALL authenticated users
  allowed:
    - "alice@example.com"      # Specific email
    - "bob@company.com"        # Another email
    - "@example.org"           # All @example.org emails
    - "@trusted-domain.com"    # All @trusted-domain.com emails
```

**Examples:**

```yaml
# Allow everyone (no whitelist)
authorization:
  allowed: []

# Allow only specific users
authorization:
  allowed:
    - "admin@example.com"
    - "manager@example.com"

# Allow entire domain
authorization:
  allowed:
    - "@example.com"

# Mix and match
authorization:
  allowed:
    - "external-user@gmail.com"
    - "@company.com"
    - "@partner-company.com"
```

### KVS Backend

Key-Value Store configuration for sessions, tokens, and rate limits:

#### Memory (Development)

```yaml
kvs:
  default:
    type: "memory"
    memory:
      cleanup_interval: "5m"
```

**Pros:** Fast, simple, no dependencies
**Cons:** Not persistent, single-server only

#### LevelDB (Single Server)

```yaml
kvs:
  default:
    type: "leveldb"
    leveldb:
      # Storage path (empty = OS temp/cache dir)
      path: "/var/lib/chatbotgate/kvs"

      # Sync writes to disk (safer but slower)
      sync_writes: false

      # Cleanup interval for expired keys
      cleanup_interval: "5m"
```

**Pros:** Persistent, fast, embedded
**Cons:** Single-server only, not distributed

#### Redis (Production)

```yaml
kvs:
  default:
    type: "redis"
    redis:
      addr: "localhost:6379"
      password: ""
      db: 0
      pool_size: 0  # 0 = auto (10 * CPU cores)
```

**Pros:** Distributed, scalable, production-ready
**Cons:** Requires Redis server

#### Namespace Isolation

All storage types support namespace isolation:

```yaml
kvs:
  default:
    type: "redis"
    redis:
      addr: "localhost:6379"

  # Customize namespace names (optional)
  namespaces:
    session: "session"
    token: "token"
    ratelimit: "ratelimit"
```

**How It Works:**
- **Memory**: Separate instance per namespace
- **LevelDB**: Separate directory per namespace
- **Redis**: Key prefix (e.g., `session:abc123`)

#### Dedicated Backends (Advanced)

Override storage for specific use cases:

```yaml
kvs:
  # Default for all
  default:
    type: "memory"

  # Dedicated Redis for sessions
  session:
    type: "redis"
    redis:
      addr: "localhost:6379"
      db: 1

  # Dedicated LevelDB for rate limiting
  ratelimit:
    type: "leveldb"
    leveldb:
      path: "/var/lib/chatbotgate/ratelimit"
```

### User Information Forwarding

Forward authenticated user data to upstream applications:

```yaml
forwarding:
  # Encryption settings (required if using 'encrypt' filter)
  encryption:
    key: "CHANGE-THIS-TO-32-CHAR-SECRET"
    algorithm: "aes-256-gcm"

  # Field definitions
  fields:
    # Example 1: Email as query param and header
    - path: email
      query: email
      header: X-Auth-Email

    # Example 2: Username with encryption
    - path: username
      header: X-Auth-User
      filters: encrypt

    # Example 3: Email with encryption and compression
    - path: email
      query: user_email
      header: X-Auth-Encrypted-Email
      filters:
        - encrypt
        - zip

    # Example 4: Standardized avatar URL (common across all OAuth2 providers)
    - path: _avatar_url
      header: X-Avatar-URL

    # Example 5: Entire user object as JSON
    - path: .
      query: userinfo
      filters:
        - encrypt
        - zip

    # Example 6: OAuth2 access token
    - path: extra.secrets.access_token
      header: X-Access-Token
      filters: encrypt

    # Example 7: Provider name
    - path: provider
      header: X-Auth-Provider

    # Example 8: Standardized user fields (common across all OAuth2 providers and email auth)
    - path: _email
      header: X-User-Email
    - path: _username
      header: X-User-Name
    - path: _avatar_url
      header: X-User-Avatar

    # Example 9: Email auth userpart (same as _username for email auth)
    - path: userpart
      header: X-User-Part
```

**Available User Fields:**
- `email`: User email address
- `username`: Username (provider-dependent; for email auth: email local part before @)
- `provider`: Provider name (google, github, microsoft, email)

**Standardized Fields** (common across all OAuth2 providers and email auth):
- `_email`: User email address (same as `email`)
- `_username`: User display name
  - OAuth2 providers: GitHub (name → login fallback), Microsoft (displayName), Google (name)
  - Email auth: email local part (before @)
- `_avatar_url`: User profile picture URL
  - OAuth2 providers: Google and GitHub supported; empty for Microsoft
  - Email auth: empty

**Provider-Specific Fields** (under `extra`):
- Google: `email`, `name`, `picture`, `verified_email`, `given_name`, `family_name`
- GitHub: `email`, `name`, `login`, `avatar_url`, plus other public profile data
- Microsoft: `email`, `displayName`, `userPrincipalName`, `preferredUsername`
- Email auth: `userpart` (email local part before @, same as `_username`)

**OAuth2 Tokens** (under `extra.secrets`):
- `extra.secrets.access_token`: OAuth2 access token
- `extra.secrets.refresh_token`: OAuth2 refresh token

**Special Paths:**
- `.`: Entire user object as JSON

**Available Filters:**
- `encrypt`: AES-256-GCM encryption (requires encryption config)
- `zip`: gzip compression
- `base64`: Base64 encoding (auto-added for binary data)

**Filter Order:** Filters are applied left-to-right (e.g., `encrypt,zip` = encrypt first, then compress)

**Decryption Example (Node.js):**

```javascript
const crypto = require('crypto');
const zlib = require('zlib');

function decryptUserInfo(encrypted, key) {
  // Base64 decode
  const buffer = Buffer.from(encrypted, 'base64');

  // Decompress (if 'zip' filter was used)
  const compressed = zlib.gunzipSync(buffer);

  // Extract nonce and ciphertext
  const nonce = compressed.slice(0, 12);
  const ciphertext = compressed.slice(12);

  // Decrypt
  const decipher = crypto.createDecipheriv('aes-256-gcm', key, nonce);
  const decrypted = Buffer.concat([
    decipher.update(ciphertext),
    decipher.final()
  ]);

  return JSON.parse(decrypted.toString());
}
```

### Access Control Rules

Path-based access control with pattern matching:

```yaml
rules:
  # Allow public static files without authentication
  - prefix: "/static/"
    action: allow
    description: "Public static assets"

  # Health check endpoint
  - exact: "/health"
    action: allow
    description: "Health check"

  # Public API endpoints (regex)
  - regex: "^/api/public/"
    action: allow
    description: "Public API"

  # JavaScript and CSS files (minimatch/glob)
  - minimatch: "**/*.{js,css}"
    action: allow
    description: "Frontend assets"

  # Deny admin access
  - prefix: "/admin/"
    action: deny
    description: "Admin area blocked"

  # Default: require authentication
  - all: true
    action: auth
    description: "Require auth for everything else"
```

**Rule Types:**
- `exact`: Exact path match
- `prefix`: Path prefix match
- `regex`: Regular expression match
- `minimatch`: Glob pattern match (supports `*`, `**`, `?`, `{a,b}`)
- `all`: Catch-all rule (matches everything)

**Actions:**
- `allow`: Allow access without authentication
- `auth`: Require authentication
- `deny`: Deny access (403 Forbidden)

**Evaluation Order:**
- Rules are evaluated top-to-bottom
- First matching rule wins
- If no rule matches, default is `deny`

**Example: Public homepage, authenticated app:**

```yaml
rules:
  - exact: "/"
    action: allow
  - prefix: "/app/"
    action: auth
  - all: true
    action: deny
```

### Assets Optimization

Control CSS and JavaScript loading:

```yaml
assets:
  optimization:
    # Dify chatbot integration optimizations
    # Adds transparent backgrounds and responsive layout
    dify: false
```

When `dify: true`, adds optimizations for:
- Transparent backgrounds
- Bottom-aligned layout
- Responsive settings toggle
- Iframe embedding support

### Logging

Configure logging output:

```yaml
logging:
  # Main log level: debug, info, warn, error
  level: "info"

  # Module-specific log level
  module_level: "debug"

  # Colored output (auto-detects TTY)
  color: true
```

## Running the Server

### Basic Usage

```bash
# With config file
./chatbotgate -config config.yaml

# Override host/port
./chatbotgate -config config.yaml --host 127.0.0.1 -p 8080

# Show version
./chatbotgate --version

# Show help
./chatbotgate --help
```

### With Docker

```bash
# Basic run
docker run -p 4180:4180 \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  ideamans/chatbotgate:latest

# With environment variables
docker run -p 4180:4180 \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  -e COOKIE_SECRET="your-secret" \
  ideamans/chatbotgate:latest
```

### With Docker Compose

```yaml
version: '3.8'

services:
  chatbotgate:
    image: ideamans/chatbotgate:latest
    ports:
      - "4180:4180"
    volumes:
      - ./config.yaml:/app/config/config.yaml:ro
    environment:
      - LOG_LEVEL=info
    depends_on:
      - redis
      - upstream

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  upstream:
    image: your-app:latest
    ports:
      - "8080:8080"
```

```bash
docker-compose up
```

### Systemd Service (Linux)

Create `/etc/systemd/system/chatbotgate.service`:

```ini
[Unit]
Description=ChatbotGate Authentication Proxy
After=network.target

[Service]
Type=simple
User=chatbotgate
Group=chatbotgate
WorkingDirectory=/opt/chatbotgate
ExecStart=/opt/chatbotgate/chatbotgate -config /etc/chatbotgate/config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable chatbotgate
sudo systemctl start chatbotgate
sudo systemctl status chatbotgate
```

### Behind Nginx

Example Nginx configuration:

```nginx
server {
    listen 443 ssl http2;
    server_name auth.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:4180;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

Update `config.yaml`:

```yaml
server:
  base_url: "https://auth.example.com"
  callback_url: "https://auth.example.com/_auth/oauth2/callback"

session:
  cookie_secure: true
```

## Authentication Flow

### OAuth2 Flow

```
1. User visits: https://example.com/app
   ↓
2. ChatbotGate: No session → Redirect to /_auth/login
   ↓
3. User clicks "Sign in with Google"
   ↓
4. ChatbotGate: Redirect to /_auth/oauth2/start/google
   ↓
5. Redirect to Google OAuth2 authorize endpoint
   ↓
6. User authenticates with Google
   ↓
7. Google redirects to: /_auth/oauth2/callback?code=...
   ↓
8. ChatbotGate: Exchange code for token, fetch user info
   ↓
9. ChatbotGate: Check authorization (whitelist)
   ↓
10. Create session, set cookie, redirect to /app
    ↓
11. Authenticated request proxied to upstream
```

### Email Authentication Flow

```
1. User visits: https://example.com/app
   ↓
2. ChatbotGate: No session → Redirect to /_auth/login
   ↓
3. User clicks "Sign in with Email"
   ↓
4. User enters email → POST /_auth/email/send
   ↓
5. ChatbotGate: Generate token, send email with magic link
   ↓
6. User clicks link: /_auth/email/verify?token=...
   ↓
7. ChatbotGate: Verify token, check authorization
   ↓
8. Create session, set cookie, redirect to /app
   ↓
9. Authenticated request proxied to upstream
```

### Session Lifetime

- Sessions expire after `cookie_expire` duration (default: 7 days)
- Sliding expiration: Each request refreshes the expiration
- Logout: Clears session and redirects to login

## Production Deployment

### Docker Hub Deployment

ChatbotGate provides official Docker images on [Docker Hub](https://hub.docker.com/r/ideamans/chatbotgate) with automatic builds on every release.

#### Production Docker Compose Example

```yaml
version: '3.8'

services:
  chatbotgate:
    image: ideamans/chatbotgate:v1.0.0  # Pin to specific version
    container_name: chatbotgate
    restart: unless-stopped
    ports:
      - "4180:4180"
    volumes:
      - ./config.yaml:/app/config/config.yaml:ro
      - ./leveldb:/data/leveldb  # For LevelDB persistence
    environment:
      - TZ=Asia/Tokyo
    depends_on:
      - redis
    networks:
      - app-network
    # Resource limits
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M

  redis:
    image: redis:7-alpine
    container_name: chatbotgate-redis
    restart: unless-stopped
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
    networks:
      - app-network

  # Your upstream application
  upstream-app:
    image: your-app:latest
    container_name: upstream-app
    restart: unless-stopped
    networks:
      - app-network

volumes:
  redis-data:

networks:
  app-network:
    driver: bridge
```

#### Production Configuration Best Practices

1. **Pin Docker Image Version**
   ```yaml
   image: ideamans/chatbotgate:v1.0.0  # Not :latest
   ```

2. **Use Redis for Session Storage**
   ```yaml
   kvs:
     default:
       type: "redis"
       redis:
         addr: "redis:6379"
         pool_size: 20
   ```

3. **Enable HTTPS with Secure Cookies**
   ```yaml
   session:
     cookie_secure: true
     cookie_httponly: true
     cookie_samesite: "strict"
   ```

4. **Use Strong Secrets**
   ```bash
   # Generate random secret (32+ characters)
   openssl rand -base64 32
   ```

5. **Configure Resource Limits**
   - CPU: 0.5-1.0 cores per instance
   - Memory: 256-512 MB per instance
   - Redis: 256-512 MB depending on session count

6. **Enable Structured Logging**
   ```yaml
   logging:
     level: "info"  # Use "debug" only for troubleshooting
     color: false   # Better for log aggregators
   ```

7. **Set Up Health Checks**
   ```yaml
   healthcheck:
     test: ["CMD", "wget", "--spider", "-q", "http://localhost:4180/health"]
     interval: 30s
     timeout: 10s
     retries: 3
     start_period: 10s
   ```

#### Reverse Proxy Setup (Nginx)

Place ChatbotGate behind Nginx for SSL termination:

```nginx
upstream chatbotgate {
    server localhost:4180;
}

server {
    listen 443 ssl http2;
    server_name app.example.com;

    ssl_certificate /etc/ssl/certs/example.com.crt;
    ssl_certificate_key /etc/ssl/private/example.com.key;

    # SSL best practices
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    location / {
        proxy_pass http://chatbotgate;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name app.example.com;
    return 301 https://$server_name$request_uri;
}
```

#### Kubernetes Deployment

Example Kubernetes manifests:

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chatbotgate
spec:
  replicas: 2
  selector:
    matchLabels:
      app: chatbotgate
  template:
    metadata:
      labels:
        app: chatbotgate
    spec:
      containers:
      - name: chatbotgate
        image: ideamans/chatbotgate:v1.0.0
        ports:
        - containerPort: 4180
        volumeMounts:
        - name: config
          mountPath: /app/config
          readOnly: true
        resources:
          limits:
            cpu: "1"
            memory: "512Mi"
          requests:
            cpu: "500m"
            memory: "256Mi"
        livenessProbe:
          httpGet:
            path: /health
            port: 4180
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 4180
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: config
        configMap:
          name: chatbotgate-config
---
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: chatbotgate
spec:
  selector:
    app: chatbotgate
  ports:
  - port: 4180
    targetPort: 4180
  type: ClusterIP
```

#### Monitoring & Observability

1. **Health Endpoint**
   ```bash
   curl http://localhost:4180/health
   ```

2. **Structured Logs**
   - Use JSON format for log aggregators (Datadog, CloudWatch)
   - Enable `module_level: "debug"` for specific packages

3. **Metrics** (Future)
   - Prometheus metrics endpoint planned
   - Monitor session count, request latency, auth failures

4. **Alerts**
   - High error rate on authentication
   - Redis connection failures
   - Upstream unavailability

#### Scaling Considerations

1. **Horizontal Scaling**
   - ChatbotGate is stateless (when using Redis)
   - Scale to N replicas behind load balancer
   - Use Redis for shared session storage

2. **Session Affinity**
   - Not required when using Redis
   - Memory KVS requires sticky sessions

3. **Performance**
   - Each instance handles ~1000 req/s
   - Memory: ~50MB base + ~1KB per session
   - Redis: ~1KB per session

#### CI/CD Integration

ChatbotGate publishes Docker images automatically:

- **On Release Tag** (`v*`): Builds and publishes to Docker Hub
- **Multi-Arch**: AMD64 and ARM64 images
- **Versioning**: Semantic versioning (v1.0.0, v1.0, v1)
- **Latest**: Always points to latest stable release

Example CI/CD pipeline (GitHub Actions):

```yaml
- name: Deploy ChatbotGate
  run: |
    docker pull ideamans/chatbotgate:v1.0.0
    docker-compose up -d chatbotgate
    docker-compose exec chatbotgate chatbotgate version
```

## Advanced Topics

### Multi-Tenancy with Host-Based Routing

Route different hostnames to different upstreams:

```yaml
proxy:
  # Default upstream
  upstream:
    url: "http://localhost:8080"

  # Host-specific routes
  routes:
    - host: "app1.example.com"
      upstream: "http://localhost:8081"

    - host: "app2.example.com"
      upstream: "http://localhost:8082"
```

Each hostname can use the same authentication, but proxy to different backends.

### Custom OAuth2 Providers

ChatbotGate supports any OIDC-compliant provider:

```yaml
oauth2:
  providers:
    - name: "keycloak"
      type: "custom"
      display_name: "Keycloak"
      client_id: "chatbotgate"
      client_secret: "your-secret"
      auth_url: "https://keycloak.example.com/auth/realms/myrealm/protocol/openid-connect/auth"
      token_url: "https://keycloak.example.com/auth/realms/myrealm/protocol/openid-connect/token"
      userinfo_url: "https://keycloak.example.com/auth/realms/myrealm/protocol/openid-connect/userinfo"
      jwks_url: "https://keycloak.example.com/auth/realms/myrealm/protocol/openid-connect/certs"
```

**Auto-Discovery:**

Most OIDC providers have a `.well-known/openid-configuration` endpoint. Use it to find the URLs:

```bash
curl https://your-idp.com/.well-known/openid-configuration | jq .
```

### Rate Limiting

Configure rate limits to prevent abuse:

```yaml
ratelimit:
  enabled: true

  # Requests per minute per IP
  requests_per_minute: 60

  # Burst size
  burst: 10
```

### Custom Branding

Customize the authentication UI:

```yaml
service:
  name: "My Company Portal"
  description: "Secure access to company resources"
  icon_url: "https://cdn.example.com/icon-48.png"
  logo_url: "https://cdn.example.com/logo-200.png"
  logo_width: "180px"

oauth2:
  providers:
    - name: "google"
      display_name: "Company Google"
      icon_url: "https://cdn.example.com/google-icon.svg"
```

### Live Configuration Reloading

ChatbotGate watches `config.yaml` for changes and reloads automatically:

```bash
# Edit config
vim config.yaml

# Changes are applied automatically (logs will show reload)
```

**What Can Be Reloaded:**
- Service name and branding
- OAuth2 provider settings (add/remove/modify)
- Authorization rules
- Logging levels
- Access control rules

**What Cannot Be Reloaded (Requires Restart):**
- Server host/port
- Session cookie secret
- KVS backend type

### Security Best Practices

1. **Use Strong Secrets**
   ```bash
   # Generate cookie secret
   openssl rand -base64 32

   # Generate upstream secret
   openssl rand -hex 32
   ```

2. **Enable HTTPS**
   ```yaml
   session:
     cookie_secure: true
   ```

3. **Restrict Access**
   ```yaml
   authorization:
     allowed:
       - "@company.com"
   ```

4. **Protect Upstream**
   ```yaml
   proxy:
     upstream:
       secret:
         header: "X-Chatbotgate-Secret"
         value: "your-secret-here"
   ```

   Verify in upstream:
   ```javascript
   if (req.headers['x-chatbotgate-secret'] !== process.env.SECRET) {
     return res.status(403).send('Forbidden');
   }
   ```

5. **Use Redis in Production**
   ```yaml
   kvs:
     default:
       type: "redis"
   ```

6. **Monitor Logs**
   ```yaml
   logging:
     level: "info"  # or "warn" in production
   ```

## Troubleshooting

### OAuth2 Callback Error

**Problem:** "Invalid redirect URI" error from OAuth2 provider

**Solution:**
1. Check `callback_url` in config matches OAuth2 app settings
2. Ensure redirect URI includes protocol (https://)
3. Verify `auth_path_prefix` if using custom prefix

### Session Not Persisting

**Problem:** Users get logged out on every request

**Solution:**
1. Check cookie domain and path settings
2. Verify `cookie_secure` matches protocol (true for HTTPS)
3. Check browser cookie settings
4. Verify session storage is working (check logs)

### Email Not Sending

**Problem:** Magic link emails not arriving

**Solution:**

**SMTP:**
1. Verify SMTP credentials
2. Check spam folder
3. Test SMTP connection: `telnet smtp.gmail.com 587`
4. For Gmail, use App Password, not account password
5. Check logs for detailed error messages

**SendGrid:**
1. Verify API key has "Mail Send" permission
2. Check sender email is verified in SendGrid
3. Review SendGrid Activity Feed for delivery status

### Authorization Denied

**Problem:** User authenticated but gets "Access Denied"

**Solution:**
1. Check authorization whitelist includes user email/domain
2. Verify email from OAuth2 provider matches whitelist
3. Check logs for "user not authorized" message
4. Empty whitelist `[]` allows all authenticated users

### Upstream Connection Refused

**Problem:** "502 Bad Gateway" or "Connection refused"

**Solution:**
1. Verify upstream is running: `curl http://localhost:8080`
2. Check upstream URL in config
3. Verify firewall rules
4. Check Docker network if using containers

### High Memory Usage

**Problem:** ChatbotGate using too much memory

**Solution:**
1. Use Redis instead of memory KVS
2. Reduce session expiration time
3. Check for session leaks (memory KVS only)
4. Monitor with: `docker stats` or system tools

### CORS Errors

**Problem:** Browser CORS errors when accessing upstream

**Solution:**

ChatbotGate is a transparent proxy and preserves CORS headers. Configure CORS in your **upstream** application, not ChatbotGate.

Example (Express.js):
```javascript
const cors = require('cors');
app.use(cors({
  origin: 'https://your-domain.com',
  credentials: true
}));
```

### WebSocket Connection Failed

**Problem:** WebSocket connections not working

**Solution:**

ChatbotGate supports WebSocket proxying automatically. Ensure your reverse proxy (Nginx) has WebSocket support:

```nginx
location / {
    proxy_pass http://localhost:4180;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

### Configuration Syntax Error

**Problem:** "yaml: unmarshal error" on startup

**Solution:**
1. Validate YAML syntax: `yamllint config.yaml`
2. Check indentation (use spaces, not tabs)
3. Quote special characters in strings
4. Verify config against `config.example.yaml`

### Debug Mode

Enable debug logging for detailed diagnostics:

```yaml
logging:
  level: "debug"
  module_level: "debug"
```

Check logs for:
- OAuth2 token exchange details
- Session creation/validation
- Authorization decisions
- Proxy requests

### Getting Help

1. **Check Logs:** Set `level: "debug"` for detailed information
2. **Verify Config:** Compare with `config.example.yaml`
3. **Test Components:** Test OAuth2, email, upstream separately
4. **GitHub Issues:** [Report bugs or ask questions](https://github.com/ideamans/chatbotgate/issues)
5. **Discussions:** [Community support](https://github.com/ideamans/chatbotgate/discussions)

---

**Need more help?** Open an issue on GitHub with:
- ChatbotGate version
- Config file (redact secrets!)
- Log output (with debug enabled)
- Steps to reproduce the issue
