#!/usr/bin/env bash

set -eu -o pipefail

brew update >/dev/null

# colima has golang as dependency, so is going to install go anyway.
# So we have to get rid of it somehow.
brew uninstall go@1.15 || true
brew unlink go || true
brew uninstall go@1.17 || true
brew uninstall postgresql || true
brew install colima docker docker-compose go jq libpq mkcert mysql-client
brew link --force go libpq mysql-client

# This command allows adding CA (in mkcert, etc) without the popup trust prompt
# Mentioned in https://github.com/actions/virtual-environments/issues/4519#issuecomment-970202641
sudo security authorizationdb write com.apple.trust-settings.admin allow

# Github actions macOS runners have 14BG RAM so might as well use it.
# https://docs.github.com/en/actions/using-github-hosted-runners/about-github-hosted-runners#supported-runners-and-hardware-resources
colima start --cpu 3 --memory 6

# I haven't been able to get mkcert-trusted certs in there, not sure why
# You can't answer the security prompt, but that's what the
# sudo security authorizationdb write com.apple.trust-settings.admin allow
# was supposed to fix.  rfay 2021-12-24
#sudo mkcert -install && sudo chown -R $UID "$(mkcert -CAROOT)"
