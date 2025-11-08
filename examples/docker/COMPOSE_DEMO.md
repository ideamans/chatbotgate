# ChatbotGate Docker Compose Demo

This is a complete 3-tier architecture demonstration of ChatbotGate.

## Architecture

```
┌─────────┐      ┌──────────────┐      ┌─────────┐
│ Browser │ ───▶ │    Nginx     │ ───▶ │ Backend │
│         │      │  (Frontend)  │      │   App   │
└─────────┘      └──────────────┘      └─────────┘
                        │
                        ▼
                 ┌──────────────┐
                 │ ChatbotGate  │
                 │ (Auth Proxy) │
                 └──────────────┘
```

### Components

1. **Nginx** - Frontend reverse proxy
   - Listens on port 80
   - Forwards all requests to ChatbotGate
   - Can be extended with TLS, static content, etc.

2. **ChatbotGate** (`ideamans/chatbotgate:latest`) - Authentication middleware
   - Email-based authentication (passwordless magic links)
   - Session management
   - Forwards authenticated requests to backend
   - Adds user info headers
   - Uses public Docker Hub image

3. **Backend** - Protected application
   - Simple HTML page showing user info
   - Receives authenticated user data via headers
   - Only accessible after authentication

## Quick Start

```bash
# Start all services
docker compose up -d

# View logs (all services)
docker compose logs -f

# View logs (specific service)
docker compose logs -f chatbotgate
docker compose logs -f nginx
docker compose logs -f backend

# Check service status
docker compose ps

# Stop all services
docker compose down

# Stop and remove volumes
docker compose down -v
```

## Testing

1. **Access the application:**
   ```bash
   open http://localhost
   ```

2. **You'll be redirected to the login page:**
   - Enter your email address
   - Click "Send Login Link"

3. **Check console output for the magic link:**
   ```bash
   docker compose logs chatbotgate | grep "verify?token="
   ```

4. **Copy the link and open it in your browser:**
   - The link format: `http://localhost/_auth/email/verify?token=...`

5. **After authentication:**
   - You'll be redirected to the backend application
   - The page displays your authenticated email and user info

## Email Configuration

By default, the demo uses `sendmail` which is included in the ChatbotGate Docker image.

**Note:** In this demo setup, emails won't actually be sent. Instead, check the logs for the magic link.

For production, configure real email sending:

### Option 1: SMTP

Edit `chatbotgate/config.yaml`:

```yaml
email_auth:
  enabled: true
  sender_type: "smtp"
  from: "ChatbotGate <noreply@example.com>"
  smtp:
    host: "smtp.gmail.com"
    port: 587
    username: "your@gmail.com"
    password: "your-app-password"
    starttls: true
```

### Option 2: SendGrid

Edit `chatbotgate/config.yaml`:

```yaml
email_auth:
  enabled: true
  sender_type: "sendgrid"
  from: "ChatbotGate <noreply@example.com>"
  sendgrid:
    api_key: "SG.xxxxxxxxxxxxx"
```

Restart services after configuration changes:

```bash
docker compose restart chatbotgate
```

## Adding OAuth2 Providers

To enable OAuth2 authentication (Google, GitHub, etc.), edit `chatbotgate/config.yaml`:

```yaml
oauth2:
  providers:
    - name: "google"
      client_id: "your-client-id.apps.googleusercontent.com"
      client_secret: "your-client-secret"
      enabled: true

    - name: "github"
      client_id: "your-github-client-id"
      client_secret: "your-github-client-secret"
      enabled: true
```

Restart services:

```bash
docker compose restart chatbotgate
```

## Customization

### Change Nginx Port

Edit `compose.yaml`:

```yaml
services:
  nginx:
    ports:
      - "8080:80"  # Access via http://localhost:8080
```

### Add Persistent Storage

For production, use persistent storage for session data:

Edit `compose.yaml`:

```yaml
services:
  chatbotgate:
    volumes:
      - ./chatbotgate/config.yaml:/etc/chatbotgate/config.yaml:ro
      - chatbotgate-data:/var/lib/chatbotgate/kvs  # Add this

volumes:
  chatbotgate-data:  # Add this
```

Edit `chatbotgate/config.yaml`:

```yaml
kvs:
  default:
    type: "leveldb"  # Change from "memory"
    leveldb:
      path: "/var/lib/chatbotgate/kvs"
```

### Add Redis

For multi-instance deployments:

Edit `compose.yaml`:

```yaml
services:
  chatbotgate:
    depends_on:
      - backend
      - redis  # Add this

  redis:  # Add this service
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data
    networks:
      - chatbotgate-network
    restart: unless-stopped

volumes:
  redis-data:  # Add this
```

Edit `chatbotgate/config.yaml`:

```yaml
kvs:
  default:
    type: "redis"
    redis:
      addr: "redis:6379"
      db: 0
```

## Troubleshooting

### Check Service Health

```bash
# All services
docker compose ps

# ChatbotGate health check
curl http://localhost/_auth/login

# Nginx
curl -I http://localhost
```

### View Detailed Logs

```bash
# All services with timestamps
docker compose logs -f -t

# Specific service with tail
docker compose logs --tail=100 -f chatbotgate
```

### Access Container Shell

```bash
# ChatbotGate
docker compose exec chatbotgate sh

# Backend
docker compose exec backend sh

# Nginx
docker compose exec nginx sh
```

### Rebuild Services

```bash
# Rebuild backend (only service with build config)
docker compose build backend

# Force rebuild and restart
docker compose up -d --build
```

### Use Local ChatbotGate Build (Developers)

If you're developing ChatbotGate and want to test local changes:

1. Edit `compose.yaml`:
   ```yaml
   chatbotgate:
     # Comment out this line:
     # image: ideamans/chatbotgate:latest
     # Uncomment these lines:
     build:
       context: ../..
       dockerfile: Dockerfile
   ```

2. Rebuild and restart:
   ```bash
   docker compose build chatbotgate
   docker compose up -d
   ```

### Clean Restart

```bash
# Stop and remove everything
docker compose down -v

# Remove images
docker compose down --rmi all

# Fresh start
docker compose up -d --build
```

## Production Considerations

For production deployments:

1. **Use HTTPS:**
   - Configure TLS in nginx
   - Set `session.cookie_secure: true`
   - Update `server.base_url` to HTTPS URL

2. **Use persistent storage:**
   - LevelDB volumes for single instance
   - Redis for multi-instance

3. **Configure real email:**
   - SMTP or SendGrid
   - Valid sender address

4. **Set strong secrets:**
   - Generate random `session.cookie_secret` (32+ chars)
   - Use environment variables or Docker secrets

5. **Add authorization:**
   - Whitelist email addresses or domains
   - Configure access control rules

6. **Resource limits:**
   - Set CPU and memory limits in compose.yaml
   - Monitor resource usage

7. **Logging:**
   - Configure log rotation
   - Use centralized logging (e.g., ELK, Loki)

## Example Production Setup

See the main [README.md](./README.md) for production deployment examples with:
- HTTPS/TLS configuration
- Redis for session storage
- Resource limits
- Security best practices
