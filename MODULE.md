# ChatbotGate Module Guide

Developer guide for using ChatbotGate as a Go module and extending its functionality.

## Table of Contents

- [Introduction](#introduction)
- [Installation](#installation)
- [Core Concepts](#core-concepts)
- [Architecture](#architecture)
- [Programming Interfaces](#programming-interfaces)
- [Examples](#examples)
- [Testing](#testing)
- [API Reference](#api-reference)

## Introduction

ChatbotGate is designed as a modular, extensible authentication reverse proxy. While it works as a standalone binary, you can also use it as a Go module to:

- Build custom authentication proxies
- Integrate authentication middleware into existing Go applications
- Create custom OAuth2 providers
- Implement custom session storage backends
- Extend functionality with custom middleware

## Installation

### As a Go Module

```bash
go get github.com/ideamans/chatbotgate
```

### Module Structure

```
github.com/ideamans/chatbotgate/
├── cmd/chatbotgate/          # CLI application
├── pkg/
│   ├── middleware/           # Authentication middleware
│   │   ├── auth/             # Auth providers
│   │   │   ├── oauth2/       # OAuth2 providers
│   │   │   └── email/        # Email authentication
│   │   ├── authz/            # Authorization
│   │   ├── session/          # Session management
│   │   ├── rules/            # Access control rules
│   │   ├── forwarding/       # User info forwarding
│   │   ├── config/           # Configuration
│   │   ├── core/             # Core middleware logic
│   │   └── factory/          # Middleware factory
│   ├── proxy/                # Reverse proxy
│   │   ├── core/             # Proxy implementation
│   │   └── config/           # Proxy configuration
│   └── shared/               # Shared components
│       ├── kvs/              # Key-Value Store interface
│       ├── i18n/             # Internationalization
│       ├── logging/          # Structured logging
│       ├── config/           # Config utilities
│       ├── filewatcher/      # File watching
│       └── factory/          # Shared factory
```

## Core Concepts

### 1. Middleware Architecture

ChatbotGate uses a layered middleware architecture:

```
Request → Auth Check → Authorization → Rules → Forwarding → Proxy → Upstream
```

Each layer is implemented as Go middleware (`func(http.Handler) http.Handler`).

### 2. Provider Pattern

Authentication providers (OAuth2, email, password) implement common interfaces, allowing easy extension.

### 3. KVS Abstraction

Session storage, token storage, and rate limiting all use a unified Key-Value Store interface, supporting multiple backends (memory, LevelDB, Redis).

### 4. Configuration-Driven

All components are configured via YAML, with live reload support.

## Architecture

### Middleware Package

The `pkg/middleware` package contains all authentication and authorization logic:

#### Core Middleware (`pkg/middleware/core`)

The heart of the authentication system:

```go
import (
    "github.com/ideamans/chatbotgate/pkg/middleware/config"
    "github.com/ideamans/chatbotgate/pkg/middleware/factory"
    "github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// Create configuration
cfg := &config.Config{
    Service: config.ServiceConfig{
        Name: "My App",
    },
    Session: config.SessionConfig{
        Cookie: config.CookieConfig{
            Secret: "your-secret-32-characters-long",
            Expire: "168h",
        },
    },
    // ... other config
}

// Create logger
logger := logging.New(logging.Config{Level: "info"})

// Create factory and KVS stores
f := factory.NewFactory()
sessionStore, tokenStore, rateLimitStore, err := f.CreateKVSStores(cfg)
if err != nil {
    log.Fatal(err)
}

// Create middleware using factory
mw, err := f.CreateMiddleware(cfg, sessionStore, tokenStore, rateLimitStore, upstreamHandler, logger)
if err != nil {
    log.Fatal(err)
}

// Wrap upstream handler with authentication
handler := mw.Wrap(upstreamHandler)
http.ListenAndServe(":8080", handler)
```

**Key Types:**

- `Middleware`: Main middleware struct
- `Config`: Configuration struct
- `UserInfo`: Authenticated user information

#### OAuth2 Providers (`pkg/middleware/auth/oauth2`)

OAuth2 provider implementations:

```go
import (
    "context"
    "github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
    goauth2 "golang.org/x/oauth2"
)

// Provider interface
type Provider interface {
    // Name returns the provider identifier (e.g., "google", "github")
    Name() string

    // Config returns the OAuth2 configuration
    Config() *goauth2.Config

    // GetUserInfo retrieves user information using the OAuth2 token
    GetUserInfo(ctx context.Context, token *goauth2.Token) (*oauth2.UserInfo, error)
}

// UserInfo represents authenticated user information
type UserInfo struct {
    Email string                 // User's email address
    Name  string                 // User's display name (optional)
    Extra map[string]interface{} // Additional provider-specific data
}
```

**Built-in Providers:**
- `GoogleProvider`: Google OAuth2
- `GitHubProvider`: GitHub OAuth2
- `MicrosoftProvider`: Microsoft/Azure AD OAuth2
- `CustomProvider`: Generic OIDC provider

**Default Scopes:**

Each provider has default scopes that are used when `scopes` configuration is empty. These defaults are designed to retrieve user email, name, and avatar (where supported).

- **Google**: `openid`, `userinfo.email`, `userinfo.profile`
- **GitHub**: `user:email`, `read:user`
- **Microsoft**: `openid`, `profile`, `email`, `User.Read`
- **Custom**: `openid`, `email`, `profile`

**Note**: If you specify custom scopes in configuration, the defaults are NOT added automatically. You must explicitly include the default scopes if you want user information.

**Standardized Fields:**

All OAuth2 providers populate standardized fields in `UserInfo.Extra` for consistent access:

- `_email`: User email address (string)
- `_username`: User display name (string, GitHub fallback: name → login)
- `_avatar_url`: User profile picture URL (string, empty for Microsoft and custom providers that don't support it)

**Creating a Custom Provider:**

```go
import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
    goauth2 "golang.org/x/oauth2"
)

type MyProvider struct {
    name        string
    config      *goauth2.Config
    userinfoURL string
}

func NewMyProvider(clientID, clientSecret, redirectURL string) *MyProvider {
    return &MyProvider{
        name: "myprovider",
        config: &goauth2.Config{
            ClientID:     clientID,
            ClientSecret: clientSecret,
            RedirectURL:  redirectURL,
            Scopes:       []string{"openid", "email", "profile"},
            Endpoint: goauth2.Endpoint{
                AuthURL:  "https://myprovider.com/oauth/authorize",
                TokenURL: "https://myprovider.com/oauth/token",
            },
        },
        userinfoURL: "https://myprovider.com/oauth/userinfo",
    }
}

func (p *MyProvider) Name() string {
    return p.name
}

func (p *MyProvider) Config() *goauth2.Config {
    return p.config
}

func (p *MyProvider) GetUserInfo(ctx context.Context, token *goauth2.Token) (*oauth2.UserInfo, error) {
    client := p.config.Client(ctx, token)
    resp, err := client.Get(p.userinfoURL)
    if err != nil {
        return nil, fmt.Errorf("fetch userinfo: %w", err)
    }
    defer resp.Body.Close()

    var data struct {
        Email string `json:"email"`
        Name  string `json:"name"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return nil, fmt.Errorf("decode userinfo: %w", err)
    }

    return &oauth2.UserInfo{
        Email: data.Email,
        Name:  data.Name,
        Extra: map[string]interface{}{
            "_email":    data.Email,
            "_username": data.Name,
        },
    }, nil
}
```

#### Email Authentication (`pkg/middleware/auth/email`)

Passwordless email authentication:

```go
import (
    "time"
    "github.com/ideamans/chatbotgate/pkg/middleware/auth/email"
    "github.com/ideamans/chatbotgate/pkg/shared/kvs"
)

// Email sender interface
type Sender interface {
    // Send sends a plain text email
    Send(to, subject, body string) error

    // SendHTML sends an email with both HTML and text versions
    SendHTML(to, subject, htmlBody, textBody string) error
}

// Token store for managing OTP and magic link tokens
type TokenStore struct {
    store kvs.Store
    ttl   time.Duration
}

// Key methods:
// - GenerateToken(email string) (string, error): Generate magic link token
// - VerifyToken(token string) (string, error): Verify magic link token, returns email
// - GenerateOTP(email string) (string, error): Generate 6-digit OTP
// - VerifyOTP(email, otp string) error: Verify OTP
```

**Built-in Senders:**
- `NewSMTPSender(config)`: SMTP-based email sending
- `NewSendGridSender(config)`: SendGrid API
- `NewSendmailSender(config)`: Local sendmail command (uses system MTA)

**Standardized Fields:**

Email authentication populates the same standardized fields as OAuth2 providers for consistent access in forwarding:

- `_email`: User email address
- `_username`: Email local part (before @)
- `_avatar_url`: Empty string (no avatar for email auth)
- `userpart`: Email local part (before @, same as `_username`)

#### Password Authentication (`pkg/middleware/auth/password`)

Simple password-based authentication for testing and development:

```go
import (
    "github.com/ideamans/chatbotgate/pkg/middleware/auth/password"
    "github.com/ideamans/chatbotgate/pkg/middleware/config"
    "github.com/ideamans/chatbotgate/pkg/shared/kvs"
    "github.com/ideamans/chatbotgate/pkg/shared/i18n"
    "github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// Create password handler
handler := password.NewHandler(
    config.PasswordAuthConfig{
        Enabled:  true,
        Password: "your-secure-password",
    },
    sessionStore,      // kvs.Store for sessions
    cookieConfig,      // config.CookieConfig
    "/_auth",          // authPathPrefix
    translator,        // *i18n.Translator
    logger,            // logging.Logger
)

// Key methods:
// - HandleLogin(w http.ResponseWriter, r *http.Request): Process password submission
// - RenderPasswordForm(lang i18n.Language) string: Generate HTML form for login page
```

**Use Case**: Simple authentication for testing, demos, and internal tools without OAuth2 or email setup.

**Security Note**: Password authentication uses a single shared password. Anyone with the password can authenticate as `password@localhost`. Use strong passwords and consider it for testing/development environments only.

**Standardized Fields:**

Password authentication populates these fields for consistent forwarding:

- `email`: "password@localhost"
- `username`: "Password User"
- `provider`: "password"
- `_email`: "password@localhost" (standardized)
- `_username`: "Password User" (standardized)
- `_avatar_url`: "" (empty, standardized)

#### Authorization (`pkg/middleware/authz`)

Email/domain-based authorization:

```go
import (
    "github.com/ideamans/chatbotgate/pkg/middleware/authz"
)

// Create authorizer
authz := authz.NewAuthorizer([]string{
    "user@example.com",
    "@company.com",
})

// Check authorization
allowed := authz.IsAllowed("user@example.com")  // true
allowed = authz.IsAllowed("user@company.com")    // true
allowed = authz.IsAllowed("other@gmail.com")     // false
```

#### Session Management (`pkg/middleware/session`)

Session management using KVS-backed helper functions:

```go
import (
    "time"
    "github.com/ideamans/chatbotgate/pkg/middleware/session"
    "github.com/ideamans/chatbotgate/pkg/shared/kvs"
)

// Create KVS store for sessions
store := kvs.NewMemoryStore()  // or LevelDB, Redis

// Create a new session
sess := &session.Session{
    ID:            "session-id-123",
    Email:         "user@example.com",
    Name:          "John Doe",
    Provider:      "google",
    Extra:         map[string]interface{}{
        "_email": "user@example.com",
        "_username": "John Doe",
        "_avatar_url": "https://example.com/avatar.jpg",
    },
    CreatedAt:     time.Now(),
    ExpiresAt:     time.Now().Add(7 * 24 * time.Hour), // 7 days
    Authenticated: true,
}

// Save session to store
err := session.Set(store, sess.ID, sess)

// Retrieve session
sess, err := session.Get(store, "session-id-123")
if err != nil {
    // Handle error (session not found or expired)
}

// Check if session is valid
if sess.IsValid() {
    // Session is authenticated and not expired
}

// Delete session
err = session.Delete(store, "session-id-123")
```

#### Access Control Rules (`pkg/middleware/rules`)

Path-based access control:

```go
import (
    "github.com/ideamans/chatbotgate/pkg/middleware/rules"
)

// Define rules using RuleConfig
allTrue := true
ruleSet := []rules.RuleConfig{
    {
        Prefix: "/public/",  // Use specific field, not Type/Pattern
        Action: rules.ActionAllow,
    },
    {
        Exact:  "/health",
        Action: rules.ActionAllow,
    },
    {
        All:    &allTrue,  // Pointer to bool
        Action: rules.ActionAuth,
    },
}

// Evaluate
engine := rules.NewEngine(ruleSet)
action := engine.Evaluate("/public/image.png")  // ActionAllow
action = engine.Evaluate("/app/dashboard")       // ActionAuth
```

**Rule Matchers** (use specific fields in `RuleConfig`):
- `Exact`: Exact path match (string)
- `Prefix`: Path prefix match (string)
- `Regex`: Regular expression match (string)
- `Minimatch`: Glob pattern match (string)
- `All`: Catch-all (*bool pointer)

**Actions:**
- `ActionAllow`: Allow without authentication
- `ActionAuth`: Require authentication
- `ActionDeny`: Deny access (403)

#### User Information Forwarding (`pkg/middleware/forwarding`)

Forward user data to upstream:

```go
import (
    "github.com/ideamans/chatbotgate/pkg/middleware/forwarding"
)

// Configure forwarder
forwarder, err := forwarding.NewForwarder(forwarding.Config{
    Encryption: &forwarding.EncryptionConfig{
        Key:       "32-char-encryption-key-here-12",
        Algorithm: "aes-256-gcm",
    },
    Fields: []forwarding.FieldConfig{
        {
            Path:    "email",
            Header:  "X-Auth-Email",
            Query:   "email",
            Filters: []string{"encrypt"},
        },
    },
})

// Forward to request
userInfo := &core.UserInfo{Email: "user@example.com"}
req, err := forwarder.ForwardToRequest(req, userInfo)
```

### Proxy Package

The `pkg/proxy/core` package handles reverse proxying (package name is `proxy`):

```go
import (
    proxy "github.com/ideamans/chatbotgate/pkg/proxy/core"
)

// Simple proxy with URL only
proxyHandler, err := proxy.NewHandler("http://localhost:8080")

// Or with full configuration
proxyHandler, err := proxy.NewHandlerWithConfig(proxy.UpstreamConfig{
    URL: "http://localhost:8080",
    Secret: proxy.SecretConfig{  // Not a pointer
        Header: "X-Chatbotgate-Secret",
        Value:  "secret-token",
    },
})

// Use as handler
http.Handle("/", proxyHandler)
```

**Features:**
- HTTP/HTTPS proxying
- WebSocket support
- Host-based routing
- Custom headers
- Upstream secret injection

### Shared Package

The `pkg/shared` package provides reusable components:

#### KVS Interface (`pkg/shared/kvs`)

Unified Key-Value Store interface:

```go
import (
    "github.com/ideamans/chatbotgate/pkg/shared/kvs"
)

// KVS interface
type Store interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Close() error
}

// Implementations
memStore := kvs.NewMemoryStore()
levelStore, err := kvs.NewLevelDBStore("/path/to/db")
redisStore, err := kvs.NewRedisStore(kvs.RedisConfig{
    Addr: "localhost:6379",
})
```

**Features:**
- Namespace isolation
- TTL support
- Atomic operations
- Multiple backends

**Advanced KVS Operations:**

The KVS interface provides additional methods for advanced use cases:

```go
import (
    "context"
    "github.com/ideamans/chatbotgate/pkg/shared/kvs"
)

// Check if a key exists (without fetching value)
exists, err := store.Exists(ctx, "session:abc123")
if err != nil {
    // Handle error
}
if exists {
    // Key exists
}

// List all keys matching a prefix
sessionKeys, err := store.List(ctx, "session:")
if err != nil {
    // Handle error
}
for _, key := range sessionKeys {
    fmt.Println("Session key:", key)
}

// Count keys matching a prefix
count, err := store.Count(ctx, "session:")
if err != nil {
    // Handle error
}
fmt.Printf("Total sessions: %d\n", count)

// List all keys (empty prefix)
allKeys, err := store.List(ctx, "")
```

**Use Cases:**
- **Session monitoring**: Count active sessions across the system
- **Cleanup operations**: List and delete expired tokens
- **Storage analytics**: Monitor key distribution by namespace
- **Admin dashboards**: Display real-time statistics
- **Rate limit tracking**: Count rate limit entries per IP

**Performance Notes:**
- `Exists()` is more efficient than `Get()` when you only need to check presence
- `List()` and `Count()` may be expensive on large datasets (especially Redis)
- Use specific prefixes to limit result sets
- Consider pagination for large result sets

#### Internationalization (`pkg/shared/i18n`)

Multi-language support:

```go
import (
    "github.com/ideamans/chatbotgate/pkg/shared/i18n"
)

// Detect language
lang := i18n.DetectLanguage(r)  // "en" or "ja"

// Get translation
text := i18n.T(lang, "login.title")  // "Sign In" or "ログイン"

// With parameters
text = i18n.T(lang, "login.welcome", "username", "John")
```

**Supported Languages:**
- English (`en`)
- Japanese (`ja`)

#### Logging (`pkg/shared/logging`)

Structured logging:

```go
import (
    "github.com/ideamans/chatbotgate/pkg/shared/logging"
)

// Create logger
logger := logging.New(logging.Config{
    Level:       "info",
    ModuleLevel: "debug",
    Color:       true,
})

// Log messages
logger.Info("Server started", "port", 4180)
logger.Debug("OAuth2 token exchange", "provider", "google")
logger.Error("Authentication failed", "error", err)
```

## Programming Interfaces

### Building a Custom Auth Proxy

```go
package main

import (
    "log"
    "net/http"

    "github.com/ideamans/chatbotgate/pkg/middleware/config"
    "github.com/ideamans/chatbotgate/pkg/middleware/factory"
    proxy "github.com/ideamans/chatbotgate/pkg/proxy/core"
    "github.com/ideamans/chatbotgate/pkg/shared/logging"
)

func main() {
    // Configure application
    cfg := &config.Config{
        Service: config.ServiceConfig{
            Name: "My Custom Proxy",
        },
        Server: config.ServerConfig{
            Host: "0.0.0.0",
            Port: 4180,
        },
        Session: config.SessionConfig{
            Cookie: config.CookieConfig{
                Name:   "_session",
                Secret: "your-random-secret-here-32-chars",
                Expire: "168h",
            },
        },
        OAuth2: config.OAuth2Config{
            Providers: []config.OAuth2Provider{
                {
                    ID:           "google",
                    Type:         "google",
                    DisplayName:  "Google",
                    ClientID:     "your-client-id",
                    ClientSecret: "your-client-secret",
                },
            },
        },
        Authorization: config.AuthorizationConfig{
            Allowed: []string{"@example.com"},
        },
        Proxy: config.ProxyConfig{
            Upstream: config.UpstreamConfig{
                URL: "http://localhost:8080",
            },
        },
    }

    // Create logger
    logger := logging.New(logging.Config{Level: "info"})

    // Create factory
    f := factory.NewFactory()

    // Create KVS stores (session, token, ratelimit)
    sessionStore, tokenStore, rateLimitStore, err := f.CreateKVSStores(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer sessionStore.Close()
    defer tokenStore.Close()
    defer rateLimitStore.Close()

    // Create proxy handler for upstream
    proxyHandler, err := proxy.NewHandler(cfg.Proxy.Upstream.URL)
    if err != nil {
        log.Fatal(err)
    }

    // Create middleware with all components
    mw, err := f.CreateMiddleware(
        cfg,
        sessionStore,
        tokenStore,
        rateLimitStore,
        proxyHandler,
        logger,
    )
    if err != nil {
        log.Fatal(err)
    }

    // Wrap proxy with middleware
    handler := mw.Wrap(proxyHandler)

    // Start server
    log.Println("Starting server on :4180")
    log.Fatal(http.ListenAndServe(":4180", handler))
}
```

### Implementing a Custom OAuth2 Provider

```go
package myprovider

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
    goauth2 "golang.org/x/oauth2"
)

type MyCustomProvider struct {
    config      *goauth2.Config
    userinfoURL string
}

func NewMyCustomProvider(clientID, clientSecret, redirectURL string) *MyCustomProvider {
    return &MyCustomProvider{
        config: &goauth2.Config{
            ClientID:     clientID,
            ClientSecret: clientSecret,
            RedirectURL:  redirectURL,
            Scopes:       []string{"openid", "email", "profile"},
            Endpoint: goauth2.Endpoint{
                AuthURL:  "https://myprovider.com/oauth/authorize",
                TokenURL: "https://myprovider.com/oauth/token",
            },
        },
        userinfoURL: "https://myprovider.com/oauth/userinfo",
    }
}

func (p *MyCustomProvider) Name() string {
    return "myprovider"
}

func (p *MyCustomProvider) Config() *goauth2.Config {
    return p.config
}

func (p *MyCustomProvider) GetUserInfo(ctx context.Context, token *goauth2.Token) (*oauth2.UserInfo, error) {
    // Fetch user info
    client := p.config.Client(ctx, token)
    resp, err := client.Get(p.userinfoURL)
    if err != nil {
        return nil, fmt.Errorf("userinfo fetch failed: %w", err)
    }
    defer resp.Body.Close()

    // Parse response
    var user struct {
        Email   string `json:"email"`
        Name    string `json:"name"`
        Picture string `json:"picture"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        return nil, fmt.Errorf("userinfo decode failed: %w", err)
    }

    // Return user info with standardized fields
    return &oauth2.UserInfo{
        Email: user.Email,
        Name:  user.Name,
        Extra: map[string]interface{}{
            // Standardized fields (common across all providers)
            "_email":      user.Email,
            "_username":   user.Name,
            "_avatar_url": user.Picture,
            // Provider-specific fields
            "picture": user.Picture,
            // OAuth2 tokens (stored in secrets)
            "secrets": map[string]interface{}{
                "access_token":  token.AccessToken,
                "refresh_token": token.RefreshToken,
            },
        },
    }, nil
}
```

**Note:** Custom providers are typically registered via the factory pattern. To use a custom provider, add it to your OAuth2 configuration:

```go
cfg.OAuth2.Providers = append(cfg.OAuth2.Providers, config.OAuth2Provider{
    ID:           "myprovider",
    Type:         "custom",
    DisplayName:  "My Provider",
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    AuthURL:      "https://myprovider.com/oauth/authorize",
    TokenURL:     "https://myprovider.com/oauth/token",
    UserinfoURL:  "https://myprovider.com/oauth/userinfo",
    Scopes:       []string{"openid", "email", "profile"},
})
```

### Implementing a Custom KVS Backend

```go
package customkvs

import (
    "context"
    "time"

    "github.com/ideamans/chatbotgate/pkg/shared/kvs"
)

type CustomStore struct {
    // Your storage implementation
    data map[string][]byte
}

func NewCustomStore() kvs.Store {
    return &CustomStore{
        data: make(map[string][]byte),
    }
}

func (s *CustomStore) Get(ctx context.Context, key string) ([]byte, error) {
    value, ok := s.data[key]
    if !ok {
        return nil, kvs.ErrNotFound
    }
    return value, nil
}

func (s *CustomStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
    s.data[key] = value
    // TODO: Implement TTL handling
    return nil
}

func (s *CustomStore) Delete(ctx context.Context, key string) error {
    delete(s.data, key)
    return nil
}

func (s *CustomStore) Close() error {
    // Cleanup resources
    return nil
}
```

### Adding Custom Middleware

```go
package main

import (
    "log"
    "net/http"

    "github.com/ideamans/chatbotgate/pkg/middleware/core"
)

// Custom middleware that logs requests
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}

// Custom middleware that adds headers
func customHeaderMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Custom-Header", "My Value")
        next.ServeHTTP(w, r)
    })
}

func main() {
    // Create ChatbotGate middleware
    middleware, err := core.NewMiddleware(config)
    if err != nil {
        log.Fatal(err)
    }

    // Chain middlewares
    handler := loggingMiddleware(
        customHeaderMiddleware(
            middleware.Handler(upstreamHandler),
        ),
    )

    http.ListenAndServe(":4180", handler)
}
```

## Examples

### Example 1: Simple OAuth2 Proxy

```go
package main

import (
    "log"
    "net/http"

    "github.com/ideamans/chatbotgate/pkg/middleware/config"
    "github.com/ideamans/chatbotgate/pkg/middleware/factory"
    "github.com/ideamans/chatbotgate/pkg/shared/logging"
)

func main() {
    cfg := &config.Config{
        Service: config.ServiceConfig{Name: "My App"},
        Session: config.SessionConfig{
            Cookie: config.CookieConfig{
                Secret: "32-char-secret-here-minimum-length",
                Expire: "24h",
            },
        },
        OAuth2: config.OAuth2Config{
            Providers: []config.OAuth2Provider{
                {
                    ID:           "google",
                    Type:         "google",
                    ClientID:     "your-client-id",
                    ClientSecret: "your-client-secret",
                },
            },
        },
    }

    // Setup
    logger := logging.New(logging.Config{Level: "info"})
    f := factory.NewFactory()
    sessionStore, tokenStore, rateLimitStore, _ := f.CreateKVSStores(cfg)
    defer sessionStore.Close()
    defer tokenStore.Close()
    defer rateLimitStore.Close()

    // Upstream handler
    upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, authenticated user!"))
    })

    // Create middleware
    mw, _ := f.CreateMiddleware(cfg, sessionStore, tokenStore, rateLimitStore, upstream, logger)

    // Start server
    http.ListenAndServe(":4180", mw.Wrap(upstream))
}
```

### Example 2: Using Custom OIDC Provider

```go
package main

import (
    "github.com/ideamans/chatbotgate/pkg/middleware/config"
)

func main() {
    // Configure custom OIDC provider
    cfg := &config.Config{
        OAuth2: config.OAuth2Config{
            Providers: []config.OAuth2Provider{
                {
                    ID:           "keycloak",
                    Type:         "custom",
                    DisplayName:  "Keycloak",
                    ClientID:     "chatbotgate",
                    ClientSecret: "your-secret",
                    AuthURL:      "https://keycloak.example.com/auth/realms/myrealm/protocol/openid-connect/auth",
                    TokenURL:     "https://keycloak.example.com/auth/realms/myrealm/protocol/openid-connect/token",
                    UserinfoURL:  "https://keycloak.example.com/auth/realms/myrealm/protocol/openid-connect/userinfo",
                    Scopes:       []string{"openid", "email", "profile"},
                },
            },
        },
    }

    // Use cfg with factory as shown in Example 1
}
```

### Example 3: Programmatic Rule Evaluation

```go
package main

import (
    "fmt"

    "github.com/ideamans/chatbotgate/pkg/middleware/rules"
)

func main() {
    allTrue := true
    engine := rules.NewEngine([]rules.RuleConfig{
        {Prefix: "/api/", Action: rules.ActionAuth},
        {Exact: "/health", Action: rules.ActionAllow},
        {All: &allTrue, Action: rules.ActionDeny},
    })

    paths := []string{"/api/users", "/health", "/admin", "/static/style.css"}

    for _, path := range paths {
        action := engine.Evaluate(path)
        fmt.Printf("%s -> %s\n", path, action)
    }
}
```

### Example 4: Accessing User Info via Forwarding

User information is forwarded to upstream applications via headers and query parameters. Configure forwarding in your config:

```go
package main

import (
    "fmt"
    "net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
    // User info is forwarded via headers (configured in forwarding section)
    email := r.Header.Get("X-Auth-Email")
    username := r.Header.Get("X-Auth-User")
    provider := r.Header.Get("X-Auth-Provider")

    if email == "" {
        http.Error(w, "Not authenticated", http.StatusUnauthorized)
        return
    }

    response := fmt.Sprintf("Hello, %s (email: %s, provider: %s)", username, email, provider)
    w.Write([]byte(response))
}
```

**Configuration example:**

```yaml
forwarding:
  fields:
    - path: email
      header: X-Auth-Email
    - path: _username
      header: X-Auth-User
    - path: provider
      header: X-Auth-Provider
```

## Testing

### Unit Testing

ChatbotGate uses Go's standard testing package:

```bash
# Run all tests
go test ./...

# Run specific package
go test ./pkg/middleware/auth/oauth2

# With verbose output
go test -v ./pkg/...

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Testing

Example integration test:

```go
package integration_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/ideamans/chatbotgate/pkg/middleware/core"
    "github.com/ideamans/chatbotgate/pkg/middleware/config"
)

func TestAuthenticationFlow(t *testing.T) {
    // Create test config
    cfg := &config.Config{
        Session: config.SessionConfig{
            Cookie: config.CookieConfig{
                Secret: "test-secret-32-characters-long",
                Expire: "1h",
            },
        },
    }

    // Create logger and factory
    logger := logging.New(logging.Config{Level: "info"})
    f := factory.NewFactory()

    // Create KVS stores
    sessionStore, tokenStore, rateLimitStore, err := f.CreateKVSStores(cfg)
    if err != nil {
        t.Fatal(err)
    }
    defer sessionStore.Close()
    defer tokenStore.Close()
    defer rateLimitStore.Close()

    // Create test upstream
    upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })

    // Create middleware
    mw, err := f.CreateMiddleware(cfg, sessionStore, tokenStore, rateLimitStore, upstream, logger)
    if err != nil {
        t.Fatal(err)
    }

    // Create test server
    handler := mw.Wrap(upstream)
    server := httptest.NewServer(handler)
    defer server.Close()

    // Test unauthenticated request (should redirect to login)
    resp, err := http.Get(server.URL + "/app")
    if err != nil {
        t.Fatal(err)
    }
    if resp.StatusCode != http.StatusFound {
        t.Errorf("Expected redirect (302), got %d", resp.StatusCode)
    }
}
```

### Mocking

Example using interfaces for mocking:

```go
package myapp_test

import (
    "context"
    "testing"

    "github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
)

// Mock provider
type mockProvider struct {
    name     string
    config   *goauth2.Config
    userInfo *oauth2.UserInfo
    err      error
}

func (m *mockProvider) Name() string {
    return m.name
}

func (m *mockProvider) Config() *goauth2.Config {
    return m.config
}

func (m *mockProvider) GetUserInfo(ctx context.Context, token *goauth2.Token) (*oauth2.UserInfo, error) {
    return m.userInfo, m.err
}

func TestOAuth2Flow(t *testing.T) {
    provider := &mockProvider{
        name: "mock",
        config: &goauth2.Config{
            ClientID:     "test-client-id",
            ClientSecret: "test-secret",
        },
        userInfo: &oauth2.UserInfo{
            Email: "test@example.com",
            Name:  "Test User",
            Extra: map[string]interface{}{
                "_email":    "test@example.com",
                "_username": "Test User",
            },
        },
    }

    // Test with mock token
    token := &goauth2.Token{AccessToken: "mock-token"}
    user, err := provider.GetUserInfo(context.Background(), token)
    if err != nil {
        t.Fatal(err)
    }
    if user.Email != "test@example.com" {
        t.Errorf("Expected test@example.com, got %s", user.Email)
    }
}
```

## API Reference

### Core Types

#### `oauth2.UserInfo` and `session.Session`

**OAuth2 UserInfo** (`pkg/middleware/auth/oauth2`):

```go
type UserInfo struct {
    Email string                 // User email address
    Name  string                 // User display name (optional)
    Extra map[string]interface{} // Additional provider-specific data
}
```

**Session** (`pkg/middleware/session`):

```go
type Session struct {
    ID            string                 // Session ID
    Email         string                 // User's email address
    Name          string                 // User's display name from OAuth2 provider
    Provider      string                 // OAuth2 provider name, "email" for email auth, or "password" for password auth
    Extra         map[string]interface{} // Additional user data from OAuth2 provider
    CreatedAt     time.Time              // Session creation time
    ExpiresAt     time.Time              // Session expiration time
    Authenticated bool                   // Authentication status
}
```

**Standardized Extra Fields** (common across all OAuth2 providers, email auth, and password auth):
- `_email` (string): User email address (same as `Email`)
- `_username` (string): User display name
  - Google: `name`
  - GitHub: `name` (fallback to `login` if not set)
  - Microsoft: `displayName`
  - Email auth: email local part (before @)
- `_avatar_url` (string): User profile picture URL
  - Google: `picture` URL
  - GitHub: `avatar_url` URL
  - Microsoft: empty (requires separate photo endpoint)
  - Email auth: empty

**Provider-Specific Extra Fields:**
- Google: `email`, `name`, `picture`, `verified_email`, `given_name`, `family_name`
- GitHub: `email`, `name`, `login`, `avatar_url`, plus other public profile data
- Microsoft: `email`, `displayName`, `userPrincipalName`, `preferredUsername`
- Email auth: `userpart` (email local part before @, same as `_username`)

**OAuth2 Tokens** (under `secrets`):
- `secrets.access_token` (string): OAuth2 access token
- `secrets.refresh_token` (string): OAuth2 refresh token (if available)

#### `middleware.Middleware`

**Note:** The Middleware struct is created via `factory.Factory.CreateMiddleware()`. It implements `http.Handler` via the `Wrap()` method.

```go
// Create middleware using factory
mw, err := factory.CreateMiddleware(cfg, sessionStore, tokenStore, rateLimitStore, upstreamHandler, logger)

// Wrap upstream handler with authentication
handler := mw.Wrap(upstreamHandler)
```

**Key Method:**
- `Wrap(next http.Handler) http.Handler`: Wraps the upstream handler with authentication middleware

#### `middleware/config.Config`

```go
type Config struct {
    Service       ServiceConfig       // Service branding
    Server        ServerConfig        // Server settings
    Session       SessionConfig       // Session configuration
    OAuth2        OAuth2Config        // OAuth2 providers
    EmailAuth     EmailAuthConfig     // Email authentication
    Authorization AuthorizationConfig // Access control
    KVS           KVSConfig           // Storage backend
    Forwarding    ForwardingConfig    // User info forwarding
    Rules         rules.Config        // Access control rules (embedded)
    Logging       LoggingConfig       // Logging settings
    Proxy         ProxyConfig         // Proxy configuration
}

type SessionConfig struct {
    Cookie CookieConfig // Cookie configuration
}

type CookieConfig struct {
    Name     string // Cookie name (default: "_oauth2_proxy")
    Secret   string // Cookie secret (required, 32+ characters)
    Expire   string // Session expiration (duration string, e.g., "168h")
    Secure   bool   // HTTPS only (default: false for dev, true for prod)
    HttpOnly bool   // Prevent JavaScript access (default: true)
    SameSite string // SameSite policy: "strict", "lax", "none" (default: "lax")
}
```

### KVS Interface

#### `shared/kvs.Store`

```go
type Store interface {
    // Get retrieves a value by key
    Get(ctx context.Context, key string) ([]byte, error)

    // Set stores a value with optional TTL
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

    // Delete removes a key
    Delete(ctx context.Context, key string) error

    // Exists checks if a key exists
    Exists(ctx context.Context, key string) (bool, error)

    // List returns all keys matching the prefix
    List(ctx context.Context, prefix string) ([]string, error)

    // Count returns the number of keys matching the prefix
    Count(ctx context.Context, prefix string) (int, error)

    // Close releases resources
    Close() error
}
```

**Implementations:**
- `kvs.NewMemoryStore()`: In-memory KVS (fast, ephemeral)
- `kvs.NewLevelDBStore(path)`: LevelDB KVS (persistent, embedded)
- `kvs.NewRedisStore(config)`: Redis KVS (distributed, scalable)

### OAuth2 Interface

#### `middleware/auth/oauth2.Provider`

```go
type Provider interface {
    // Name returns provider identifier
    Name() string

    // Config returns the OAuth2 configuration
    Config() *oauth2.Config

    // GetUserInfo retrieves user information using the OAuth2 token
    GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
}
```

**Built-in Providers:**
- Google (`google`): Google OAuth2
- GitHub (`github`): GitHub OAuth2
- Microsoft (`microsoft`): Microsoft/Azure AD OAuth2
- Custom (`custom`): Generic OIDC provider

### Session Functions

#### `middleware/session` Helper Functions

The session package provides helper functions for working with sessions:

```go
// Get retrieves a session from KVS by ID
func Get(store kvs.Store, id string) (*Session, error)

// Set stores a session in KVS with the given ID
func Set(store kvs.Store, id string, session *Session) error

// Delete removes a session from KVS by ID
func Delete(store kvs.Store, id string) error

// List returns all active sessions from KVS
func List(store kvs.Store) ([]*Session, error)

// Count returns the number of active sessions in KVS
func Count(store kvs.Store) (int, error)
```

### Rules Interface

#### `middleware/rules.Engine`

```go
type Engine interface {
    // Evaluate evaluates rules for a path
    Evaluate(path string) Action
}

type Action string

const (
    ActionAllow Action = "allow" // Allow without auth
    ActionAuth  Action = "auth"  // Require auth
    ActionDeny  Action = "deny"  // Deny access
)
```

### Proxy Handler

#### `proxy/core.Handler`

The proxy handler implements `http.Handler` for reverse proxying:

```go
// Handler is the proxy implementation (no Close method)
type Handler struct {
    // Internal fields
}

// Handler implements http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request)

// Create handler
handler, err := proxy.NewHandler(upstreamURL)
handler, err := proxy.NewHandlerWithConfig(proxy.UpstreamConfig{...})
```

**Note:** The `Handler` struct does not have a `Close()` method. Resources are automatically managed.

## Best Practices

### 1. Configuration Management

```go
// Load from file
cfg, err := config.LoadFromFile("config.yaml")

// Validate before use
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

// Watch for changes
watcher := filewatcher.New("config.yaml", func(path string) {
    cfg, _ = config.LoadFromFile(path)
    // Reload components
})
```

### 2. Error Handling

ChatbotGate uses structured error handling with specific error types for different scenarios.

#### Configuration Validation Errors

Configuration errors are collected and returned as `ValidationError`:

```go
import (
    "github.com/ideamans/chatbotgate/pkg/middleware/config"
)

// Validate configuration
cfg, err := config.LoadFromFile("config.yaml")
if err != nil {
    log.Fatal(err)
}

// Validate returns all validation errors at once
err = cfg.Validate()
if err != nil {
    // Check if it's a validation error
    if verr, ok := err.(*config.ValidationError); ok {
        fmt.Println("Configuration errors:")
        for _, e := range verr.Errors {  // Field access, not method call
            fmt.Printf("  - %s\n", e)
        }
        os.Exit(1)
    }
    // Other error
    log.Fatal(err)
}
```

**Common Validation Errors:**
- Cookie secret too short (minimum 32 characters)
- No authentication methods enabled (neither OAuth2, email, nor password)
- Invalid encryption key length for forwarding
- Missing required provider configuration
- Invalid rule patterns (regex, minimatch)

#### KVS Errors

```go
import (
    "errors"
    "github.com/ideamans/chatbotgate/pkg/shared/kvs"
)

// Get operation
data, err := store.Get(ctx, "session:abc123")
if err != nil {
    // Check for specific errors
    if errors.Is(err, kvs.ErrNotFound) {
        // Key doesn't exist (not an error in many cases)
        return nil
    }
    if errors.Is(err, kvs.ErrClosed) {
        // Store was closed (fatal error)
        log.Fatal("KVS store closed")
    }
    // Other error (network, timeout, etc.)
    return fmt.Errorf("get session: %w", err)
}

// Set operation with TTL
err = store.Set(ctx, "token:xyz", data, 15*time.Minute)
if err != nil {
    return fmt.Errorf("save token: %w", err)
}

// Delete operation (idempotent, no error if key doesn't exist)
err = store.Delete(ctx, "session:abc123")
if err != nil {
    return fmt.Errorf("delete session: %w", err)
}
```

**Available KVS Errors:**
- `kvs.ErrNotFound`: Key doesn't exist in store
- `kvs.ErrClosed`: Store was closed and can't be used
- Context errors: `context.Canceled`, `context.DeadlineExceeded`

#### OAuth2 Provider Errors

```go
import (
    "errors"
    "github.com/ideamans/chatbotgate/pkg/middleware/auth/oauth2"
)

// Token exchange
userInfo, err := provider.GetUserInfo(ctx, token)
if err != nil {
    // Check for specific OAuth2 errors
    if errors.Is(err, oauth2.ErrInvalidToken) {
        // Token is invalid or expired
        return fmt.Errorf("invalid token: %w", err)
    }
    if errors.Is(err, oauth2.ErrEmailNotFound) {
        // Provider didn't return email (missing scope?)
        return fmt.Errorf("email not provided by OAuth2 provider: %w", err)
    }
    // Other error (network, provider error, etc.)
    return fmt.Errorf("oauth2 user info: %w", err)
}
```

**Available OAuth2 Errors:**
- `oauth2.ErrInvalidToken`: Token is invalid, expired, or revoked
- `oauth2.ErrEmailNotFound`: Provider didn't return email address
- HTTP errors: Wrapped from provider API calls

#### Email Authentication Errors

```go
import (
    "github.com/ideamans/chatbotgate/pkg/middleware/auth/email"
)

// Send email
err := sender.Send(to, subject, htmlBody, textBody)
if err != nil {
    // SMTP errors, SendGrid API errors, etc.
    return fmt.Errorf("send email: %w", err)
}

// Token verification
email, err := tokenStore.VerifyToken(token)
if err != nil {
    // Token invalid, expired, or not found
    return fmt.Errorf("verify token: %w", err)
}
```

#### Error Wrapping Best Practices

```go
// Always wrap errors with context using %w
if err != nil {
    return fmt.Errorf("oauth2 exchange failed: %w", err)
}

// Chain context for better debugging
if err := doSomething(); err != nil {
    return fmt.Errorf("process user %s: %w", userID, err)
}

// Use errors.Is for checking error types
if errors.Is(err, kvs.ErrNotFound) {
    // Handle not found
}

// Use errors.As for extracting error types
var verr *config.ValidationError
if errors.As(err, &verr) {
    // Access validation errors
    for _, e := range verr.Errors {  // Field access, not method call
        fmt.Println(e)
    }
}
```

### 3. Context Propagation

```go
// Always pass context
ctx := r.Context()
user, err := provider.Exchange(ctx, code)

// Check context cancellation
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // Continue
}
```

### 4. Resource Cleanup

```go
// Always defer cleanup
store, err := kvs.NewRedisStore(cfg)
if err != nil {
    return err
}
defer store.Close()
```

### 5. Testing

```go
// Use table-driven tests
tests := []struct {
    name     string
    input    string
    expected Action
}{
    {"public", "/public/file.js", ActionAllow},
    {"app", "/app/dashboard", ActionAuth},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := engine.Evaluate(tt.input)
        if got != tt.expected {
            t.Errorf("got %v, want %v", got, tt.expected)
        }
    })
}
```

## Contributing

### Development Setup

```bash
# Clone repository
git clone https://github.com/ideamans/chatbotgate.git
cd chatbotgate

# Install dependencies
go mod download

# Run tests
go test ./...

# Run linters
go fmt ./...
go vet ./...
```

### Adding Features

1. Create feature branch
2. Implement feature with tests
3. Update documentation
4. Submit pull request

### Code Style

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `go fmt` for formatting
- Write tests for new code
- Document exported functions
- Aim for 80%+ test coverage

---

**Questions?** Open an issue or discussion on [GitHub](https://github.com/ideamans/chatbotgate).
