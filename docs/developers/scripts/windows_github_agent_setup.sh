#!/bin/bash

set -eu -o pipefail
set -x

# Update kernel for WSL2
cd /tmp && curl -O -sSL https://wslstorestorage.blob.core.windows.net/wslblob/wsl_update_x64.msi && start wsl_update_x64.msi

# Wait for user to install the kernel
sleep 10

wsl --set-default-version 2

mkcert -install

# Set *global* line endings (not user) because the buildkite-agent may not be running as testbot user
perl -pi -e 's/autocrlf = true/autocrlf = false\n\teol = lf/' "/c/Program Files/Git/etc/gitconfig"

# Install Ubuntu from Microsoft store
# Then wsl --set-default Ubuntu

# Get firewall set up with a single run
winpty docker run -it --rm -p 80 busybox:stable ls

bash "/c/Program Files/ddev/windows_ddev_nfs_setup.sh"
