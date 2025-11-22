# systemd Configuration Examples

This directory contains example systemd configuration files for running ChatbotGate as a system service.

## Files

- **`chatbotgate.service`** - Basic systemd service unit
- **`chatbotgate-with-redis.service`** - Service with Redis dependency
- **`journald.conf`** - journald configuration for log management
- **`setup-systemd.sh`** - Installation script (helps with setup)
- **`config.yaml`** - Example configuration optimized for systemd

## Quick Start

### 1. Install ChatbotGate Binary

```bash
# Download binary (replace with actual version)
wget https://github.com/ideamans/chatbotgate/releases/download/v1.0.0/chatbotgate-linux-amd64

# Install to system path
sudo install -m 755 chatbotgate-linux-amd64 /usr/local/bin/chatbotgate

# Verify installation
chatbotgate version
```

### 2. Create System User

```bash
# Create dedicated user (no login, no home directory)
sudo useradd -r -s /bin/false -d /nonexistent chatbotgate
```

### 3. Create Directory Structure

```bash
# Configuration directory
sudo mkdir -p /etc/chatbotgate
sudo chown root:chatbotgate /etc/chatbotgate
sudo chmod 750 /etc/chatbotgate

# Working directory
sudo mkdir -p /opt/chatbotgate
sudo chown chatbotgate:chatbotgate /opt/chatbotgate

# Log directory (optional, for file logging)
sudo mkdir -p /var/log/chatbotgate
sudo chown chatbotgate:chatbotgate /var/log/chatbotgate
sudo chmod 755 /var/log/chatbotgate
```

### 4. Install Configuration

```bash
# Use the example config as a starting point
sudo cp config.yaml /etc/chatbotgate/config.yaml

# Or copy from project root
# sudo cp ../../config.example.yaml /etc/chatbotgate/config.yaml

# Set proper ownership and permissions
sudo chown root:chatbotgate /etc/chatbotgate/config.yaml
sudo chmod 640 /etc/chatbotgate/config.yaml

# Edit configuration
sudo nano /etc/chatbotgate/config.yaml
```

**Important configuration changes:**
- Update `session.cookie_secret` to a random 32+ character string
- Configure your upstream URL in `proxy.upstream.url`
- Set up OAuth2 providers or email authentication
- Configure access control rules if needed

### 5. Install systemd Service

**Without Redis:**
```bash
sudo cp chatbotgate.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable chatbotgate
sudo systemctl start chatbotgate
```

**With Redis:**
```bash
sudo cp chatbotgate-with-redis.service /etc/systemd/system/chatbotgate.service
sudo systemctl daemon-reload
sudo systemctl enable chatbotgate
sudo systemctl start chatbotgate
```

### 6. Verify Service

```bash
# Check status
sudo systemctl status chatbotgate

# View logs
journalctl -u chatbotgate -f

# Test endpoint
curl http://localhost:4180/health
```

## Log Management

### View Logs

```bash
# Follow logs in real-time
journalctl -u chatbotgate -f

# Show logs from last hour
journalctl -u chatbotgate --since "1 hour ago"

# Show only errors
journalctl -u chatbotgate -p err

# Export logs
journalctl -u chatbotgate --since today > chatbotgate.log
```

### Configure Log Retention

```bash
# Copy journald config
sudo cp journald.conf /etc/systemd/journald.conf.d/chatbotgate.conf

# Restart journald
sudo systemctl restart systemd-journald

# Check disk usage
journalctl --disk-usage

# Manually clean old logs
journalctl --vacuum-time=3d
journalctl --vacuum-size=500M
```

## Service Management

### Basic Commands

```bash
# Start service
sudo systemctl start chatbotgate

# Stop service
sudo systemctl stop chatbotgate

# Restart service
sudo systemctl restart chatbotgate

# Reload configuration (if supported)
sudo systemctl reload chatbotgate

# Check status
sudo systemctl status chatbotgate

# Enable auto-start on boot
sudo systemctl enable chatbotgate

# Disable auto-start
sudo systemctl disable chatbotgate
```

### Troubleshooting

```bash
# Check if service is running
systemctl is-active chatbotgate

# Check if service is enabled
systemctl is-enabled chatbotgate

# View service file
systemctl cat chatbotgate

# View all service properties
systemctl show chatbotgate

# Check for failed units
systemctl --failed
```

## Security Hardening

The example service files include security hardening options:

- **`NoNewPrivileges=true`** - Prevents privilege escalation
- **`PrivateTmp=true`** - Isolated /tmp directory
- **`ProtectSystem=strict`** - Read-only filesystem except specified paths
- **`ProtectHome=true`** - No access to home directories
- **`ReadWritePaths=/var/log/chatbotgate`** - Explicit write permissions
- **`ReadOnlyPaths=/etc/chatbotgate`** - Config is read-only

Additional hardening options available in systemd 240+:

```ini
[Service]
# Restrict network access
PrivateNetwork=false
RestrictAddressFamilies=AF_INET AF_INET6

# Restrict system calls
SystemCallFilter=@system-service
SystemCallErrorNumber=EPERM

# No kernel modules
ProtectKernelModules=true

# No kernel tunables
ProtectKernelTunables=true

# No control groups
ProtectControlGroups=true

# Lock down /proc
ProcSubset=pid
```

## Resource Limits

Control resource usage:

```ini
[Service]
# CPU (100% = 1 core, 200% = 2 cores)
CPUQuota=100%

# Memory limit
MemoryLimit=512M
MemoryHigh=400M  # Soft limit, triggers memory pressure

# File descriptors
LimitNOFILE=65536

# Number of processes
TasksMax=512
```

## Advanced Configuration

### Environment Variables

```ini
[Service]
Environment="CHATBOTGATE_LOG_LEVEL=info"
Environment="CHATBOTGATE_ENV=production"
EnvironmentFile=/etc/chatbotgate/environment
```

### Socket Activation

For socket activation (systemd manages the socket):

```ini
# chatbotgate.socket
[Unit]
Description=ChatbotGate Socket

[Socket]
ListenStream=4180
BindIPv6Only=both

[Install]
WantedBy=sockets.target
```

```ini
# chatbotgate.service
[Unit]
Requires=chatbotgate.socket

[Service]
ExecStart=/usr/local/bin/chatbotgate serve --config /etc/chatbotgate/config.yaml
```

### Health Checks

Monitor service health (requires systemd 240+):

```ini
[Service]
Type=notify
WatchdogSec=30s
Restart=on-watchdog
```

In your application, send watchdog notifications:
```go
import "github.com/coreos/go-systemd/v22/daemon"

// In main loop
daemon.SdNotify(false, daemon.SdNotifyWatchdog)
```

## Uninstallation

```bash
# Stop and disable service
sudo systemctl stop chatbotgate
sudo systemctl disable chatbotgate

# Remove service file
sudo rm /etc/systemd/system/chatbotgate.service
sudo systemctl daemon-reload

# Remove binary
sudo rm /usr/local/bin/chatbotgate

# Remove configuration and data
sudo rm -rf /etc/chatbotgate
sudo rm -rf /opt/chatbotgate
sudo rm -rf /var/log/chatbotgate

# Remove user
sudo userdel chatbotgate
```

## References

- [systemd.service documentation](https://www.freedesktop.org/software/systemd/man/systemd.service.html)
- [systemd.exec documentation](https://www.freedesktop.org/software/systemd/man/systemd.exec.html)
- [journald.conf documentation](https://www.freedesktop.org/software/systemd/man/journald.conf.html)
- [journalctl documentation](https://www.freedesktop.org/software/systemd/man/journalctl.html)
