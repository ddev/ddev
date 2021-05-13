#!/bin/bash

# Install NSIS on test machines, any other post-install
# This should be run with administrative privileges.

set -eu -o pipefail
set -x

NSIS_VERSION=3.06.1
NSIS_HOME="/c/Program Files (x86)/NSIS"

# Note that this is not a silent NSIS install because the silent install
# does not install the default Plugins and Stubs
curl -sSL -o /tmp/nsis-setup.exe https://prdownloads.sourceforge.net/nsis/nsis-${NSIS_VERSION}-setup.exe?download && chmod +x /tmp/nsis-setup.exe
/tmp/nsis-setup.exe

# Get the Plugins for NSIS
curl -fsSL -o /tmp/EnVar-Plugin.zip https://github.com/GsNSIS/EnVar/releases/latest/download/EnVar-Plugin.zip && unzip -o -d "${NSIS_HOME}" /tmp/EnVar-Plugin.zip
curl -fsSL -o /tmp/INetC.zip https://github.com/DigitalMediaServer/NSIS-INetC-plugin/releases/latest/download/INetC.zip && unzip -o -d "${NSIS_HOME}/Plugins" /tmp/INetC.zip

echo "You must now add to the system path C:\Program Files (x86)\NSIS\bin, which for some reason is not added by the installer"
