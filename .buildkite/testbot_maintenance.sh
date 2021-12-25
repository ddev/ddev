#!/bin/bash

set -eu -o pipefail

os=$(go env GOOS)

rm -rf ~/.ddev/Test* ~/.ddev/global_config.yaml ~/.ddev/homeadditions ~/.ddev/commands ~/.ddev/bin/docker-comnpose* ~/tmp/ddevtest

# Install ngrok if it's not there.
if ! command -v ngrok >/dev/null; then
    case $os in
    darwin)
        brew install homebrew/cask/ngrok
        ;;
    windows)
        (yes | choco install -y ngrok) || true
        ;;
    linux)
        curl -sSL --fail -o /tmp/ngrok.zip https://bin.equinox.io/c/4VmDzA7iaHb/ngrok-stable-linux-amd64.zip && sudo unzip -o -d /usr/local/bin /tmp/ngrok.zip
        ;;
    esac
fi

# Upgrade various items on various operating systems
case $os in
darwin)
    brew uninstall mutagen-io/mutagen/mutagen-beta mutagen-io/mutagen/mutagen || true
    for item in drud/ddev/ddev golang golangci-lint mkcert mkdocs; do
        brew upgrade $item || brew install $item || true
    done
    ;;
windows)
    (yes | choco upgrade -y golang nodejs markdownlint-cli mkcert mkdocs) || true
    (yes | choco uninstall -y mutagen) || true
    ;;
# linux is currently WSL2
linux)
    # homebrew is only on amd64
    if [ "$(arch)" = "x86_64" ]; then
      brew uninstall mutagen-beta mutagen || true
      for item in drud/ddev/ddev golang mkcert mkdocs; do
        brew upgrade $item || brew install $item || true
      done
    fi
    ;;

esac

(yes | ddev delete images) || true

# Remove any -built images, as we want to make sure tests do the building.
docker rmi -f $(docker images --filter "dangling=true" -q --no-trunc) >/dev/null || true
docker rmi -f $(docker images | awk '/drud.*-built/ {print $3}' ) >/dev/null || true

# Make sure there aren't any dangling NFS volumes
if docker volume ls | grep '[Tt]est.*_nfsmount'; then
  docker volume rm -f $(docker volume ls | awk '/[Tt]est.*_nfsmount/ { print $2; }') || true
fi
