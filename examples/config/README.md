# Configuration Examples

This directory contains example configuration files for different use cases.

## service.yaml - Simple Service Configuration

Minimal configuration for quick start with email authentication only.

**Features:**
- Email authentication via sendmail (uses system MTA)
- No OAuth2 providers
- No authorization whitelist (all authenticated users allowed)
- Memory-based storage (no external dependencies)
- Simple setup for development or small deployments

**Prerequisites:**

1. Working sendmail command on your system:
   ```bash
   # Check if sendmail is available
   which sendmail

   # Test sendmail
   echo "Test email" | sendmail -v your@email.com
   ```

2. Configure your upstream service URL in the config:
   ```yaml
   proxy:
     upstream:
       url: "http://localhost:8080/"  # Replace with your service
   ```

3. Generate a secure cookie secret:
   ```bash
   # Generate random 32-character string
   openssl rand -base64 32
   ```

**Usage:**

```bash
# Run with this config
chatbotgate -config examples/config/service.yaml

# Or copy and customize
cp examples/config/service.yaml config.yaml
# Edit config.yaml with your settings
chatbotgate -config config.yaml
```

**Docker Usage:**

The example configuration is included in the Docker image:

```bash
# Test the example config
docker run --rm ideamans/chatbotgate:latest \
  test-config -c /app/examples/config/service.yaml

# Copy example to customize
docker run --rm ideamans/chatbotgate:latest \
  cat /app/examples/config/service.yaml > config.yaml

# Edit config.yaml, then run with your config
docker run -d \
  -p 4180:4180 \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  ideamans/chatbotgate:latest \
  serve -c /app/config/config.yaml
```

**Important Settings to Change:**

1. **cookie_secret**: Generate a random secret (minimum 32 characters)
2. **proxy.upstream.url**: Your upstream application URL
3. **email_auth.sendmail.from**: Your sender email address
4. **cookie_secure**: Set to `true` when using HTTPS in production

**Testing:**

1. Start your upstream application on port 8080 (or change the port in config)
2. Run ChatbotGate: `chatbotgate -config examples/config/service.yaml`
3. Access http://localhost:4180
4. Enter your email address to receive a login link
5. Click the link in the email to authenticate

**Production Considerations:**

For production use, consider:
- Using HTTPS and setting `cookie_secure: true`
- Using Redis for session storage instead of memory
- Adding authorization whitelist to restrict access
- Configuring SMTP or SendGrid for reliable email delivery
- Setting appropriate log level (`info` or `warn`)

**Example with HTTPS:**

```yaml
server:
  base_url: "https://gateway.example.com"

session:
  cookie_secure: true
```

Run behind a reverse proxy (nginx, Caddy) that handles TLS termination.
