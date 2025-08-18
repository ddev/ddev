---
search:
  boost: .1
---

# HTTP Proxy Test Environments

!!!tip "Advanced: SSL Interception Testing"
    For testing SSL interception scenarios (like corporate VPN/firewall environments), see [Network Test Environments: Packet-Inspection VPN Simulation](network-test-environments.md). This complements the basic HTTP proxy testing covered here.

This guide demonstrates how to set up HTTP proxy environments for testing DDEV behavior with proxy servers. This is useful for testing corporate environments that route traffic through HTTP proxies, simulating real-world network configurations where all traffic must pass through proxy servers.

## Setup Overview

The setup involves:

- Running a SOCKS5 proxy (Tor or Cloudflare WARP)
- Converting SOCKS5 to HTTP using Privoxy
- Configuring Docker to use the HTTP proxy
- Testing Docker pulls and container network access

## Step-by-Step Instructions

### 1. Install and Run SOCKS5 Proxy

!!!tip "Simple Testing Option"
    For basic HTTP proxy testing without SOCKS5 forwarding, you can skip this step and go directly to [Step 2](#2-forward-socks5-to-http-with-privoxy) to run Privoxy without any actual proxy behind it.

Choose one of the following options:

#### Option A: Using Tor

Install [Tor](https://www.torproject.org/download/):

```bash
# macOS
brew install tor

# Ubuntu/Debian
sudo apt-get install tor

# CentOS/RHEL/Fedora
sudo dnf install tor
```

Run Tor in the foreground:

```bash
sudo tor --SocksPort 127.0.0.1:9050
```

Verify Tor is running:

```bash
# Test SOCKS5 connection
curl --socks5 127.0.0.1:9050 https://httpbin.org/ip
```

#### Option B: Using Cloudflare WARP

Install [Cloudflare WARP](https://developers.cloudflare.com/warp-client/get-started/):

```bash
# macOS
brew install --cask cloudflare-warp

# Linux
# See https://pkg.cloudflareclient.com/
```

Configure for proxy mode:

**macOS/Windows (GUI):**

Configure proxy mode and port through the GUI interface.

**Linux (CLI):**

```bash
sudo systemctl enable --now warp-svc.service
warp-cli registration new
warp-cli mode proxy
warp-cli proxy port 9050
warp-cli connect

# Verify WARP is running
warp-cli status

# Test SOCKS5 connection
curl --socks5 127.0.0.1:9050 https://httpbin.org/ip

# When testing is done, disconnect
warp-cli disconnect
```

### 2. Forward SOCKS5 to HTTP with Privoxy

Install [Privoxy](https://www.privoxy.org/):

```bash
# macOS
brew install privoxy

# Ubuntu/Debian
sudo apt-get install privoxy

# CentOS/RHEL/Fedora
sudo dnf install privoxy
```

Run Privoxy in the foreground:

!!!note "Direct HTTP Proxy Testing"
    For simple testing without SOCKS5 forwarding, you can run Privoxy without any actual proxy behind it:

    ```bash
    privoxy --no-daemon <(echo -e "listen-address 0.0.0.0:8118\ndebug 1\n")
    ```

    This creates an HTTP proxy at `0.0.0.0:8118` without forwarding to Tor/WARP, useful for testing Docker proxy configuration alone.

**With SOCKS5 forwarding (for full proxy chain):**

```bash
privoxy --no-daemon <(echo -e "listen-address 0.0.0.0:8118\nforward-socks5 / 127.0.0.1:9050 .\ndebug 1\n")
```

**Alternative - Using temporary config file:**

```bash
# Create temporary config
cat > /tmp/privoxy.conf << EOF
listen-address 0.0.0.0:8118
forward-socks5 / 127.0.0.1:9050 .
debug 1
EOF

# Run Privoxy
privoxy --no-daemon /tmp/privoxy.conf
```

This configures Privoxy to:

- Listen on all interfaces port `8118` for HTTP proxy requests
- Forward all traffic to the SOCKS5 proxy at `127.0.0.1:9050`
- Enable debug logging to monitor connections

### 3. Configure Docker Daemon Proxy

Create or edit `/etc/docker/daemon.json` (requires sudo):

```json
{
    "proxies": {
        "http-proxy": "http://127.0.0.1:8118",
        "https-proxy": "http://127.0.0.1:8118",
        "no-proxy": "localhost,127.0.0.0/8"
    }
}
```

Restart Docker daemon:

```bash
sudo systemctl restart docker
```

### 4. Configure Docker Client Proxy

Create or edit `~/.docker/config.json` (user-level configuration):

```json
{
    "proxies": {
        "default": {
            "httpProxy": "http://host.docker.internal:8118",
            "httpsProxy": "http://host.docker.internal:8118",
            "noProxy": "localhost,127.0.0.1/8,::1,*.ddev.site"
        }
    }
}
```

!!!note "Docker Network Access"
    Using `host.docker.internal:8118` allows containers to reach the proxy running on the host system.

### 5. Test the Proxy Setup

!!!tip "Monitoring Connections"
    Keep the Privoxy terminal window visible to monitor proxy connections and debug any issues.

**Test Docker daemon proxy:**

```bash
# Test Docker registry access
docker pull ddev/ddev-utilities:latest

# Check Docker info for proxy configuration
docker info | grep -i proxy
```

**Test container proxy access:**

```bash
# Initialize and start a DDEV project if needed
# ddev config --auto
# ddev start

# Test container network access
ddev exec curl -v https://httpbin.org/ip

# Test package updates
ddev exec sudo apt-get update
```

**Verify proxy chain:**

You should see new connections appear in the Privoxy debug output, confirming that traffic is being routed through the proxy chain.

## Troubleshooting

### Connection Issues

If containers cannot reach the proxy:

**Verify Privoxy configuration:**

```bash
# Check if Privoxy is listening
netstat -an | grep 8118
# or
ss -tlnp | grep 8118
```

**Test host connectivity from container:**

```bash
# Check if host.docker.internal resolves
ddev exec nslookup host.docker.internal

# Test direct connection to proxy
ddev exec curl -v --connect-timeout 5 http://host.docker.internal:8118
```

**Check firewall and networking:**

- Ensure firewall rules allow traffic on port `8118`
- Verify Docker network configuration: `docker network inspect bridge`
- Check if SELinux/AppArmor is blocking connections

### Proxy Chain Issues

If the proxy chain is broken:

**Verify SOCKS5 proxy:**

```bash
# Check if SOCKS5 proxy is listening
netstat -an | grep 9050

# Test SOCKS5 proxy directly
curl --socks5 127.0.0.1:9050 https://httpbin.org/ip

# For WARP, check status
warp-cli status
```

**Debug Privoxy forwarding:**

```bash
# Test HTTP proxy directly
curl --proxy http://127.0.0.1:8118 https://httpbin.org/ip

# Check Privoxy logs for error messages
# Look for "forward-socks5" errors in the terminal output
```

**Common fixes:**

- Restart Privoxy with fresh configuration
- Verify SOCKS5 proxy port matches Privoxy forward configuration
- Check for conflicting proxy settings in environment variables

## Use Cases

This setup is valuable for testing:

- **Corporate proxy environments** - Simulate enterprise networks with mandatory proxy usage
- **Proxy authentication scenarios** - With additional Privoxy configuration for username/password auth

## Cleanup

When finished testing:

```bash
# Stop DDEV if running
ddev poweroff

# Remove Docker proxy configuration from
# /etc/docker/daemon.json
# ~/.docker/config.json

# Restart Docker
sudo systemctl restart docker

# Stop proxy services
# Ctrl+C in Privoxy terminal
# For WARP: warp-cli disconnect
# For Tor: Ctrl+C in Tor terminal
```

---

This HTTP proxy setup provides a realistic simulation of corporate network environments where all traffic must route through proxy servers, helping ensure DDEV works reliably in such configurations.
