# Share Provider Redesign

## Overview

Redesign `ddev share` to support multiple tunnel providers (ngrok, cloudflared, etc.) using a script-based extensibility pattern consistent with DDEV's existing provider and custom command systems.

## Problem Statement

### Issue #7784: Capture and Expose Share URL

Currently, `ddev share` runs pre-share hooks **before** ngrok starts, so the `DDEV_SHARE_URL` environment variable is not available to hooks. Users need the public URL to:
- Register webhooks with external services
- Update DNS records
- Send notifications
- Configure SSO callbacks

**Current bug in `cmd/ddev/cmd/share.go`**:
```go
// Line 41 - PRE-SHARE HOOKS RUN TOO EARLY
err = app.ProcessHooks("pre-share")

// ... 30+ lines later ...

// Line 75 - NGROK STARTS HERE
err = ngrokCmd.Start()
```

### Strategic Context

**Issue #4365**: Consider using ngrok-go library
- Decision: CLI-based approach maintains flexibility for multiple providers
- Embedding ngrok-go would lock DDEV to ngrok-only

**Issue #6441**: Document and provide alternatives to ddev share
- Goal: Support cloudflared, expose.dev, localtunnel, etc.
- User request: Ability to customize and extend share providers

## Architecture

### Core Principle: Script-Based Providers

Follow DDEV's existing patterns:
- **Hosting providers**: `.ddev/providers/*.yaml` with `#ddev-generated`
- **Custom commands**: `.ddev/commands/*.sh` with `#ddev-generated`
- **Share providers**: `.ddev/share-providers/*.sh` with `#ddev-generated`

### Directory Structure

```
pkg/ddevapp/dotddev_assets/
└── share-providers/        # NEW - embedded in binary
    ├── ngrok.sh           # Built-in provider
    └── cloudflared.sh     # Built-in provider

.ddev/
└── share-providers/        # Written on ddev start/share
    ├── ngrok.sh           # #ddev-generated (DDEV-owned)
    ├── cloudflared.sh     # #ddev-generated (DDEV-owned)
    ├── my-ngrok.sh        # User customized (no #ddev-generated)
    └── expose.sh          # User created (no #ddev-generated)
```

### Embedding and Copy Pattern

**No new code needed** - reuse existing infrastructure:

```go
// pkg/ddevapp/assets.go - ALREADY EXISTS
//go:embed dotddev_assets/* dotddev_assets/commands/.gitattributes
var bundledAssets embed.FS

// pkg/ddevapp/assets.go - PopulateExamplesCommandsHomeadditions
// ALREADY COPIES share-providers/ automatically:
err = fileutil.CopyEmbedAssets(bundledAssets, "dotddev_assets", app.GetConfigPath(""), ...)
```

The existing `CopyEmbedAssets` function:
- Only overwrites files with `#ddev-generated`
- Preserves user modifications (files without `#ddev-generated`)
- Creates new files if missing

### Provider Script Contract

Every share provider script follows this simple contract:

**Input** (via environment variables):
- `DDEV_LOCAL_URL` - Local project URL (e.g., `http://127.0.0.1:8080`)
- `DDEV_SHARE_<PROVIDER>_ARGS` - Provider-specific arguments
- All standard DDEV environment variables

**Output**:
- **Stdout**: Public URL (first line only, captured by DDEV)
- **Stderr**: Logs, status messages (passed through to user)

**Lifecycle**:
1. Start tunnel process
2. Capture public URL (via API, stdout, file, etc.)
3. Print URL to stdout
4. Keep running until SIGINT/SIGTERM

**Example: `pkg/ddevapp/dotddev_assets/share-providers/ngrok.sh`**

```bash
#!/bin/bash
#ddev-generated

# ngrok share provider for DDEV
#
# Remove the '#ddev-generated' line above to take ownership of this file
# and prevent DDEV from overwriting it on updates.

set -euo pipefail

# Start ngrok in background
ngrok http "$DDEV_LOCAL_URL" ${DDEV_SHARE_NGROK_ARGS:-} &
NGROK_PID=$!

# Poll ngrok API for public URL (retry for 30 seconds)
for i in {1..30}; do
  URL=$(curl -s http://localhost:4040/api/tunnels 2>/dev/null | \
        jq -r '.tunnels[0].public_url' 2>/dev/null || echo "")

  if [[ -n "$URL" && "$URL" != "null" ]]; then
    echo "$URL"  # Output to stdout - captured by DDEV
    break
  fi
  sleep 1
done

# Wait for ngrok to exit (user hits Ctrl+C or process terminates)
wait $NGROK_PID
```

**Example: `pkg/ddevapp/dotddev_assets/share-providers/cloudflared.sh`**

```bash
#!/bin/bash
#ddev-generated

# cloudflared share provider for DDEV
#
# Remove the '#ddev-generated' line above to take ownership of this file.

set -euo pipefail

# Start cloudflared in background
cloudflared tunnel --url "$DDEV_LOCAL_URL" ${DDEV_SHARE_CLOUDFLARED_ARGS:-} &
CF_PID=$!

# cloudflared uses random API port in range 20241-20245
# Poll all possible ports for public URL
for i in {1..30}; do
  for PORT in {20241..20245}; do
    HOSTNAME=$(curl -s "http://127.0.0.1:$PORT/quicktunnel" 2>/dev/null | \
               jq -r '.hostname' 2>/dev/null || echo "")

    if [[ -n "$HOSTNAME" && "$HOSTNAME" != "null" ]]; then
      echo "https://$HOSTNAME"  # Add https prefix and output
      break 2
    fi
  done
  sleep 1
done

# Wait for cloudflared to exit
wait $CF_PID
```

### Configuration

**In `.ddev/config.yaml`**:
```yaml
# Default provider for this project
share_default_provider: ngrok

# Provider-specific configuration (converted to environment variables)
share_ngrok_args: "--basic-auth username:pass1234"
share_cloudflared_args: ""
```

**Via `ddev config` command**:
```bash
# Set default provider
ddev config --share-provider=cloudflared

# Configure provider arguments
ddev config --share-ngrok-args="--domain foo.ngrok-free.app"
```

**Runtime override**:
```bash
# Use different provider one-time (overrides config.yaml)
ddev share --provider=cloudflared
ddev share --provider=my-custom-tunnel
```

### Updated `share.go` Implementation

**High-level flow**:
```go
func (cmd *ShareCmd) Run(args []string) error {
    app := getApp()

    // 1. Determine which provider to use
    provider := getProviderName(app, cmd.Flags())  // From flag, config, or default

    // 2. Find provider script
    scriptPath := app.GetConfigPath("share-providers", provider+".sh")
    if !fileutil.FileExists(scriptPath) {
        return fmt.Errorf("share provider '%s' not found in .ddev/share-providers/", provider)
    }

    // 3. Execute provider script with environment
    cmd := exec.Command(scriptPath)
    cmd.Env = buildProviderEnv(app, provider)

    // 4. Capture public URL from stdout (first line)
    stdout, _ := cmd.StdoutPipe()
    cmd.Stderr = os.Stderr  // Pass through logs to user
    cmd.Start()

    scanner := bufio.NewScanner(stdout)
    if !scanner.Scan() {
        return fmt.Errorf("provider '%s' did not output a URL", provider)
    }
    publicURL := strings.TrimSpace(scanner.Text())

    // 5. Set DDEV_SHARE_URL environment variable
    os.Setenv("DDEV_SHARE_URL", publicURL)
    util.Success("Tunnel URL: %s", publicURL)

    // 6. NOW run pre-share hooks (with DDEV_SHARE_URL available)
    if err := app.ProcessHooks("pre-share"); err != nil {
        util.Warning("pre-share hooks failed: %v", err)
    }

    // 7. Wait for provider script to exit (signal handling)
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    done := make(chan error, 1)
    go func() { done <- cmd.Wait() }()

    select {
    case err := <-done:
        // Provider exited normally
    case <-sigChan:
        // User hit Ctrl+C - kill provider
        cmd.Process.Kill()
        <-done
    }

    // 8. Run post-share hooks
    if err := app.ProcessHooks("post-share"); err != nil {
        util.Warning("post-share hooks failed: %v", err)
    }

    return nil
}

func buildProviderEnv(app *DdevApp, provider string) []string {
    env := os.Environ()
    env = append(env, fmt.Sprintf("DDEV_LOCAL_URL=%s", app.GetWebContainerDirectHTTPURL()))

    // Add provider-specific args from config
    if provider == "ngrok" && app.NgrokArgs != "" {
        env = append(env, fmt.Sprintf("DDEV_SHARE_NGROK_ARGS=%s", app.NgrokArgs))
    }
    if provider == "cloudflared" && app.CloudflaredArgs != "" {
        env = append(env, fmt.Sprintf("DDEV_SHARE_CLOUDFLARED_ARGS=%s", app.CloudflaredArgs))
    }

    return env
}
```

## User Customization Examples

### Example 1: Customize Built-in Provider

```bash
# Edit .ddev/share-providers/ngrok.sh
# Remove '#ddev-generated' line
# Add custom logic (e.g., parse different API endpoint, add logging)
# DDEV will never overwrite this file again
```

### Example 2: Create Custom Variant

```bash
# Copy and customize
cp .ddev/share-providers/ngrok.sh .ddev/share-providers/my-ngrok.sh

# Edit my-ngrok.sh:
# - Remove '#ddev-generated'
# - Change API polling logic
# - Add custom environment variables

# Use with:
ddev share --provider=my-ngrok

# Or set as default:
ddev config --share-provider=my-ngrok
```

### Example 3: Add New Provider

```bash
# Create .ddev/share-providers/expose.sh
cat > .ddev/share-providers/expose.sh << 'EOF'
#!/bin/bash
set -euo pipefail

# Start expose.dev tunnel
expose share "$DDEV_LOCAL_URL" ${DDEV_SHARE_EXPOSE_ARGS:-} &
EXPOSE_PID=$!

# Parse URL from stdout
while IFS= read -r line; do
  if [[ $line =~ https://[a-z0-9-]+\.expose\.dev ]]; then
    echo "${BASH_REMATCH[0]}"
    break
  fi
done < <(expose logs --follow 2>&1)

wait $EXPOSE_PID
EOF

chmod +x .ddev/share-providers/expose.sh

# Use it:
ddev share --provider=expose
```

## Migration and Backward Compatibility

### For Existing Users

- **No breaking changes**: `ddev share` continues to use ngrok by default
- **Existing config**: `ngrok_args` in config.yaml still works (mapped to `DDEV_SHARE_NGROK_ARGS`)
- **Existing flag**: `--ngrok-args` flag still works
- **First run**: On `ddev start` or `ddev share`, scripts are generated to `.ddev/share-providers/`

### Migration Path

```yaml
# OLD config.yaml (still works)
ngrok_args: "--basic-auth username:pass1234"

# NEW config.yaml (preferred)
share_default_provider: ngrok
share_ngrok_args: "--basic-auth username:pass1234"
```

Both configurations work identically. The old `ngrok_args` is mapped to `DDEV_SHARE_NGROK_ARGS` for backward compatibility.

## Implementation Phases

### Phase 1: Core Infrastructure (Issue #7784)

**Goal**: Fix hook timing, add script-based provider support

1. Create `pkg/ddevapp/dotddev_assets/share-providers/ngrok.sh`
2. Update `cmd/ddev/cmd/share.go`:
   - Execute provider script instead of ngrok directly
   - Capture URL from stdout
   - Set `DDEV_SHARE_URL` before pre-share hooks
3. Update config structures to support `share_default_provider`
4. Add tests for script execution and URL capture

**Deliverable**: `ddev share` works with ngrok script, hooks receive `DDEV_SHARE_URL`

### Phase 2: Cloudflared Support (Issue #6441)

**Goal**: Demonstrate multi-provider capability

1. Create `pkg/ddevapp/dotddev_assets/share-providers/cloudflared.sh`
2. Add `--provider` flag to `ddev share`
3. Add `ddev config --share-provider=<name>` command
4. Update documentation with provider examples

**Deliverable**: Users can choose between ngrok and cloudflared

### Phase 3: Documentation and Polish

**Goal**: Enable community to create custom providers

1. Document provider script contract
2. Create provider development guide
3. Add example custom providers
4. Optional: `ddev share --list-providers` command

**Deliverable**: Community can easily create and share custom providers

## Benefits

1. **Transparency**: Users can inspect exactly what each provider does
2. **Extensibility**: Anyone can create a provider without Go knowledge
3. **Consistency**: Matches DDEV's existing patterns (providers, custom commands)
4. **No vendor lock-in**: Easy to add any tunnel service
5. **User control**: Copy, modify, or create providers as needed
6. **Maintainability**: Provider improvements don't require Go recompilation

## Testing Strategy

1. **Unit tests**: Provider script discovery, environment variable passing
2. **Integration tests**: Execute ngrok script, verify URL capture
3. **Mock providers**: Test scripts that output URL immediately
4. **Error cases**: Missing provider, script timeout, no URL output
5. **Signal handling**: Verify Ctrl+C properly terminates provider

## Open Questions

1. Should we support provider scripts in global `.ddev`? (e.g., `~/.ddev/share-providers/`)
2. Should we add validation for provider scripts (e.g., executable bit, shebang)?
3. Should we provide a template script for new providers?
4. Should we add provider versioning or dependency checking?

## References

- Issue #7784: Provide way to capture ngrok URL for hooks
- Issue #4365: Consider using ngrok-go library
- Issue #6441: Document and provide alternatives to ddev share
- Similar pattern: `.ddev/providers/` for hosting providers
- Similar pattern: `.ddev/commands/` for custom commands