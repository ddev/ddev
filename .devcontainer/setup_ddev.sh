#!/bin/bash

make
sudo ln -sf /workspaces/ddev/.gotmp/bin/linux_amd64/ddev /usr/local/bin/ddev
ddev debug download-images
ddev delete -Oy tmp >/dev/null
ddev --version

export DDEV_NONINTERACTIVE=true
DDEV_REPO=${DDEV_REPO:-https://github.com/drud/d9simple}
DDEV_ARTIFACTS=${DDEV_REPO}-artifacts
git clone ${DDEV_ARTIFACTS} "/tmp/${DDEV_ARTIFACTS##*/}" || true
reponame=${DDEV_REPO##*/}
mkdir -p /workspaces/${reponame} && cd /workspace/${reponame}
if [ ! -d /workspaces/${reponame}/.git ]; then
    git clone ${DDEV_REPO} /workspace/${reponame}
fi
if [ ! -f .ddev/config.yaml ]; then
    ddev config --auto
fi
ddev stop -a
ddev start -y
if [ -d "/tmp/${DDEV_ARTIFACTS##*/}" ]; then
    ddev import-db --src=/tmp/${DDEV_ARTIFACTS##*/}/db.sql.gz
    ddev import-files --src=/tmp/${DDEV_ARTIFACTS##*/}/files.tgz
fi