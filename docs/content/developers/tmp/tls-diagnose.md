# Manual Testing Guide: `ddev utility tls-diagnose`

This guide is written for Claude running on each target OS. It covers what to verify, how to deliberately break things to test failure paths, and what healthy vs. broken output looks like.

The command is in PR ddev/ddev#8259, branch `20260328_rfay_tls_diagnose`.

## Prerequisites on All Platforms

Build the branch binary first with `make`

All examples below use `ddev` as a shorthand — substitute the full path to the built binary.

---

## macOS

### Baseline (everything healthy)

```bash
ddev utility tls-diagnose
```

Expected:

- `mkcert Installation`: mkcert found, CAROOT shown, `rootCA.pem` and `rootCA-key.pem` found
- `OS Trust Store`: "CA already installed in system trust store"
- `Firefox`: No Firefox detected, or warning about Firefox requiring manual CA import if Firefox.app is present
- `Certificate Files`: Default cert valid, project cert valid (if in a project directory)
- `Live Connectivity`: TLS verified (if project is running)
- `Summary`: "No issues found"
- Exit code 0

Run from a started project directory to get the live connectivity section:

```bash
cd ~/path/to/project
ddev start
ddev utility tls-diagnose
```

### Test: mkcert not in PATH

```bash
# Move mkcert out of PATH temporarily
sudo mv $(which mkcert) /tmp/mkcert-bak

ddev utility tls-diagnose
# Expected: red ✗ "mkcert not found in PATH" and install instructions
# Expected: exit code 1

# Restore
sudo mv /tmp/mkcert-bak $(dirname $(which mkcert 2>/dev/null || echo /usr/local/bin))/mkcert
```

### Test: CA not installed in OS trust store

```bash
mkcert -uninstall
ddev utility tls-diagnose
# Expected: mkcert -install section shows failure or "not installed"
# Expected: exit code 1

# Restore
mkcert -install
```

### Test: rootCA.pem deleted

```bash
CAROOT=$(mkcert -CAROOT)
mv "$CAROOT/rootCA.pem" "$CAROOT/rootCA.pem.bak"

ddev utility tls-diagnose
# Expected: red ✗ "rootCA.pem not found in CAROOT"
# Expected: Certificate Files section skipped or fails gracefully
# Expected: exit code 1

# Restore
mv "$CAROOT/rootCA.pem.bak" "$CAROOT/rootCA.pem"
```

### Test: CAROOT env var mismatch

```bash
CAROOT=/tmp/wrong ddev utility tls-diagnose
# Expected: ⚠ "$CAROOT env var (/tmp/wrong) differs from mkcert -CAROOT"
# Expected: exit code 1
```

### Test: Firefox present

If Firefox.app is in /Applications:

```bash
ddev utility tls-diagnose
# Expected: ⚠ "Firefox detected — may not use macOS Keychain..."
# Expected: Link to configuring-browsers docs
# NOTE: This is a warning, not a hard failure preventing browsing —
#       regular Firefox on macOS typically uses the system keychain.
#       Firefox Nightly and Developer Edition definitely need manual import.
```

If Firefox is not installed, install it temporarily or just confirm the "✓ No Firefox variants detected" line appears.

### Test: Certificate file expired

This requires either waiting or manipulating system time — easiest to test via the unit tests in `utility-tls-diagnose_test.go` (`TestCheckCertFileExpired`).

For a real integration test, you can swap in an expired cert:

```bash
CAROOT=$(mkcert -CAROOT)
PROJECT_CERT=~/.ddev/traefik/certs/default_cert.crt

cp "$PROJECT_CERT" "$PROJECT_CERT.bak"

# Generate an expired cert using openssl or rely on the unit tests instead
# (The unit tests cover this more cleanly)

# Restore
cp "$PROJECT_CERT.bak" "$PROJECT_CERT"
```

---

## Linux (native, non-WSL2)

### Baseline

```bash
ddev utility tls-diagnose
```

Expected sections identical to macOS, except:

- `OS Trust Store` will also report certutil status
- `WSL2 Configuration` section is absent

### Test: certutil not installed (affects Firefox NSS)

```bash
# Check if certutil is installed
which certutil

# If installed, temporarily rename it:
sudo mv $(which certutil) /tmp/certutil-bak

ddev utility tls-diagnose
# Expected in OS Trust Store: ⚠ "certutil not found — Firefox will not trust DDEV certificates"
# Expected: Install instructions for libnss3-tools
# Expected: exit code 1

# Restore
sudo mv /tmp/certutil-bak $(which certutil 2>/dev/null || echo /usr/bin/certutil)
```

### Test: snap Firefox

If snap is available:

```bash
snap install firefox   # if not already installed

ddev utility tls-diagnose
# Expected: ⚠ "Firefox (snap) detected — snap Firefox uses its own NSS database"
# Expected: instructions to manually import CA in Firefox

snap remove firefox    # if you installed it just for testing
```

### Test: CA not in trust store

```bash
mkcert -uninstall
ddev utility tls-diagnose
# Expected: mkcert -install failure or "not installed" message
# Expected: exit code 1

mkcert -install
```

---

## WSL2 (Ubuntu inside Windows)

This is the most complex environment. The goal of `tls-diagnose` is to diagnose why Windows Chrome/Edge don't trust DDEV certs even when things look fine inside WSL2.

### Baseline (fully configured WSL2)

A healthy WSL2 setup has all of:
- `$CAROOT` set to the Windows mkcert CAROOT path (e.g. `/mnt/c/Users/<user>/AppData/Local/mkcert`)
- `CAROOT` listed in `$WSLENV`
- `mkcert.exe` installed on Windows
- mkcert CA in Windows certificate store (installed via `mkcert -install` in PowerShell)
- WSL2-side and Windows-side CA fingerprints match

```bash
ddev utility tls-diagnose
# Expected: All checks green in WSL2 Configuration section
# Expected: Live Connectivity checks both Linux-side tls.Dial AND PowerShell Invoke-WebRequest
```

### Test: CAROOT not set

```bash
unset CAROOT
ddev utility tls-diagnose
# Expected: red ✗ "$CAROOT env var is not set in WSL2"
# Expected: Instructions to set it in PowerShell and restart WSL2
# Expected: exit code 1

# Restore: open a new WSL2 terminal (WSLENV propagation restores it)
```

### Test: CAROOT points to Linux path (not Windows filesystem)

```bash
export CAROOT=~/.local/share/mkcert
ddev utility tls-diagnose
# Expected: ⚠ "$CAROOT (/home/...) does not point to Windows filesystem (/mnt/...)"
# Expected: exit code 1
```

### Test: CAROOT not in WSLENV

In Windows PowerShell:

```powershell
# Save current value
$oldWSLENV = [System.Environment]::GetEnvironmentVariable("WSLENV", "User")

# Remove CAROOT from WSLENV
[System.Environment]::SetEnvironmentVariable("WSLENV", "", "User")
wsl --shutdown
# Open a new WSL2 terminal, then:
```

```bash
ddev utility tls-diagnose
# Expected: red ✗ "WSLENV does not contain CAROOT"
# Expected: exit code 1
```

```powershell
# Restore in PowerShell
[System.Environment]::SetEnvironmentVariable("WSLENV", $oldWSLENV, "User")
wsl --shutdown
```

### Test: mkcert CA not in Windows certificate store

In Windows PowerShell:

```powershell
# Find and remove the mkcert CA from the Windows cert store
$cert = Get-ChildItem Cert:\CurrentUser\Root | Where-Object { $_.Subject -like "*mkcert*" }
if ($cert) { Remove-Item $cert.PSPath }
$cert = Get-ChildItem Cert:\LocalMachine\Root | Where-Object { $_.Subject -like "*mkcert*" }
if ($cert) { Remove-Item $cert.PSPath }
```

In WSL2:

```bash
ddev utility tls-diagnose
# Expected: red ✗ "mkcert CA NOT found in Windows certificate store"
# Expected: Instructions to run mkcert -install in PowerShell
# Expected: Live Connectivity: PowerShell Invoke-WebRequest shows CERT_ERROR
# Expected: exit code 1
```

Restore in PowerShell:

```powershell
mkcert -install
```

### Test: CA fingerprint mismatch (different CA on each side)

Simulate by regenerating the WSL2-side CA:

```bash
# In WSL2 — this changes the WSL2 CA but leaves the Windows cert store with the old CA
mkcert -uninstall
rm -rf $(mkcert -CAROOT)
mkcert -install

ddev utility tls-diagnose
# Expected: ⚠ "CA fingerprint mismatch: WSL2=<hash>  Windows=<hash>"
# Expected: Instructions to run mkcert -install in PowerShell
# Expected: exit code 1
```

Restore:

```powershell
# In PowerShell
mkcert -install
```

```bash
# In WSL2 — restart to pick up the Windows CAROOT again
wsl --shutdown
```

### Test: Windows-side mkcert not installed

In Windows PowerShell, rename mkcert temporarily:

```powershell
Rename-Item "$env:ProgramData\chocolatey\bin\mkcert.exe" "mkcert.exe.bak"
```

In WSL2:

```bash
ddev utility tls-diagnose
# Expected: red ✗ "Windows-side mkcert.exe not found in Windows PATH"
# Expected: Install instructions
# Expected: exit code 1
```

Restore in PowerShell:

```powershell
Rename-Item "$env:ProgramData\chocolatey\bin\mkcert.exe.bak" "mkcert.exe"
```

### Test: WSLg mode (Linux browser inside WSL2)

If a Linux browser (e.g. chromium) is installed inside WSL2 via apt:

```bash
sudo apt install -y chromium-browser

ddev utility tls-diagnose
# Expected: "WSLg Browser Detection" section appears
# Expected: Prompt asking if troubleshooting a WSLg browser
# If you answer yes:
#   - WSL2 Configuration and Windows cert store checks are skipped
#   - Linux-side certutil check shown instead
# If you answer no:
#   - Full WSL2 configuration checks proceed as normal
```

### Test: WSL2 mirrored networking without hostAddressLoopback

If your `.wslconfig` has `[wsl2] networkingMode=mirrored` but not `hostAddressLoopback=true`:

```bash
ddev utility tls-diagnose
# Expected in WSL2 Configuration: ⚠ "WSL2 mirrored mode: hostAddressLoopback is not enabled"
# Expected: Instructions to add hostAddressLoopback=true to .wslconfig
```

---

## Windows (native, not WSL2)

The binary for Windows is `.gotmp/bin/windows_amd64/ddev.exe`. Run in a standard Command Prompt or PowerShell (not as Administrator).

### Baseline

```
ddev utility tls-diagnose
```

Expected:

- mkcert found (installed via `choco install mkcert` or `winget install mkcert`)
- OS Trust Store: CA already installed
- Firefox: detected or not detected; if detected, shows warning about manual CA import
- WSL2 Configuration: section absent
- Certificate Files: default cert valid
- Summary: No issues found, exit code 0

### Test: mkcert CA not in Windows certificate store

```powershell
# Remove from both stores
Get-ChildItem Cert:\CurrentUser\Root | Where-Object { $_.Subject -like "*mkcert*" } | Remove-Item
Get-ChildItem Cert:\LocalMachine\Root | Where-Object { $_.Subject -like "*mkcert*" } | Remove-Item

ddev utility tls-diagnose
# Expected: "mkcert -install failed" or warning that CA is not trusted
# Expected: exit code 1

mkcert -install
```

### Test: Firefox present on Windows

If Firefox is installed:

```
ddev utility tls-diagnose
# Expected: ⚠ "Firefox detected on Windows — Firefox does not use the Windows system certificate store"
# Expected: Path to rootCA.pem for manual import
# Expected: Link to configuring-browsers docs
```

---

## Things to Watch For (All Platforms)

- Exit code: 0 means all clean, 1 means at least one issue found. Verify with `echo $?` (bash) or `echo $LASTEXITCODE` (PowerShell).
- The `Summary` section should always appear at the end, even if earlier sections fail.
- The command should not panic or produce a Go stack trace under any broken-input condition.
- When run outside a project directory (or from a directory with no `.ddev/config.yaml`), the `Certificate Files` section still checks the global default cert, and `Live Connectivity` is skipped with an info message.
- Colors: failures print a red `✗`, info prints `ℹ`, and success prints `✓`. If output is piped or redirected, colors may not appear — this is expected.
- Non-interactive use: if run in a non-TTY environment (pipe, CI), the WSLg prompt is skipped.
