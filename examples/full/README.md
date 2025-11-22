# Full Configuration Example

This directory contains a comprehensive example configuration (`config.yaml`) that demonstrates nearly all available ChatbotGate features and options.

## Overview

The `config.yaml` file showcases:

- **Multiple OAuth2 providers**: Google, GitHub, Microsoft, and custom OIDC
- **Email authentication**: SMTP, SendGrid, and Sendmail options
- **User information forwarding**: Various field forwarding with encryption and compression
- **Access control rules**: Path-based allow/auth/deny rules with multiple matcher types
- **KVS storage options**: Memory, LevelDB, and Redis with namespace isolation
- **Logging**: Console and file logging with rotation
- **Production-ready settings**: HTTPS, secure cookies, Redis sessions

## Important Notes

### Mutually Exclusive Options

Some configuration options cannot be used simultaneously. In the example config, incompatible options are commented out with explanations:

1. **Email Sender Type** (only one can be active):
   - `sender_type: "smtp"` ← Active in example
   - `sender_type: "sendgrid"` ← Commented out
   - `sender_type: "sendmail"` ← Commented out

2. **KVS Default Type** (only one can be active):
   - `type: "memory"` ← Commented out
   - `type: "leveldb"` ← Commented out
   - `type: "redis"` ← Active in example

3. **KVS Override vs Shared** (cannot override and share simultaneously):
   - Using shared default with namespaces ← Active in example
   - Session override with dedicated backend ← Commented out
   - Token override with dedicated backend ← Commented out
   - RateLimit override with dedicated backend ← Commented out

## Before Using This Config

### 1. Generate Secrets

Replace placeholder secrets with secure random values:

```bash
# Generate cookie secret (at least 32 characters)
openssl rand -base64 32

# Generate encryption key (at least 32 characters)
openssl rand -base64 32
```

Update these fields in `config.yaml`:
- `session.cookie_secret`
- `proxy.upstream.secret.value`
- `forwarding.encryption.key`

### 2. Configure OAuth2 Providers

For each OAuth2 provider you want to use:

1. Register your application with the provider
2. Set the callback URL: `{base_url}/_auth/oauth2/callback`
   - Example: `https://auth.example.com/_auth/oauth2/callback`
3. Update `client_id` and `client_secret` in the config
4. Set `disabled: false` to enable the provider

**Provider Setup Guides:**
- [Google OAuth2](https://console.cloud.google.com/apis/credentials)
- [GitHub OAuth Apps](https://github.com/settings/developers)
- [Microsoft Azure AD](https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps)

### 3. Configure Email Authentication

Choose one sender type and update credentials:

**SMTP** (Gmail example):
```yaml
email_auth:
  sender_type: "smtp"
  smtp:
    host: "smtp.gmail.com"
    port: 587
    username: "your-email@gmail.com"
    password: "your-app-password"  # Generate at https://myaccount.google.com/apppasswords
```

**SendGrid**:
```yaml
email_auth:
  sender_type: "sendgrid"
  sendgrid:
    api_key: "SG.xxxxxxxxxxxxxxxxxxxx"  # From https://app.sendgrid.com/settings/api_keys
```

**Sendmail** (local MTA):
```yaml
email_auth:
  sender_type: "sendmail"
  sendmail:
    path: "/usr/sbin/sendmail"
```

### 4. Configure Redis (if using)

Update Redis connection settings:

```yaml
kvs:
  default:
    type: "redis"
    redis:
      addr: "redis.example.com:6379"
      password: "your-redis-password"
      db: 0
```

**Test Redis connection:**
```bash
redis-cli -h redis.example.com -p 6379 -a your-redis-password ping
```

### 5. Update URLs and Domains

Replace example domains with your actual URLs:

- `server.base_url`: Your public URL (e.g., `https://auth.example.com`)
- `proxy.upstream.url`: Your upstream application URL
- `service.icon_url` / `service.logo_url`: Your branding assets
- `authorization.allowed`: Allowed email addresses and domains

### 6. Customize Access Rules

Review and modify the access control rules to match your application's paths:

```yaml
rules:
  rules:
    - prefix: "/static/"
      action: allow
    - prefix: "/admin/"
      action: deny
    - all: true
      action: auth
```

## Validation

Test your configuration before deployment:

```bash
# From project root
./chatbotgate test-config --config examples/full/config.yaml
```

Expected output:
```
✓ Configuration file loaded successfully
✓ Configuration validation passed
```

## Running with This Config

### Development

```bash
./chatbotgate serve --config examples/full/config.yaml
```

### Production (systemd)

1. Copy config to production location:
   ```bash
   sudo mkdir -p /etc/chatbotgate
   sudo cp examples/full/config.yaml /etc/chatbotgate/config.yaml
   sudo chown root:root /etc/chatbotgate/config.yaml
   sudo chmod 600 /etc/chatbotgate/config.yaml  # Protect secrets
   ```

2. Update systemd service to use the config:
   ```bash
   sudo systemctl edit chatbotgate
   ```

   Add:
   ```ini
   [Service]
   ExecStart=/usr/local/bin/chatbotgate serve --config /etc/chatbotgate/config.yaml
   ```

3. Restart service:
   ```bash
   sudo systemctl restart chatbotgate
   ```

### Docker

```bash
docker run -d \
  --name chatbotgate \
  -p 4180:4180 \
  -v $(pwd)/examples/full/config.yaml:/app/config/config.yaml:ro \
  ideamans/chatbotgate:latest \
  serve --config /app/config/config.yaml
```

### Docker Compose

```yaml
version: '3.8'

services:
  chatbotgate:
    image: ideamans/chatbotgate:latest
    ports:
      - "4180:4180"
    volumes:
      - ./examples/full/config.yaml:/app/config/config.yaml:ro
    command: serve --config /app/config/config.yaml
    environment:
      - TZ=Asia/Tokyo
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    command: redis-server --requirepass your-redis-password
    restart: unless-stopped

volumes:
  redis-data:
```

## Configuration Sections

### Service Branding
- `service.name`: Application name shown in UI
- `service.icon_url`: Small icon (48px) for auth header
- `service.logo_url`: Larger logo image for auth header
- `service.logo_width`: Logo width (default: "200px")

### Server Settings
- `server.host`: Bind address (default: "0.0.0.0")
- `server.port`: Listen port (default: 4180)
- `server.auth_path_prefix`: Auth endpoint prefix (default: "/_auth")
- `server.base_url`: Public URL for callbacks and emails

### Session Management
- `session.cookie_name`: Cookie name (default: "_oauth2_proxy")
- `session.cookie_secret`: Encryption secret (required, 32+ chars)
- `session.cookie_expire`: Session duration (e.g., "168h" = 7 days)
- `session.cookie_secure`: HTTPS-only (true for production)
- `session.cookie_samesite`: "strict", "lax", or "none"

### OAuth2 Providers
Each provider supports:
- `name`: Provider identifier (unique)
- `display_name`: Display name in UI
- `client_id` / `client_secret`: OAuth2 credentials
- `disabled`: Hide from login page (default: false)
- `icon_url`: Custom icon URL (optional)
- `scopes`: OAuth2 scopes (provider-specific defaults if empty)

### Email Authentication
- `enabled`: Enable email auth (default: false)
- `sender_type`: "smtp", "sendgrid", or "sendmail"
- `from`: Sender address (RFC 5322 format supported)
- `token.expire`: Magic link expiration (default: "15m")

### Authorization
- `allowed`: Email addresses or domains (prefix with `@` for domains)
- Empty array `[]` allows all authenticated users

### Forwarding
Forward user info to upstream apps:
- `path`: Field path (e.g., "email", "extra._avatar_url", ".")
- `query`: Query parameter name (login redirect)
- `header`: HTTP header name (all requests)
- `filters`: "encrypt", "zip", "base64"

Standard fields: `email`, `username`, `provider`, `_email`, `_username`, `_avatar_url`

### Access Rules
- `exact`: Exact path match
- `prefix`: Path prefix match
- `regex`: Regular expression match
- `minimatch`: Glob pattern (supports `**/*.js`)
- `all`: true (catch-all rule)

Actions: `allow` (no auth), `auth` (require auth), `deny` (403)

### KVS Storage
- `default.type`: "memory", "leveldb", or "redis"
- `namespaces`: Logical isolation (session, token, ratelimit)
- Override per use case with `session`, `token`, `ratelimit` keys

### Logging
- `level`: "debug", "info", "warn", "error"
- `color`: Enable ANSI colors (false for systemd)
- `file`: Optional file logging with rotation

### Assets
- `optimization.dify`: Load dify.css for iframe optimizations

## Troubleshooting

### Validation Errors

**Error: "cookie_secret is required"**
- Set `session.cookie_secret` (at least 32 characters)

**Error: "no authentication method enabled"**
- Enable at least one OAuth2 provider or email auth

**Error: "encryption config required"**
- Set `forwarding.encryption.key` when using "encrypt" filter

**Error: "invalid regex pattern"**
- Fix regex syntax in `rules.rules[].regex`

### Runtime Issues

**OAuth2 callback fails:**
- Verify `server.base_url` matches your public URL
- Check OAuth2 provider callback URL configuration
- Ensure `{base_url}/_auth/oauth2/callback` is whitelisted

**Email not sent:**
- Check SMTP credentials and port (587 for STARTTLS, 465 for TLS)
- Verify SendGrid API key and endpoint
- Test sendmail with `echo "test" | sendmail -v user@example.com`

**Redis connection failed:**
- Verify `kvs.default.redis.addr` and credentials
- Check network connectivity and firewall rules
- Test with `redis-cli -h HOST -p PORT -a PASSWORD ping`

**Session not persisted:**
- Check KVS backend is running (Redis/LevelDB)
- Verify `session.cookie_secure` matches your protocol (HTTP/HTTPS)
- Review browser cookie settings and SameSite policy

## Production Checklist

- [ ] All secrets replaced with secure random values
- [ ] OAuth2 providers configured with production credentials
- [ ] Email sender configured and tested
- [ ] Redis configured for session storage
- [ ] `server.base_url` set to production URL
- [ ] `session.cookie_secure: true` for HTTPS
- [ ] `authorization.allowed` configured with allowed users/domains
- [ ] Access rules customized for your application
- [ ] Configuration file permissions set to 600 (read-only by owner)
- [ ] Logging configured (systemd journal or file rotation)
- [ ] Validated with `chatbotgate test-config`

## References

- [ChatbotGate README](../../README.md)
- [Deployment Guide](../../GUIDE.md)
- [Configuration Reference](../../config.example.yaml)
- [Docker Examples](../docker/)
- [Systemd Examples](../systemd/)
