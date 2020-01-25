#!/bin/bash

set -eu

os=$(go env GOOS)

# Install ngrok if it's not there.
if ! command -v ngrok >/dev/null; then
    case $os in
    darwin)
        brew cask install ngrok
        ;;
    windows)
        choco install -y ngrok
        ;;
    linux)
        curl -sSL --fail -o /tmp/ngrok.zip https://bin.equinox.io/c/4VmDzA7iaHb/ngrok-stable-linux-amd64.zip && sudo unzip -o -d /usr/local/bin /tmp/ngrok.zip
        ;;
    esac
fi

# Upgrade mkcert on macOS
case $os in
darwin)
    brew upgrade mkcert || brew install mkcert || true
    brew upgrade golang || brew install golang || true
    ;;
windows)
    choco upgrade -y mkcert golang
    ;;
esac

# Remove any -built images, as we want to make sure tests do the building.
docker rmi -f $(docker images --filter "dangling=true" -q --no-trunc) >/dev/null || true
docker rmi -f $(docker images | awk '/drud.*-built/ {print $3}' ) >/dev/null || true
