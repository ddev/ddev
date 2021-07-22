#!/bin/bash

set -eu -o pipefail

os=$(go env GOOS)

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
    brew uninstall mutagen-io/mutagen/mutagen || true
    for item in drud/ddev/ddev golang golangci-lint mkcert mkdocs mutagen-io/mutagen/mutagen-beta; do
        brew upgrade $item || brew install $item || true
    done
    ;;
windows)
    (yes | choco upgrade -y golang nodejs markdownlint-cli mkcert mkdocs) || true
    MUTAGEN_VERSION=0.12.0-beta3
    if ! command -v mutagen >/dev/null || [ "$(mutagen version)" != "${MUTAGEN_VERSION}" ]; then
      mkdir -p ~/tmp/mutagen ~/bin && curl -sSL -o ~/tmp/mutagen.tgz https://github.com/mutagen-io/mutagen/releases/download/v${MUTAGEN_VERSION}/mutagen_windows_amd64_v${MUTAGEN_VERSION}.tar.gz
      tar -zxf ~/tmp/mutagen.tgz -C ~/bin
    fi
    ;;
# linux is currently WSL2
linux)
    # homebrew is only on amd64
    if [ "$(arch)" = "x86_64" ]; then
      brew uninstall mutagen || true
      for item in drud/ddev/ddev golang mkcert mkdocs mutagen-io/mutagen/mutagen-beta; do
        brew upgrade $item || brew install $item || true
      done
    fi
    ;;

esac

# Stop mutagen daemon in case of existing one with different version
( echo y | mutagen daemon stop ) || true

(yes | ddev delete images) || true

# Remove any -built images, as we want to make sure tests do the building.
docker rmi -f $(docker images --filter "dangling=true" -q --no-trunc) >/dev/null || true
docker rmi -f $(docker images | awk '/drud.*-built/ {print $3}' ) >/dev/null || true
