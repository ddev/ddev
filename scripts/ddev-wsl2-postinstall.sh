#!/bin/bash
# Post-install script for ddev-wsl2 package
# Unblocks Windows executables to prevent security warnings

# Check if we can access Windows PowerShell
if command -v powershell.exe >/dev/null 2>&1; then
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
    
    # Add wsl\$ to Local Intranet zone for legacy support
    \$wslDollarPath = "\$registryPath\\wsl\$"
    if (-not (Test-Path \$wslDollarPath)) {
        New-Item -Path \$wslDollarPath -Force | Out-Null
    }
    Set-ItemProperty -Path \$wslDollarPath -Name "file" -Value 1 -Type DWord
    Write-Host "Added *.wsl\$ to Local Intranet zone"
    
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
    Write-Host "3. Add these websites to the zone:"
    Write-Host "   - *.wsl.localhost"
    Write-Host "   - *.wsl\$"
    Write-Host "4. Click OK to save"
}
EOF
    
    # Run the PowerShell script (try wsl.exe first, fallback to direct)
    echo "Running PowerShell script to configure security zones..."
    if command -v wsl.exe >/dev/null 2>&1; then
        wsl.exe powershell.exe -ExecutionPolicy Bypass -File "$TEMP_PS1"
    else
        powershell.exe -ExecutionPolicy Bypass -File "$TEMP_PS1"
    fi || {
        echo "Note: Could not automatically configure WSL2 security settings."
        echo "To resolve Windows security warnings manually:"
        echo "1. Open Internet Options (Control Panel > Internet Options)"
        echo "2. Go to Security tab > Local Intranet > Sites > Advanced"
        echo "3. Add these websites to the zone:"
        echo "   - *.wsl.localhost"
        echo "   - *.wsl\$"
        echo "4. Click OK to save"
    }
    
    # Clean up
    rm -f "$TEMP_PS1"
else
    echo "Note: To prevent Windows security warnings for WSL2 executables:"
    echo "1. Open Internet Options (Control Panel > Internet Options)"
    echo "2. Go to Security tab > Local Intranet > Sites > Advanced"
    echo "3. Add these websites to the zone:"
    echo "   - *.wsl.localhost"
    echo "   - *.wsl\$"
    echo "4. Click OK to save"
fi