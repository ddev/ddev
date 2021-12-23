#!/usr/bin/env bash

set -eu -o pipefail

# colima has golang as dependency, so is going to install go anyway.
# So we have to get rid of it somehow.
brew uninstall go@1.15 || true
brew install docker docker-compose mkcert mysql-client
brew install colima --HEAD
brew link --force mysql-client
brew link go

# This command allows adding CA (in mkcert, etc) without the popup trust prompt
# Mentioned in https://github.com/actions/virtual-environments/issues/4519#issuecomment-970202641
sudo security authorizationdb write com.apple.trust-settings.admin allow
colima start
mkcert -install
