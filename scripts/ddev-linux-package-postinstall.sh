#!/usr/bin/env bash
# Post-install script for ddev package
# Detects WSL2 and notifies users about the ddev-wsl2 package if not already installed

set -e

# Function to check if ddev-wsl2 is already installed
is_ddev_wsl2_installed() {
    # Check for deb package (dpkg) - look for "ii" status (installed)
    if command -v dpkg >/dev/null 2>&1; then
        dpkg -l ddev-wsl2 2>/dev/null | grep -q "^ii" && return 0
    fi
    
    # Check for rpm package (rpm)
    if command -v rpm >/dev/null 2>&1; then
        rpm -q ddev-wsl2 >/dev/null 2>&1 && return 0
    fi
    
    # Check if the WSL2 binaries exist (fallback)
    if [ -f /usr/bin/ddev-hostname.exe ] && [ -f /usr/bin/mkcert.exe ]; then
        return 0
    fi
    
    return 1
}

# Only show message if we're in WSL2 AND ddev-wsl2 is NOT installed
if grep -q "microsoft-standard" /proc/version && ! is_ddev_wsl2_installed; then
      echo ""
      echo "=================================================="
      echo "For optimal DDEV functionality in WSL2,"
      echo "please install the ddev-wsl2 package:"
      echo ""
      echo "  sudo apt-get update && sudo apt-get install ddev-wsl2"
      echo "  # or for RPM-based systems:"
      echo "  # sudo dnf install ddev-wsl2"
      echo ""
      echo "The ddev-wsl2 package provides Windows-side binaries"
      echo "(ddev-hostname.exe and mkcert.exe) required for proper"
      echo "WSL2 integration with Windows-side facilities."
      echo "=================================================="
      echo ""
fi

exit 0