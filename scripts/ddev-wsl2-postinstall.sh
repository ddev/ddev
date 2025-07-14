#!/bin/bash
# Post-install script for ddev-wsl2 package
# Unblocks Windows executables to prevent security warnings

echo "Configuring WSL2 security settings..."

# Try registry access using reg.exe from PATH
if command -v reg.exe >/dev/null 2>&1; then
    echo "Attempting registry modification via reg.exe..."
    REG_OUTPUT=$(reg.exe add "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings\\ZoneMap\\Domains\\wsl.localhost" /v "file" /t REG_DWORD /d 1 /f 2>&1)
    REG_EXIT_CODE=$?
    if [ $REG_EXIT_CODE -eq 0 ]; then
        echo "WSL2 security settings configured successfully via registry"
        exit 0
    else
        echo "Registry method failed: $REG_OUTPUT"
    fi
fi

# If reg.exe method failed or not found, show manual instructions
echo "Note: Could not automatically configure WSL2 security settings."
echo "To resolve Windows security warnings manually:"
echo "1. Open Internet Options (Control Panel > Internet Options)"
echo "2. Go to Security tab > Local Intranet > Sites > Advanced"
echo "3. Add this website to the zone:"
echo "   - \\\\wsl.localhost"
echo "4. Click OK to save"
