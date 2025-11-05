# ChatbotGate

**ChatbotGate** is a lightweight, flexible authentication reverse proxy that sits in front of your upstream applications and provides unified authentication through multiple OAuth2 providers and passwordless email authentication.

## Features

### 🔐 Multiple Authentication Methods
- **OAuth2/OIDC**: Google, GitHub, Microsoft, and custom OIDC providers
- **Passwordless Email**: Magic link authentication via SMTP or SendGrid
- Mix and match providers for different user groups

### 🛡️ Flexible Access Control
- Email and domain-based whitelisting
- Path-based access rules (allow, auth, deny)
- Pattern matching (exact, prefix, regex, minimatch)
- First-match-wins rule evaluation

### 🔄 Seamless Reverse Proxy
- Transparent proxying of HTTP/WebSocket requests
- Configurable authentication path prefix (default: `/_auth`)
- Host-based routing for multi-tenant deployments
- Automatic upstream secret header injection

### 📦 Multiple Storage Backends
- **Memory**: Fast, ephemeral storage for development
- **LevelDB**: Persistent, embedded database
- **Redis**: Distributed, scalable storage for production
- Unified KVS interface with namespace isolation

### 🎨 User-Friendly Interface
- Clean, responsive authentication UI
- Multi-language support (English/Japanese)
- Theme switcher (Auto/Light/Dark)
- Customizable branding (logo, icon, colors)

### 🔌 User Information Forwarding
- Forward authenticated user data to upstream apps
- Flexible field mapping (email, username, provider, etc.)
- Encryption and compression support (AES-256-GCM, gzip)
- Query parameters and HTTP headers

### ⚙️ Production-Ready
- Live configuration reloading
- Structured logging with configurable levels
- Rate limiting support
- Comprehensive test coverage
- Docker support

## Quick Start

### Installation

**From Source:**
```bash
git clone https://github.com/ideamans/chatbotgate.git
cd chatbotgate
go build -o chatbotgate ./cmd/chatbotgate
```

**Using Docker:**
```bash
docker pull ideamans/chatbotgate:latest
```

### Basic Configuration

Create a `config.yaml` file:

```yaml
service:
  name: "My App Auth"

server:
  host: "0.0.0.0"
  port: 4180

proxy:
  upstream:
    url: "http://localhost:8080"

session:
  cookie_secret: "CHANGE-THIS-TO-A-RANDOM-SECRET"
  cookie_expire: "168h"

oauth2:
  providers:
    - name: "google"
      client_id: "YOUR-CLIENT-ID"
      client_secret: "YOUR-CLIENT-SECRET"

authorization:
  allowed:
    - "@example.com"  # Allow all @example.com emails
```

### Run the Server

```bash
./chatbotgate -config config.yaml
```

Visit `http://localhost:4180` to see the authentication flow in action.

## Documentation

- **[User Guide (GUIDE.md)](GUIDE.md)** - Complete guide for deploying and configuring ChatbotGate
- **[Module Guide (MODULE.md)](MODULE.md)** - Developer guide for using ChatbotGate as a Go module

## Project Structure

```
chatbotgate/
├── cmd/
│   └── chatbotgate/          # Main entry point and CLI
├── pkg/
│   ├── middleware/           # Authentication middleware
│   │   ├── auth/             # OAuth2 and email auth
│   │   ├── authz/            # Authorization (whitelisting)
│   │   ├── session/          # Session management
│   │   ├── rules/            # Path-based access control
│   │   ├── forwarding/       # User info forwarding
│   │   └── ...
│   ├── proxy/                # Reverse proxy
│   └── shared/               # Shared components
│       ├── kvs/              # Key-value store interface
│       ├── i18n/             # Internationalization
│       └── logging/          # Structured logging
├── web/                      # Web assets (HTML, CSS, JS)
├── email/                    # Email templates
├── e2e/                      # End-to-end tests
├── config.example.yaml       # Example configuration
└── README.md                 # This file
```

## How It Works

```
┌─────────┐      ┌──────────────┐      ┌──────────┐
│  User   │─────▶│ ChatbotGate  │─────▶│ Upstream │
│ Browser │      │    (Auth)    │      │   App    │
└─────────┘      └──────────────┘      └──────────┘
     ▲                  │
     │                  ▼
     │           ┌─────────────┐
     └───────────│   OAuth2    │
                 │  Provider   │
                 └─────────────┘
```

1. **User Request** → ChatbotGate checks authentication
2. **Not Authenticated** → Redirect to `/_auth/login`
3. **User Chooses** → OAuth2 or Email authentication
4. **Authentication Success** → Session created, redirect to original URL
5. **Authenticated Request** → Proxied to upstream with user info headers

## Development

### Prerequisites

- Go 1.21 or later
- Redis (optional, for distributed sessions)

### Build

```bash
# Build the binary
go build -o chatbotgate ./cmd/chatbotgate

# Run tests
go test ./...

# Run with coverage
go test -cover ./...
```

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./pkg/middleware/auth/oauth2

# With verbose output
go test -v ./pkg/...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Docker Build

```bash
# Build image
docker build -t chatbotgate .

# Run with docker-compose
docker-compose up
```

## Configuration

See `config.example.yaml` for a complete configuration example with detailed comments.

Key configuration sections:
- **Service**: Service name and branding
- **Server**: Host, port, and authentication path prefix
- **Proxy**: Upstream URL and routing rules
- **Session**: Cookie settings and expiration
- **OAuth2**: Provider configurations
- **Email Auth**: SMTP or SendGrid setup
- **Authorization**: Email/domain whitelisting
- **KVS**: Storage backend (memory/leveldb/redis)
- **Forwarding**: User information to upstream
- **Rules**: Path-based access control
- **Logging**: Log levels and output

## Use Cases

### Chatbot Widget Authentication
Protect chatbot interfaces (Dify, Rasa, etc.) with OAuth2 or email authentication.

### Internal Tool Access Control
Add authentication to internal tools that lack their own auth system.

### Multi-Tenant Applications
Route requests to different upstream backends based on hostname.

### API Gateway with Auth
Combine reverse proxy and authentication for microservices.

## Security Considerations

- **Cookie Secret**: Use a strong random secret (32+ characters)
- **HTTPS**: Always use HTTPS in production (set `cookie_secure: true`)
- **Secrets Storage**: Use environment variables or secret managers for sensitive data
- **Upstream Secret**: Protect your upstream from direct access with secret headers
- **Whitelisting**: Restrict access by email/domain when possible
- **Rate Limiting**: Configure rate limits to prevent abuse

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and idioms
- Write tests for new features
- Update documentation for user-facing changes
- Run `go fmt` and `go vet` before committing
- Aim for 80%+ test coverage

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/ideamans/chatbotgate/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ideamans/chatbotgate/discussions)

## Acknowledgments

- Inspired by [oauth2-proxy](https://oauth2-proxy.github.io/oauth2-proxy/)
- Built with [Go](https://golang.org/), [fsnotify](https://github.com/fsnotify/fsnotify), and [many other great libraries](go.mod)

---

**Made with ❤️ for the authentication-challenged web**
