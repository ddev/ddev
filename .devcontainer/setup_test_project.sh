#!/bin/bash

set -eu -o pipefail

echo "You don't need to wait for the test project to be set up."
set -x
make
sudo ln -sf /workspaces/ddev/.gotmp/bin/linux_amd64/ddev /usr/local/bin/ddev
ddev debug download-images
ddev delete -Oy tmp >/dev/null || true
ddev --version

export DDEV_NONINTERACTIVE=true
DDEV_REPO=${DDEV_REPO:-https://github.com/ddev/d9simple}
DDEV_ARTIFACTS=${DDEV_REPO}-artifacts
git clone ${DDEV_ARTIFACTS} "/tmp/${DDEV_ARTIFACTS##*/}" || true
reponame=${DDEV_REPO##*/}
mkdir -p /workspaces/${reponame} && cd /workspaces/${reponame}
if [ ! -d /workspaces/${reponame}/.git ]; then
    git clone ${DDEV_REPO} /workspaces/${reponame}
fi
if [ ! -f .ddev/config.yaml ]; then
    ddev config --auto
fi
ddev stop -a
ddev start -y
if [ -d "/tmp/${DDEV_ARTIFACTS##*/}" ]; then
    ddev import-db --file=/tmp/${DDEV_ARTIFACTS##*/}/db.sql.gz
    ddev import-files --source=/tmp/${DDEV_ARTIFACTS##*/}/files.tgz
fi
