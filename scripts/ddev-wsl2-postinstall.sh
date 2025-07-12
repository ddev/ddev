#!/bin/bash
# Post-install script for ddev-wsl2 package
# Unblocks Windows executables to prevent security warnings

# Check if we can access Windows PowerShell
if command -v powershell.exe >/dev/null 2>&1; then
    echo "Unblocking WSL2 executables to prevent Windows security warnings..."
    
    # Create a temporary PowerShell script to handle both path formats
    TEMP_PS1="/tmp/unblock-wsl-exes-$$.ps1"
    cat > "$TEMP_PS1" << 'EOF'
$unblocked = $false

# Try newer \\wsl.localhost path format first
try {
    $files = Get-ChildItem '\\wsl.localhost\*\usr\bin\ddev-hostname.exe' -ErrorAction SilentlyContinue
    $files += Get-ChildItem '\\wsl.localhost\*\usr\bin\mkcert.exe' -ErrorAction SilentlyContinue
    if ($files.Count -gt 0) {
        $files | Unblock-File
        Write-Host "WSL2 executables unblocked successfully (wsl.localhost)"
        $unblocked = $true
    }
} catch {
    # Continue to try legacy format
}

# If newer format didn't work, try legacy \\wsl$ format
if (-not $unblocked) {
    try {
        $files = Get-ChildItem '\\wsl$\*\usr\bin\ddev-hostname.exe' -ErrorAction SilentlyContinue
        $files += Get-ChildItem '\\wsl$\*\usr\bin\mkcert.exe' -ErrorAction SilentlyContinue
        if ($files.Count -gt 0) {
            $files | Unblock-File
            Write-Host "WSL2 executables unblocked successfully (wsl$)"
            $unblocked = $true
        }
    } catch {
        # Both formats failed
    }
}

if (-not $unblocked) {
    Write-Host "Note: Could not find WSL2 executables to unblock automatically."
    Write-Host "If you see Windows security warnings, manually run one of:"
    Write-Host "powershell.exe -c \"Get-ChildItem '\\wsl.localhost\*\usr\bin\*.exe' | Unblock-File\""
    Write-Host "powershell.exe -c \"Get-ChildItem '\\wsl$\*\usr\bin\*.exe' | Unblock-File\""
}
EOF
    
    # Run the PowerShell script (try wsl.exe first, fallback to direct)
    if command -v wsl.exe >/dev/null 2>&1; then
        wsl.exe powershell.exe -ExecutionPolicy Bypass -WindowStyle Hidden -File "$TEMP_PS1" 2>/dev/null
    else
        powershell.exe -ExecutionPolicy Bypass -WindowStyle Hidden -File "$TEMP_PS1" 2>/dev/null
    fi || {
        echo "Note: Could not automatically unblock WSL2 executables."
        echo "If you see Windows security warnings, manually run one of:"
        echo "powershell.exe -c \"Get-ChildItem '\\\\wsl.localhost\\\\*\\\\usr\\\\bin\\\\*.exe' | Unblock-File\""
        echo "powershell.exe -c \"Get-ChildItem '\\\\wsl\$\\\\*\\\\usr\\\\bin\\\\*.exe' | Unblock-File\""
    }
    
    # Clean up
    rm -f "$TEMP_PS1"
else
    echo "Note: To prevent Windows security warnings for WSL2 executables, manually run one of:"
    echo "powershell.exe -c \"Get-ChildItem '\\\\wsl.localhost\\\\*\\\\usr\\\\bin\\\\*.exe' | Unblock-File\""
    echo "powershell.exe -c \"Get-ChildItem '\\\\wsl\$\\\\*\\\\usr\\\\bin\\\\*.exe' | Unblock-File\""
fi