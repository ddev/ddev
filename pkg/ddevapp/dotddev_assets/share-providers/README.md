# DDEV Share Providers

Share providers are scripts that create internet-accessible tunnels to your local DDEV project.

## Built-in Providers

- **ngrok** (default): Uses ngrok.com service
- **cloudflared**: Uses Cloudflare Tunnel (free, no account required)

## Usage

```bash
# Use default provider (ngrok)
ddev share

# Use specific provider
ddev share --provider=cloudflared

# Set default provider for project
ddev config --share-provider=cloudflared
```

## Configuration

In `.ddev/config.yaml`:

```yaml
share_default_provider: ngrok
share_ngrok_args: "--basic-auth username:password"
share_cloudflared_args: ""
```

## Customizing Providers

### Take Ownership of Built-in Provider

1. Edit the provider script (e.g., `ngrok.sh`)
2. Remove the `#ddev-generated` line at the top
3. Make your changes
4. DDEV will never overwrite this file again

### Create Custom Variant

```bash
# Copy built-in provider
cp .ddev/share-providers/ngrok.sh .ddev/share-providers/my-ngrok.sh

# Edit my-ngrok.sh:
# - Remove '#ddev-generated' line
# - Customize as needed

# Use your variant
ddev share --provider=my-ngrok
```

### Create New Provider

Create a new executable script in `.ddev/share-providers/`:

```bash
#!/bin/bash
set -euo pipefail

# Start your tunnel tool
mytunnel http "$DDEV_LOCAL_URL" &
TUNNEL_PID=$!

# Capture public URL (however your tool exposes it)
URL=$(get-tunnel-url)

# Output URL to stdout (CRITICAL: first line only)
echo "$URL"

# Wait for tunnel to exit
wait $TUNNEL_PID
```

## Provider Script Contract

Every share provider must follow this contract:

### Input (Environment Variables)

- `DDEV_LOCAL_URL`: Local URL to tunnel (e.g., `http://127.0.0.1:8080`)
- `DDEV_SHARE_<PROVIDER>_ARGS`: Provider-specific arguments (optional)
- All standard DDEV environment variables

### Output

- **stdout**: Public URL (first line only - captured by DDEV)
- **stderr**: Logs, status messages (passed through to user)

### Lifecycle

1. Validate tool is installed
2. Validate required environment variables
3. Start tunnel process in background
4. Capture public URL (via API, stdout, file, etc.)
5. Output URL to stdout
6. Wait for tunnel process to exit

### Signal Handling

- Must handle SIGINT (Ctrl+C) and SIGTERM gracefully
- Use `trap` to cleanup background processes on exit
- Example:
  ```bash
  cleanup() {
      if kill -0 $PID 2>/dev/null; then
          kill $PID 2>/dev/null || true
      fi
  }
  trap cleanup EXIT
  ```

### Error Handling

- Exit with non-zero status on errors
- Write error messages to stderr (not stdout)
- Provide helpful error messages (missing tool, timeout, etc.)

## Examples

### Simple Provider (stdout parsing)

```bash
#!/bin/bash
set -euo pipefail

# Start tunnel, parse URL from output
localtunnel --port 8080 | grep -o 'https://[^[:space:]]*' | head -1

# Note: This simple example doesn't handle background process or signals
```

### API Polling Provider

```bash
#!/bin/bash
set -euo pipefail

# Start tunnel
mytunnel http "$DDEV_LOCAL_URL" &
TUNNEL_PID=$!

trap "kill $TUNNEL_PID 2>/dev/null || true" EXIT

# Poll API for URL
for i in {1..30}; do
    URL=$(curl -s http://localhost:4040/api | jq -r '.url' || echo "")
    if [[ -n "$URL" ]]; then
        echo "$URL"
        break
    fi
    sleep 1
done

wait $TUNNEL_PID
```

### File-based Provider

```bash
#!/bin/bash
set -euo pipefail

# Start tunnel (writes URL to file)
mytunnel --url-file /tmp/tunnel.url "$DDEV_LOCAL_URL" &
TUNNEL_PID=$!

trap "kill $TUNNEL_PID 2>/dev/null || true" EXIT

# Wait for URL file
for i in {1..30}; do
    if [[ -f /tmp/tunnel.url ]]; then
        cat /tmp/tunnel.url
        break
    fi
    sleep 1
done

wait $TUNNEL_PID
```

## Hooks Integration

After the tunnel URL is captured, DDEV sets the `DDEV_SHARE_URL` environment variable and runs pre-share hooks. This allows you to:

- Register webhooks with external services
- Update DNS records
- Send notifications
- Configure SSO callbacks

Example `.ddev/config.yaml`:

```yaml
hooks:
  pre-share:
    - exec: |
        echo "Tunnel URL: $DDEV_SHARE_URL"
        curl -X POST https://api.example.com/webhook \
          -d "url=$DDEV_SHARE_URL"
```

## Troubleshooting

### Provider not found

```
Error: share provider 'foo' not found
```

**Solution**: Check that `.ddev/share-providers/foo.sh` exists and is executable:

```bash
ls -la .ddev/share-providers/
chmod +x .ddev/share-providers/foo.sh
```

### Provider outputs no URL

```
Error: provider 'ngrok' did not output a URL
```

**Causes**:
- Tool not installed or not in PATH
- Tool requires authentication (e.g., ngrok account)
- API endpoint not responding
- Timeout (30 seconds)

**Debug**: Run the provider script directly:

```bash
export DDEV_LOCAL_URL=http://127.0.0.1:8080
.ddev/share-providers/ngrok.sh
```

### Invalid URL output

```
Error: provider 'foo' output invalid URL: not-a-url
```

**Solution**: Provider must output URL starting with `http://` or `https://`

## Platform Compatibility

- **Linux**: Fully supported
- **macOS**: Fully supported
- **Windows**: Requires Git Bash or WSL

Provider scripts should use `bash` (not `sh`) and avoid platform-specific commands where possible.

## More Information

- [DDEV Sharing Documentation](https://ddev.readthedocs.io/en/stable/users/topics/sharing/)
- [ngrok Documentation](https://ngrok.com/docs)
- [Cloudflare Tunnel Documentation](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/)