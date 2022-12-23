#!/bin/bash
set -eu -o pipefail

sudo apt update >/dev/null && sudo apt install -y xdg-utils >/dev/null

# This will eventually move to a simple apt install
sudo apt remove ddev >/dev/null 2>&1 || true
DDEV_URL=https://nightly.link/drud/ddev/actions/artifacts/485751722.zip
echo "Installing DDEV"

cd /tmp && curl -s -L -O ${DDEV_URL}
zipball=$(basename ${DDEV_URL})
unzip ${zipball}
chmod +x ddev && sudo mv ddev /usr/local/bin

# When we have a better way to preloaod images, update this
mkdir /tmp/tempproject && cd /tmp/tempproject
ddev config --auto
ddev config global --omit-containers=ddev-router
ddev debug download-images

rm -rf ${zipball} /tmp/tempproject
