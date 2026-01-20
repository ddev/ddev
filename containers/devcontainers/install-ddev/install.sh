#!/usr/bin/env bash
set -eu -o pipefail
set -x

# Install DDEV and dependencies
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | sudo tee /etc/apt/keyrings/ddev.gpg > /dev/null
echo "deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * *" | sudo tee /etc/apt/sources.list.d/ddev.list
sudo apt-get update >/dev/null && sudo apt-get install -y ddev mkcert xdg-utils >/dev/null
sudo mkcert -install

# Copy post-create script to permanent location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
sudo cp "$SCRIPT_DIR/post-create.sh" /usr/local/bin/ddev-post-create.sh
sudo chmod +x /usr/local/bin/ddev-post-create.sh
