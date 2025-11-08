# Docker Deployment Examples

This directory contains example configuration files for running ChatbotGate in Docker containers.

## Files

- **`config.yaml`** - Example configuration optimized for Docker (standalone usage)
- **`compose.yaml`** - Docker Compose setup with 3-tier architecture demo
- **`chatbotgate/`** - ChatbotGate configuration directory
  - **`config.yaml`** - Configuration for Docker Compose demo
- **`nginx/`** - Nginx configuration files for frontend proxy
- **`backend/`** - Simple backend application for demonstration

## Quick Start - 3-Tier Demo with Docker Compose

The easiest way to try ChatbotGate is using the included Docker Compose setup, which demonstrates a complete 3-tier architecture:

```
Browser → Nginx (Frontend) → ChatbotGate (Auth Proxy) → Backend App
```

**Start the demo:**

```bash
# Navigate to examples/docker directory
cd examples/docker

# Start all services
docker compose up -d

# View logs
docker compose logs -f

# Access the application
open http://localhost
```

The demo will:
1. Start nginx on port 80 as the frontend
2. Start ChatbotGate as the authentication proxy
3. Start a simple backend application
4. Require email authentication to access the backend

**Stop the demo:**

```bash
docker compose down
```

**Architecture:**

- **nginx** (`nginx:alpine`) - Frontend reverse proxy on port 80
  - Forwards all requests to ChatbotGate
  - Handles static content serving
  - Can be configured for TLS termination

- **ChatbotGate** (built from source) - Authentication middleware
  - Email-based authentication (no OAuth2 in demo)
  - Session management with in-memory store
  - Forwards authenticated user info to backend

- **Backend** (`nginx:alpine` with custom HTML) - Protected application
  - Displays authenticated user information
  - Receives user data via HTTP headers
  - Simple demonstration of a protected resource

## Quick Start - Standalone Container

### 1. Pull Docker Image

```bash
# Pull latest version
docker pull ideamans/chatbotgate:latest

# Or pull specific version
docker pull ideamans/chatbotgate:v1.0.0
```

### 2. Prepare Configuration

```bash
# Copy example config
docker run --rm ideamans/chatbotgate:latest \
  cat /app/examples/docker/config.yaml > config.yaml

# Edit configuration
nano config.yaml
```

**Important settings to change:**
1. `session.cookie_secret` - Generate random 32+ character string
2. `proxy.upstream.url` - Your upstream application URL
3. `email_auth.from` - Your sender email address

**For production with HTTPS:**
1. `server.base_url` - Your public HTTPS URL (e.g., `https://auth.example.com`)
2. `session.cookie_secure` - Set to `true`

### 3. Run Container

**Basic usage:**
```bash
docker run -d \
  --name chatbotgate \
  -p 4180:4180 \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  ideamans/chatbotgate:latest
```

**With data persistence:**
```bash
# Create data volume
docker volume create chatbotgate-data

docker run -d \
  --name chatbotgate \
  -p 4180:4180 \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  -v chatbotgate-data:/var/lib/chatbotgate \
  ideamans/chatbotgate:latest
```

**With network bridge (recommended):**
```bash
# Create network
docker network create chatbotgate-net

# Run upstream app
docker run -d \
  --name myapp \
  --network chatbotgate-net \
  myapp:latest

# Run chatbotgate
docker run -d \
  --name chatbotgate \
  --network chatbotgate-net \
  -p 4180:4180 \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  ideamans/chatbotgate:latest
```

Update `config.yaml`:
```yaml
proxy:
  upstream:
    url: "http://myapp:8080/"  # Use container name as hostname
```

### 4. Verify Deployment

```bash
# Check logs
docker logs -f chatbotgate

# Check health
curl http://localhost:4180/health

# Access application
open http://localhost:4180
```

## Email Configuration

The Docker image includes `ssmtp` (lightweight sendmail replacement) for email authentication.

### Option 1: Use External SMTP (Recommended)

Configure SMTP in `config.yaml`:

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

### Option 2: Use SendGrid

```yaml
email_auth:
  enabled: true
  sender_type: "sendgrid"
  from: "ChatbotGate <noreply@example.com>"
  sendgrid:
    api_key: "SG.xxxxxxxxxxxxx"
```

### Option 3: Configure ssmtp (Advanced)

Create `ssmtp.conf`:
```ini
root=noreply@example.com
mailhub=smtp.gmail.com:587
AuthUser=your@gmail.com
AuthPass=your-app-password
UseSTARTTLS=YES
```

Mount it in the container:
```bash
docker run -d \
  --name chatbotgate \
  -p 4180:4180 \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  -v $(pwd)/ssmtp.conf:/etc/ssmtp/ssmtp.conf:ro \
  ideamans/chatbotgate:latest
```

## Production Deployment

### Behind Reverse Proxy (Recommended)

Run ChatbotGate behind nginx, Caddy, or Traefik for TLS termination.

**nginx example:**
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
    }
}
```

**config.yaml:**
```yaml
server:
  base_url: "https://auth.example.com"

session:
  cookie_secure: true
```

### With Docker Compose

Create `docker-compose.yml`:
```yaml
version: '3.8'

services:
  chatbotgate:
    image: ideamans/chatbotgate:latest
    container_name: chatbotgate
    ports:
      - "4180:4180"
    volumes:
      - ./config.yaml:/app/config/config.yaml:ro
      - chatbotgate-data:/var/lib/chatbotgate
    environment:
      - TZ=Asia/Tokyo
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:4180/health"]
      interval: 30s
      timeout: 3s
      retries: 3

  # Your upstream application
  myapp:
    image: myapp:latest
    container_name: myapp
    # ... your app config ...

volumes:
  chatbotgate-data:
```

Start services:
```bash
docker-compose up -d
```

### With Redis (Recommended for Production)

```yaml
version: '3.8'

services:
  chatbotgate:
    image: ideamans/chatbotgate:latest
    depends_on:
      - redis
    ports:
      - "4180:4180"
    volumes:
      - ./config.yaml:/app/config/config.yaml:ro
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data
    restart: unless-stopped

volumes:
  redis-data:
```

Update `config.yaml`:
```yaml
kvs:
  default:
    type: "redis"
    redis:
      addr: "redis:6379"
      db: 0
```

## Logging

View logs with Docker:

```bash
# Follow logs
docker logs -f chatbotgate

# View last 100 lines
docker logs --tail 100 chatbotgate

# View logs with timestamps
docker logs -t chatbotgate

# View logs since specific time
docker logs --since 2024-01-01T00:00:00 chatbotgate
docker logs --since 1h chatbotgate
```

### Log to File

If you need file-based logs, configure in `config.yaml`:

```yaml
logging:
  file:
    path: "/var/log/chatbotgate/chatbotgate.log"
    max_size_mb: 100
    max_backups: 3
    max_age: 28
    compress: true
```

Mount log directory:
```bash
docker run -d \
  --name chatbotgate \
  -p 4180:4180 \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  -v $(pwd)/logs:/var/log/chatbotgate \
  ideamans/chatbotgate:latest
```

## Troubleshooting

### Check Container Status

```bash
# Container status
docker ps -a

# Inspect container
docker inspect chatbotgate

# Container resource usage
docker stats chatbotgate
```

### Debug Mode

Run with debug logging:

```yaml
logging:
  level: "debug"
```

Or override at runtime:
```bash
docker run -d \
  --name chatbotgate \
  -e CHATBOTGATE_LOG_LEVEL=debug \
  -p 4180:4180 \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  ideamans/chatbotgate:latest
```

### Test Configuration

```bash
# Test config without starting server
docker run --rm \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  ideamans/chatbotgate:latest \
  test-config -c /app/config/config.yaml
```

### Access Container Shell

```bash
# Start shell in running container
docker exec -it chatbotgate sh

# Check sendmail
docker exec chatbotgate which sendmail
docker exec chatbotgate ls -l /usr/sbin/sendmail
```

## Security

### Best Practices

1. **Use secrets management:**
   ```bash
   # Docker secrets (Swarm mode)
   echo "your-secret" | docker secret create cookie_secret -
   ```

2. **Run with read-only filesystem:**
   ```bash
   docker run -d \
     --name chatbotgate \
     --read-only \
     --tmpfs /tmp \
     -v chatbotgate-data:/var/lib/chatbotgate \
     -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
     ideamans/chatbotgate:latest
   ```

3. **Limit resources:**
   ```bash
   docker run -d \
     --name chatbotgate \
     --memory="512m" \
     --cpus="0.5" \
     ideamans/chatbotgate:latest
   ```

4. **Use specific version tags:**
   ```bash
   # Good
   docker pull ideamans/chatbotgate:v1.0.0

   # Avoid in production
   docker pull ideamans/chatbotgate:latest
   ```

### Network Security

```bash
# Expose only to localhost
docker run -d \
  --name chatbotgate \
  -p 127.0.0.1:4180:4180 \
  ideamans/chatbotgate:latest

# Use Docker network for inter-container communication
docker network create --driver bridge chatbotgate-net
```

## Updating

```bash
# Pull new version
docker pull ideamans/chatbotgate:v1.1.0

# Stop old container
docker stop chatbotgate
docker rm chatbotgate

# Start new container
docker run -d \
  --name chatbotgate \
  -p 4180:4180 \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  -v chatbotgate-data:/var/lib/chatbotgate \
  ideamans/chatbotgate:v1.1.0

# Verify
docker logs -f chatbotgate
```

## References

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Best practices for writing Dockerfiles](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
