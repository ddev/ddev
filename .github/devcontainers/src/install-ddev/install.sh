#!/bin/bash
set -eu -o pipefail

sudo apt update && sudo apt install -y curl wget xdg-utils

# This will eventually move to a simple apt install
sudo apt remove ddev >/dev/null 2>&1 || true
DDEV_URL=https://nightly.link/drud/ddev/actions/artifacts/485751722.zip
echo "Installing DDEV"

cd /tmp && curl -s -L -O ${DDEV_URL}
zipball=$(basename ${DDEV_URL})
unzip ${zipball}
chmod +x ddev && sudo mv ddev /usr/local/bin
rm -f ${zipball}
