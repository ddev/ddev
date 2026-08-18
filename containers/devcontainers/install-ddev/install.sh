#!/usr/bin/env bash
set -eu -o pipefail
set -x

# Install DDEV and dependencies
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://packages.ddev.com/public/gpg.key | sudo tee /etc/apt/keyrings/ddev.asc > /dev/null
sudo chmod a+r /etc/apt/keyrings/ddev.asc
sudo rm -f /etc/apt/keyrings/ddev.gpg /etc/apt/sources.list.d/ddev.list
printf "Types: deb\nURIs: https://packages.ddev.com/public/deb/ubuntu\nSuites: stable\nComponents: main\nSigned-By: /etc/apt/keyrings/ddev.asc\n" | sudo tee /etc/apt/sources.list.d/ddev.sources >/dev/null
sudo apt-get update >/dev/null && sudo apt-get install -y ddev mkcert xdg-utils >/dev/null

# Copy post-create script to permanent location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
sudo cp "$SCRIPT_DIR/post-create.sh" /usr/local/bin/ddev-post-create.sh
sudo chmod +x /usr/local/bin/ddev-post-create.sh
