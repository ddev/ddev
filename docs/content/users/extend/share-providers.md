---
search:
  boost: .5
---

# Custom Share Providers

You can customize the built-in providers or create your own share providers in `.ddev/share-providers/`.

## Customizing Built-in Providers

### Take Ownership of a Built-in Provider

1. Edit the provider script (e.g., `.ddev/share-providers/ngrok.sh`)
2. Remove the `#ddev-generated` line at the top
3. Make your changes
4. DDEV will never overwrite this file again

### Create a Custom Variant

```bash
# Copy built-in provider
cp .ddev/share-providers/ngrok.sh .ddev/share-providers/my-ngrok.sh

# Edit my-ngrok.sh:
# - Remove '#ddev-generated' line
# - Customize as needed

# Use your variant
ddev share --provider=my-ngrok
```

## Creating a New Provider

Create a new executable script in `.ddev/share-providers/`:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Start your tunnel tool
mytunnel http "$DDEV_LOCAL_URL" &
TUNNEL_PID=$!

trap "kill $TUNNEL_PID 2>/dev/null || true" EXIT

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

| Variable | Description |
|----------|-------------|
| `DDEV_LOCAL_URL` | Local URL to tunnel (e.g., `http://127.0.0.1:8080`) |
| `DDEV_SHARE_ARGS` | Provider-specific arguments (optional) |

All standard DDEV environment variables are also available.

### Output

* **`stdout`**: Public URL (first line only - captured by DDEV)
* **`stderr`**: Logs, status messages (passed through to user)

### Lifecycle

1. Validate tool is installed
2. Validate required environment variables
3. Start tunnel process in background
4. Capture public URL (via API, stdout, file, etc.)
5. Output URL to stdout
6. Wait for tunnel process to exit

### Signal Handling

Providers must handle `SIGINT` (Ctrl+C) and `SIGTERM` gracefully. Use `trap` to cleanup background processes:

```bash
cleanup() {
    if kill -0 $PID 2>/dev/null; then
        kill $PID 2>/dev/null || true
    fi
}
trap cleanup EXIT
```

## Hooks Integration

After the tunnel URL is captured, DDEV sets the `DDEV_SHARE_URL` environment variable and runs pre-share hooks. This allows you to alter projects as needed (like WordPress `ddev wp search-replace`, for example).

Example `.ddev/config.share.yaml`:

```yaml
hooks:
  pre-share:
    - exec: |
        echo "Tunnel URL: ${DDEV_SHARE_URL}"
        wp search-replace ${DDEV_PRIMARY_URL} ${DDEV_SHARE_URL}
```

## Troubleshooting Custom Providers

**Provider not found:**

```text
Error: share provider 'foo' not found
```

Check that `.ddev/share-providers/foo.sh` exists and is executable:

```bash
ls -la .ddev/share-providers/
chmod +x .ddev/share-providers/foo.sh
```

**Provider outputs no URL:**

```text
Error: provider 'ngrok' did not output a URL
```

Common causes: tool not installed, authentication required, no internet. Debug by running the provider directly:

```bash
export DDEV_LOCAL_URL=http://127.0.0.1:8080
.ddev/share-providers/ngrok.sh
```
