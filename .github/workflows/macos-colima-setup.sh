#!/usr/bin/env bash

set -eu -o pipefail

sudo chown -R ${USER} /usr/local/*
brew update >/dev/null

# After colima 0.5.0 it's best to redo the colima image completely
if (colima version | grep "colima version 0.5.0"); then
  colima delete -f
fi

# colima has golang as dependency, so is going to install go anyway.
# So we have to get rid of it somehow.
brew uninstall go@1.15 || true
brew unlink go || true
brew uninstall go@1.17 || true
brew uninstall postgresql || true
echo "====== Running brew install ======"
brew install -q colima docker docker-compose jq libpq mkcert mysql-client
echo "====== Running brew link ======"
brew link --force libpq mysql-client
echo "====== Completed brew link ======"


# This command allows adding CA (in mkcert, etc) without the popup trust prompt
# Mentioned in https://github.com/actions/virtual-environments/issues/4519#issuecomment-970202641
echo "====== Setting trust settings ======"
sudo security authorizationdb write com.apple.trust-settings.admin allow

# Github actions macOS runners have 14BG RAM so might as well use it.
# https://docs.github.com/en/actions/using-github-hosted-runners/about-github-hosted-runners#supported-runners-and-hardware-resources
echo "====== Starting colima ======"
colima start --cpu 3 --memory 6

# I haven't been able to get mkcert-trusted certs in there, not sure why
# You can't answer the security prompt, but that's what the
# sudo security authorizationdb write com.apple.trust-settings.admin allow
# was supposed to fix.  rfay 2021-12-24
#sudo mkcert -install && sudo chown -R $UID "$(mkcert -CAROOT)"
