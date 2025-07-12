#!/bin/bash
# Post-install script for ddev-wsl2 package
# Unblocks Windows executables to prevent security warnings

# Check if we're in WSL2 environment (try registry access with absolute paths)
echo "Configuring WSL2 security settings..."

# Try to find Windows executables in common locations
WINDOWS_REG=""
WINDOWS_POWERSHELL=""

# Common Windows paths for reg.exe and powershell.exe
for windir in "/mnt/c/Windows" "/mnt/c/WINDOWS"; do
    if [ -f "$windir/System32/reg.exe" ]; then
        WINDOWS_REG="$windir/System32/reg.exe"
    fi
    if [ -f "$windir/System32/WindowsPowerShell/v1.0/powershell.exe" ]; then
        WINDOWS_POWERSHELL="$windir/System32/WindowsPowerShell/v1.0/powershell.exe"
    fi
done

# Try registry access first with absolute path
if [ -n "$WINDOWS_REG" ]; then
    echo "Attempting registry modification via $WINDOWS_REG..."
    REG_OUTPUT=$("$WINDOWS_REG" add "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings\\ZoneMap\\Domains\\wsl.localhost" /v "file" /t REG_DWORD /d 1 /f 2>&1)
    REG_EXIT_CODE=$?
    if [ $REG_EXIT_CODE -eq 0 ]; then
        echo "WSL2 security settings configured successfully via registry"
        exit 0
    else
        echo "Registry method failed: $REG_OUTPUT"
    fi
fi

# Fallback to PowerShell if available
if [ -n "$WINDOWS_POWERSHELL" ]; then
    echo "DEBUG: PowerShell found, trying fallback methods..."
    echo "Unblocking WSL2 executables to prevent Windows security warnings..."
    
    # Create a temporary PowerShell script to handle both path formats
    TEMP_PS1="/tmp/unblock-wsl-exes-$$.ps1"
    # Get current WSL distro name from environment
    CURRENT_DISTRO=${WSL_DISTRO_NAME:-Ubuntu}
    
    cat > "$TEMP_PS1" << EOF
\$success = \$false
\$currentDistro = "$CURRENT_DISTRO"

# Add WSL paths to Local Intranet security zone
# This is the known working solution for WSL2 executable warnings
try {
    Write-Host "Adding WSL2 paths to Local Intranet security zone..."
    
    # Local Intranet zone is zone 1 in Windows Internet Security
    \$registryPath = "HKCU:\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings\\ZoneMap\\Domains"
    
    # Add wsl.localhost to Local Intranet zone
    \$wslLocalhostPath = "\$registryPath\\wsl.localhost"
    if (-not (Test-Path \$wslLocalhostPath)) {
        New-Item -Path \$wslLocalhostPath -Force | Out-Null
    }
    Set-ItemProperty -Path \$wslLocalhostPath -Name "file" -Value 1 -Type DWord
    Write-Host "Added *.wsl.localhost to Local Intranet zone"
    
    Write-Host "WSL2 paths added to Local Intranet zone successfully"
    Write-Host "Windows security warnings for WSL2 executables should be resolved"
    \$success = \$true
    
} catch {
    Write-Host "Error adding WSL2 paths to Local Intranet zone: \$(\$_.Exception.Message)"
}

if (-not \$success) {
    Write-Host "Automatic configuration failed. Manual steps:"
    Write-Host "1. Open Internet Options (Control Panel > Internet Options)"
    Write-Host "2. Go to Security tab > Local Intranet > Sites > Advanced"
    Write-Host "3. Add this website to the zone:"
    Write-Host "   - \\\\wsl.localhost"
    Write-Host "4. Click OK to save"
}
EOF
    
    # Run the PowerShell script (try different execution methods)
    echo "Running PowerShell script to configure security zones..."
    
    # Determine the actual Windows user (not root if running via sudo/package install)
    WINDOWS_USER=""
    if [ "$(whoami)" = "root" ]; then
        # If running as root (e.g., during package install), try to find the real user
        WINDOWS_USER=$(who | head -1 | awk '{print $1}')
        echo "Running as root, detected Windows user: $WINDOWS_USER"
    fi
    
    # Try PowerShell methods since direct registry already failed
    if command -v wsl.exe >/dev/null 2>&1; then
        echo "Registry method failed, attempting PowerShell via wsl.exe..."
        # Second try: Run PowerShell via wsl.exe
        if [ -n "$WINDOWS_USER" ]; then
            wsl.exe --user "$WINDOWS_USER" "$WINDOWS_POWERSHELL" -NoProfile -ExecutionPolicy Bypass -File "$TEMP_PS1" 2>/dev/null
        else
            wsl.exe "$WINDOWS_POWERSHELL" -NoProfile -ExecutionPolicy Bypass -File "$TEMP_PS1" 2>/dev/null
        fi
    else
        echo "PowerShell method failed, trying direct PowerShell execution..."
        # Third try: Direct PowerShell execution
        if ! "$WINDOWS_POWERSHELL" -NoProfile -ExecutionPolicy Bypass -File "$TEMP_PS1" 2>/dev/null; then
            echo "Note: Could not automatically configure WSL2 security settings."
        fi
    fi
    
    # If all methods failed, show manual instructions
    if [ -n "$WINDOWS_REG" ] && ! "$WINDOWS_REG" query "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings\\ZoneMap\\Domains\\wsl.localhost" /v "file" 2>/dev/null | grep -q "0x1"; then
        echo "Note: Could not automatically configure WSL2 security settings."
        echo "To resolve Windows security warnings manually:"
        echo "1. Open Internet Options (Control Panel > Internet Options)"
        echo "2. Go to Security tab > Local Intranet > Sites > Advanced"
        echo "3. Add this website to the zone:"
        echo "   - \\\\wsl.localhost"
        echo "4. Click OK to save"
    fi
    
    # Clean up
    rm -f "$TEMP_PS1"
else
    echo "DEBUG: PowerShell not found, showing manual instructions"
    echo "Note: To prevent Windows security warnings for WSL2 executables:"
    echo "1. Open Internet Options (Control Panel > Internet Options)"
    echo "2. Go to Security tab > Local Intranet > Sites > Advanced"
    echo "3. Add this website to the zone:"
    echo "   - \\\\wsl.localhost"
    echo "4. Click OK to save"
fi