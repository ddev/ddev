# Share Provider Implementation Plan

## Overview

Step-by-step implementation plan for the share provider redesign. This plan follows the existing DDEV patterns for embedded assets and `#ddev-generated` files.

## Phase 1: Core Infrastructure

### Task 1.1: Create Provider Script Directory Structure

**Files to create**:
```
pkg/ddevapp/dotddev_assets/share-providers/
├── README.md
├── ngrok.sh
└── cloudflared.sh
```

**Action**: Create directory and initial files
- Location: `pkg/ddevapp/dotddev_assets/share-providers/`
- No changes to `assets.go` needed - existing `//go:embed dotddev_assets/*` already includes subdirectories
- Existing `CopyEmbedAssets` call will automatically copy this directory

**Dependencies**: None

**Testing**:
- Verify directory is embedded: `go build && strings .gotmp/bin/*/ddev | grep "share-providers"`
- After build, check files copied to test project: `ls .ddev/share-providers/`

---

### Task 1.2: Write ngrok.sh Provider Script

**File**: `pkg/ddevapp/dotddev_assets/share-providers/ngrok.sh`

**Content**:
```bash
#!/bin/bash
#ddev-generated

# ngrok share provider for DDEV
# Documentation: https://ddev.readthedocs.io/en/stable/users/topics/sharing/
#
# To customize: remove the '#ddev-generated' line above and edit as needed.
# To create a variant: copy to a new file like my-ngrok.sh

set -euo pipefail

# Validate ngrok is installed
if ! command -v ngrok &> /dev/null; then
    echo "Error: ngrok not found in PATH. Install from https://ngrok.com/download" >&2
    exit 1
fi

# Validate required environment variables
if [[ -z "${DDEV_LOCAL_URL:-}" ]]; then
    echo "Error: DDEV_LOCAL_URL not set" >&2
    exit 1
fi

# Start ngrok in background
ngrok http "$DDEV_LOCAL_URL" ${DDEV_SHARE_NGROK_ARGS:-} &
NGROK_PID=$!

# Function to cleanup on exit
cleanup() {
    if kill -0 $NGROK_PID 2>/dev/null; then
        kill $NGROK_PID 2>/dev/null || true
    fi
}
trap cleanup EXIT

# Poll ngrok API for public URL (30 second timeout)
echo "Starting ngrok tunnel..." >&2
URL=""
for i in {1..30}; do
    URL=$(curl -s http://localhost:4040/api/tunnels 2>/dev/null | \
          jq -r '.tunnels[0].public_url' 2>/dev/null || echo "")

    if [[ -n "$URL" && "$URL" != "null" ]]; then
        echo "$URL"  # Output to stdout - CRITICAL: This is captured by DDEV
        break
    fi
    sleep 1
done

if [[ -z "$URL" || "$URL" == "null" ]]; then
    echo "Error: Failed to get ngrok URL after 30 seconds" >&2
    exit 1
fi

# Wait for ngrok to exit
wait $NGROK_PID
```

**Key implementation notes**:
- Must be executable: `chmod +x ngrok.sh`
- First line must be exactly `#!/bin/bash`
- Second line must be exactly `#ddev-generated`
- URL output to stdout must be first non-stderr output
- All other output goes to stderr with `>&2`

**Dependencies**: None

**Testing**:
```bash
# Manual test
export DDEV_LOCAL_URL=http://127.0.0.1:8080
export DDEV_SHARE_NGROK_ARGS=""
./ngrok.sh
```

---

### Task 1.3: Write cloudflared.sh Provider Script

**File**: `pkg/ddevapp/dotddev_assets/share-providers/cloudflared.sh`

**Content**:
```bash
#!/bin/bash
#ddev-generated

# cloudflared share provider for DDEV
# Documentation: https://ddev.readthedocs.io/en/stable/users/topics/sharing/
#
# To customize: remove the '#ddev-generated' line above and edit as needed.

set -euo pipefail

# Validate cloudflared is installed
if ! command -v cloudflared &> /dev/null; then
    echo "Error: cloudflared not found in PATH. Install from https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation" >&2
    exit 1
fi

# Validate required environment variables
if [[ -z "${DDEV_LOCAL_URL:-}" ]]; then
    echo "Error: DDEV_LOCAL_URL not set" >&2
    exit 1
fi

# Start cloudflared in background
cloudflared tunnel --url "$DDEV_LOCAL_URL" ${DDEV_SHARE_CLOUDFLARED_ARGS:-} &
CF_PID=$!

# Function to cleanup on exit
cleanup() {
    if kill -0 $CF_PID 2>/dev/null; then
        kill $CF_PID 2>/dev/null || true
    fi
}
trap cleanup EXIT

# cloudflared exposes metrics API on random port 20241-20245
# Poll all possible ports for tunnel URL
echo "Starting cloudflared tunnel..." >&2
HOSTNAME=""
for i in {1..30}; do
    for PORT in {20241..20245}; do
        HOSTNAME=$(curl -s "http://127.0.0.1:$PORT/quicktunnel" 2>/dev/null | \
                   jq -r '.hostname' 2>/dev/null || echo "")

        if [[ -n "$HOSTNAME" && "$HOSTNAME" != "null" ]]; then
            echo "https://$HOSTNAME"  # Output to stdout - CRITICAL
            break 2
        fi
    done
    sleep 1
done

if [[ -z "$HOSTNAME" || "$HOSTNAME" == "null" ]]; then
    echo "Error: Failed to get cloudflared URL after 30 seconds" >&2
    exit 1
fi

# Wait for cloudflared to exit
wait $CF_PID
```

**Dependencies**: None

**Testing**:
```bash
# Manual test
export DDEV_LOCAL_URL=http://127.0.0.1:8080
./cloudflared.sh
```

---

### Task 1.4: Write README.md for Share Providers

**File**: `pkg/ddevapp/dotddev_assets/share-providers/README.md`

**Content**: Basic documentation about share providers, customization, and script contract

**Dependencies**: None

---

### Task 1.5: Add Config Fields for Share Providers

**File**: `pkg/ddevapp/types.go`

**Changes needed**:
```go
type DdevApp struct {
    // ... existing fields ...

    // Share provider configuration
    ShareDefaultProvider  string `yaml:"share_default_provider,omitempty" json:"share_default_provider,omitempty"`
    ShareNgrokArgs       string `yaml:"share_ngrok_args,omitempty" json:"share_ngrok_args,omitempty"`
    ShareCloudflaredArgs string `yaml:"share_cloudflared_args,omitempty" json:"share_cloudflared_args,omitempty"`

    // Deprecated but maintained for backward compatibility
    NgrokArgs string `yaml:"ngrok_args,omitempty" json:"ngrok_args,omitempty"`
}
```

**File**: `pkg/ddevapp/config.go`

**Add config initialization**:
```go
// In NewApp() or similar initialization function
func (app *DdevApp) initializeDefaults() {
    // ... existing defaults ...

    // Share provider defaults
    if app.ShareDefaultProvider == "" {
        app.ShareDefaultProvider = "ngrok"
    }

    // Backward compatibility: map old NgrokArgs to new ShareNgrokArgs
    if app.NgrokArgs != "" && app.ShareNgrokArgs == "" {
        app.ShareNgrokArgs = app.NgrokArgs
    }
}
```

**Dependencies**: None

**Testing**:
- Create test project with `share_default_provider: cloudflared`
- Verify config loads correctly
- Test backward compat with `ngrok_args`

---

### Task 1.6: Add Helper Functions for Provider Management

**File**: `pkg/ddevapp/share_providers.go` (NEW)

**Content**:
```go
package ddevapp

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/ddev/ddev/pkg/fileutil"
)

// GetShareProviderScript returns the absolute path to a share provider script
func (app *DdevApp) GetShareProviderScript(providerName string) (string, error) {
    scriptPath := app.GetConfigPath("share-providers", providerName+".sh")

    if !fileutil.FileExists(scriptPath) {
        return "", fmt.Errorf("share provider '%s' not found at %s", providerName, scriptPath)
    }

    // Check if executable
    info, err := os.Stat(scriptPath)
    if err != nil {
        return "", err
    }
    if info.Mode()&0111 == 0 {
        return "", fmt.Errorf("share provider '%s' is not executable (chmod +x %s)", providerName, scriptPath)
    }

    return scriptPath, nil
}

// ListShareProviders returns all available share provider names
func (app *DdevApp) ListShareProviders() ([]string, error) {
    providerDir := app.GetConfigPath("share-providers")
    if !fileutil.IsDirectory(providerDir) {
        return []string{}, nil
    }

    entries, err := os.ReadDir(providerDir)
    if err != nil {
        return nil, err
    }

    var providers []string
    for _, entry := range entries {
        if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sh" {
            name := strings.TrimSuffix(entry.Name(), ".sh")
            providers = append(providers, name)
        }
    }

    return providers, nil
}

// GetShareProviderEnvironment builds environment variables for provider script
func (app *DdevApp) GetShareProviderEnvironment(providerName string) []string {
    env := os.Environ()

    // Add DDEV_LOCAL_URL
    localURL := app.GetWebContainerDirectHTTPURL()
    env = append(env, fmt.Sprintf("DDEV_LOCAL_URL=%s", localURL))

    // Add provider-specific args
    switch providerName {
    case "ngrok":
        args := app.ShareNgrokArgs
        if args == "" && app.NgrokArgs != "" {
            args = app.NgrokArgs // Backward compatibility
        }
        if args != "" {
            env = append(env, fmt.Sprintf("DDEV_SHARE_NGROK_ARGS=%s", args))
        }
    case "cloudflared":
        if app.ShareCloudflaredArgs != "" {
            env = append(env, fmt.Sprintf("DDEV_SHARE_CLOUDFLARED_ARGS=%s", app.ShareCloudflaredArgs))
        }
    }

    return env
}
```

**Dependencies**: Task 1.5 (config fields)

**Testing**: Unit tests for each function

---

### Task 1.7: Refactor share.go to Use Provider Scripts

**File**: `cmd/ddev/cmd/share.go`

**Major refactoring required**. Current structure:
- Lines 40-44: Pre-share hooks (TOO EARLY)
- Lines 46-49: Find ngrok binary
- Lines 50-120: Execute ngrok, handle signals, wait for exit
- Lines 122-126: Post-share hooks

**New structure**:
```go
// DdevShareCommand contains the "ddev share" command
var DdevShareCommand = &cobra.Command{
    Use:   "share [project]",
    Short: "Share project on the internet via tunnel provider",
    Long:  `Share your project using a tunnel provider (ngrok, cloudflared, etc.). Configure provider with "ddev config --share-provider=<name>". Default: ngrok`,
    Example: `ddev share
ddev share --provider=cloudflared
ddev share --provider=my-custom-tunnel
ddev share myproject`,
    Run: func(cmd *cobra.Command, args []string) {
        if len(args) > 1 {
            util.Failed("Too many arguments provided. Please use 'ddev share' or 'ddev share [projectname]'")
        }

        apps, err := getRequestedProjects(args, false)
        if err != nil {
            util.Failed("Failed to describe project(s): %v", err)
        }
        app := apps[0]

        status, _ := app.SiteStatus()
        if status != ddevapp.SiteRunning {
            util.Failed("Project is not yet running. Use 'ddev start' first.")
        }

        // Determine which provider to use
        providerName, err := cmd.Flags().GetString("provider")
        if err != nil || providerName == "" {
            providerName = app.ShareDefaultProvider
            if providerName == "" {
                providerName = "ngrok" // Ultimate fallback
            }
        }

        // Get provider script path
        scriptPath, err := app.GetShareProviderScript(providerName)
        if err != nil {
            util.Failed("Share provider error: %v\nAvailable providers: %v", err, app.ListShareProviders())
        }

        // Execute provider script and capture URL
        publicURL, providerCmd, err := executeShareProvider(app, scriptPath, providerName)
        if err != nil {
            util.Failed("Failed to start share provider '%s': %v", providerName, err)
        }

        // Set DDEV_SHARE_URL environment variable
        os.Setenv("DDEV_SHARE_URL", publicURL)
        util.Success("Tunnel URL: %s", publicURL)

        // NOW run pre-share hooks (with DDEV_SHARE_URL available)
        err = app.ProcessHooks("pre-share")
        if err != nil {
            util.Warning("Failed to process pre-share hooks: %v", err)
        }

        // Wait for provider to exit (with signal handling)
        waitForProviderExit(providerCmd)

        // Process post-share hooks
        err = app.ProcessHooks("post-share")
        if err != nil {
            util.Warning("Failed to process post-share hooks: %v", err)
        }

        os.Exit(0)
    },
}

// executeShareProvider runs the provider script and captures the public URL
func executeShareProvider(app *ddevapp.DdevApp, scriptPath string, providerName string) (string, *exec.Cmd, error) {
    cmd := exec.Command(scriptPath)
    cmd.Env = app.GetShareProviderEnvironment(providerName)

    // Capture stdout for URL, pass through stderr for logs
    stdoutPipe, err := cmd.StdoutPipe()
    if err != nil {
        return "", nil, err
    }
    cmd.Stderr = os.Stderr

    // Start provider script
    if err := cmd.Start(); err != nil {
        return "", nil, fmt.Errorf("failed to execute provider script: %w", err)
    }

    // Read first line from stdout (the public URL)
    scanner := bufio.NewScanner(stdoutPipe)
    if !scanner.Scan() {
        cmd.Process.Kill()
        return "", nil, fmt.Errorf("provider '%s' did not output a URL", providerName)
    }

    publicURL := strings.TrimSpace(scanner.Text())
    if publicURL == "" {
        cmd.Process.Kill()
        return "", nil, fmt.Errorf("provider '%s' output empty URL", providerName)
    }

    // Validate URL format
    if !strings.HasPrefix(publicURL, "http://") && !strings.HasPrefix(publicURL, "https://") {
        cmd.Process.Kill()
        return "", nil, fmt.Errorf("provider '%s' output invalid URL: %s", providerName, publicURL)
    }

    return publicURL, cmd, nil
}

// waitForProviderExit handles signal forwarding and waits for provider to exit
func waitForProviderExit(cmd *exec.Cmd) {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    done := make(chan error, 1)
    go func() {
        done <- cmd.Wait()
    }()

    select {
    case err := <-done:
        // Provider exited normally
        if err != nil {
            util.Warning("Provider exited with error: %v", err)
        }
    case <-sigChan:
        // Signal received (Ctrl+C) - kill provider
        if cmd.Process != nil {
            _ = cmd.Process.Kill()
        }
        <-done
    }
}

func init() {
    RootCmd.AddCommand(DdevShareCommand)
    DdevShareCommand.Flags().String("provider", "", "Share provider to use (ngrok, cloudflared, etc.)")

    // Keep --ngrok-args for backward compatibility
    DdevShareCommand.Flags().String("ngrok-args", "", "Deprecated: use 'ddev config --share-ngrok-args' instead")
}
```

**Dependencies**: Tasks 1.5, 1.6

**Testing**:
- Manual test with ngrok
- Test with `--provider` flag
- Test backward compat with `--ngrok-args`
- Test signal handling (Ctrl+C)

---

### Task 1.8: Update `ddev config` Command

**File**: `cmd/ddev/cmd/config.go`

**Add flags**:
```go
configCmd.Flags().String("share-provider", "", "Set default share provider (ngrok, cloudflared, etc.)")
configCmd.Flags().String("share-ngrok-args", "", "Arguments to pass to ngrok provider")
configCmd.Flags().String("share-cloudflared-args", "", "Arguments to pass to cloudflared provider")
```

**Add flag processing in Run function**:
```go
if configCmd.Flags().Changed("share-provider") {
    provider, _ := configCmd.Flags().GetString("share-provider")
    app.ShareDefaultProvider = provider
}

if configCmd.Flags().Changed("share-ngrok-args") {
    args, _ := configCmd.Flags().GetString("share-ngrok-args")
    app.ShareNgrokArgs = args
}

if configCmd.Flags().Changed("share-cloudflared-args") {
    args, _ := configCmd.Flags().GetString("share-cloudflared-args")
    app.ShareCloudflaredArgs = args
}
```

**Dependencies**: Task 1.5

**Testing**:
```bash
ddev config --share-provider=cloudflared
grep share_default_provider .ddev/config.yaml

ddev config --share-ngrok-args="--domain foo.ngrok-free.app"
grep share_ngrok_args .ddev/config.yaml
```

---

## Phase 2: Testing

### Task 2.1: Unit Tests for Provider Discovery

**File**: `pkg/ddevapp/share_providers_test.go` (NEW)

**Tests**:
- `TestGetShareProviderScript` - find existing scripts
- `TestGetShareProviderScript_NotFound` - error on missing script
- `TestGetShareProviderScript_NotExecutable` - error if not executable
- `TestListShareProviders` - list all available providers
- `TestGetShareProviderEnvironment` - verify env vars set correctly
- `TestGetShareProviderEnvironment_BackwardCompat` - test NgrokArgs fallback

**Dependencies**: Task 1.6

---

### Task 2.2: Integration Tests with Mock Provider

**File**: `cmd/ddev/cmd/share_test.go` (NEW or update existing)

**Create mock provider for testing**:
```bash
# testdata/mock-share-provider.sh
#!/bin/bash
#ddev-generated
echo "https://mock-tunnel-12345.example.com"
sleep 10  # Simulate long-running process
```

**Tests**:
- `TestShareCommand_MockProvider` - execute mock provider, verify URL captured
- `TestShareCommand_HookTiming` - verify pre-share hook receives DDEV_SHARE_URL
- `TestShareCommand_SignalHandling` - send SIGINT, verify cleanup
- `TestShareCommand_ProviderNotFound` - error message for missing provider
- `TestShareCommand_InvalidURL` - error if provider outputs invalid URL
- `TestShareCommand_BackwardCompat` - verify old --ngrok-args flag still works

**Dependencies**: Task 1.7

---

### Task 2.3: Manual Testing Checklist

Create testing checklist:
- [ ] Build ddev binary
- [ ] Create test project
- [ ] Verify `.ddev/share-providers/` created on start
- [ ] Verify ngrok.sh has `#ddev-generated`
- [ ] Test `ddev share` with ngrok
- [ ] Verify DDEV_SHARE_URL in pre-share hook
- [ ] Test `ddev share --provider=cloudflared`
- [ ] Test `ddev config --share-provider=cloudflared`
- [ ] Verify config.yaml updated
- [ ] Remove `#ddev-generated`, edit ngrok.sh
- [ ] Verify DDEV doesn't overwrite modified script
- [ ] Test Ctrl+C signal handling
- [ ] Test backward compat with old `ngrok_args`

---

## Phase 3: Documentation

### Task 3.1: Update Share Documentation

**File**: `docs/content/users/topics/sharing.md`

**Sections to add/update**:
1. Share provider overview
2. Built-in providers (ngrok, cloudflared)
3. Configuring default provider
4. Provider-specific configuration
5. Creating custom providers
6. Provider script contract
7. Examples

**Dependencies**: Phase 1 complete

---

### Task 3.2: Create Provider Development Guide

**File**: `docs/content/developers/share-providers.md` (NEW)

**Content**:
- Provider script contract specification
- Environment variables available
- stdout/stderr conventions
- Signal handling requirements
- Error handling best practices
- Example provider implementations
- Testing custom providers

**Dependencies**: Phase 1 complete

---

### Task 3.3: Update CHANGELOG

**File**: `CHANGELOG.md` or release notes

**Entry**:
```markdown
### Added
- Share provider system supporting multiple tunnel services (#7784, #6441)
- Built-in cloudflared share provider as alternative to ngrok
- `ddev config --share-provider=<name>` to set default provider
- `ddev share --provider=<name>` to use specific provider
- Custom share provider support via `.ddev/share-providers/*.sh`

### Fixed
- Share hooks now receive DDEV_SHARE_URL environment variable (#7784)

### Changed
- `ngrok_args` config option deprecated in favor of `share_ngrok_args`
- Pre-share hooks now run after tunnel URL is available (breaking change for early hook users)
```

---

## Testing Matrix

| Test Case | Expected Result |
|-----------|----------------|
| `ddev share` (default ngrok) | Starts ngrok, captures URL, hooks receive DDEV_SHARE_URL |
| `ddev share --provider=cloudflared` | Starts cloudflared, captures URL |
| `ddev share --provider=custom` | Starts custom provider from `.ddev/share-providers/` |
| `ddev config --share-provider=cloudflared` | Updates config.yaml, persists setting |
| Old `ngrok_args` in config | Still works, maps to DDEV_SHARE_NGROK_ARGS |
| Old `--ngrok-args` flag | Still works, sets environment variable |
| Provider not found | Clear error message with available providers |
| Provider not executable | Error with chmod suggestion |
| Provider outputs no URL | Error after timeout |
| Ctrl+C during share | Provider killed cleanly, post-share hooks run |
| Edit script, remove #ddev-generated | DDEV never overwrites file again |
| Copy script to new name | New provider available immediately |

---

## Rollout Strategy

### Pre-release Testing
1. Create branch from main
2. Implement Phase 1 tasks
3. Test with ngrok and cloudflared on macOS, Linux, Windows
4. Get feedback from early testers
5. Iterate on script interface if needed

### Release
1. Merge to main
2. Include in next DDEV release
3. Blog post about new share provider system
4. Example custom providers in community

### Post-release
1. Monitor for issues
2. Collect feedback on provider script interface
3. Consider adding more built-in providers based on demand
4. Update examples based on community providers

---

## Risk Mitigation

### Risk: Breaking changes for existing users
**Mitigation**:
- Maintain backward compatibility with `ngrok_args`
- Default provider remains ngrok
- Old `--ngrok-args` flag continues to work

### Risk: Provider scripts fail silently
**Mitigation**:
- Timeout after 30 seconds
- Validate URL format
- Check script is executable
- Provide clear error messages

### Risk: Security concerns with arbitrary scripts
**Mitigation**:
- Scripts in `.ddev/share-providers/` are user-controlled (same trust model as custom commands)
- Built-in scripts marked with `#ddev-generated`
- Document security considerations

### Risk: Platform-specific script issues
**Mitigation**:
- Use bash (available on macOS, Linux, Windows via Git Bash)
- Avoid bashisms, stick to POSIX where possible
- Test on all platforms
- Document platform requirements

---

## Success Criteria

- [ ] `ddev share` works with default ngrok provider
- [ ] `ddev share --provider=cloudflared` works
- [ ] Pre-share hooks receive `DDEV_SHARE_URL`
- [ ] Custom providers can be added to `.ddev/share-providers/`
- [ ] Backward compatibility maintained for existing configs
- [ ] All tests pass on macOS, Linux, Windows
- [ ] Documentation complete and reviewed
- [ ] Zero regression bugs reported in first week after release

---

## Timeline Estimate

- **Phase 1 (Core)**: 2-3 days
  - Day 1: Tasks 1.1-1.4 (scripts)
  - Day 2: Tasks 1.5-1.6 (config, helpers)
  - Day 3: Tasks 1.7-1.8 (refactor share.go, config command)

- **Phase 2 (Testing)**: 1-2 days
  - Unit tests, integration tests, manual testing

- **Phase 3 (Docs)**: 1 day
  - User documentation, developer guide, changelog

**Total**: 4-6 days for complete implementation

---

## Open Questions

1. **Script location precedence**: Should we check global `.ddev` before project `.ddev`?
   - Current: Only check `.ddev/share-providers/`
   - Alternative: Check `~/.ddev/share-providers/` first (like custom commands)

2. **Provider validation**: Should we validate provider scripts on `ddev config`?
   - Pro: Early error detection
   - Con: Requires tool to be installed even if not using that provider

3. **Provider versioning**: Should providers declare DDEV version compatibility?
   - Not needed initially, but could add `# ddev-version: >=1.23.0` metadata

4. **Windows support**: Git Bash should work, but needs testing
   - Alternative: Provide `.bat` or `.ps1` equivalents

5. **Provider discovery command**: Should we add `ddev share --list-providers`?
   - Would be helpful for discoverability
   - Easy to add later if requested