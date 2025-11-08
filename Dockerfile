# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o chatbotgate \
    ./cmd/chatbotgate

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
# - ca-certificates: for HTTPS
# - tzdata: timezone support
# - ssmtp: lightweight sendmail replacement for email authentication
RUN apk add --no-cache ca-certificates tzdata ssmtp && \
    ln -sf /usr/sbin/ssmtp /usr/sbin/sendmail

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/chatbotgate /app/chatbotgate

# Copy example configurations
COPY --from=builder /build/examples /app/examples

# Note: Web assets are embedded in the binary via Go embed
# No need to copy /build/web directory

# Create config directory
RUN mkdir -p /app/config && \
    chown -R app:app /app

# Switch to non-root user
USER app

# Expose default port
EXPOSE 4180

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:4180/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/chatbotgate"]

# Default command (can be overridden)
CMD ["-config", "/app/config/config.yaml"]
