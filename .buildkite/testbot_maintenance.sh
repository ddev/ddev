#!/bin/bash

set -eu

os=$(go env GOOS)

# Install ngrok if it's not there.
if ! command -v ngrok >/dev/null; then
    case $os in
    darwin)
        brew install homebrew/cask/ngrok
        ;;
    windows)
        choco install -y ngrok
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
    for item in drud/ddev/ddev golang golangci-lint mkcert mkdocs brew install mutagen-io/mutagen/mutagen-beta python3-yq; do
        brew upgrade $item || brew install $item || true
    done
    ;;
windows)
    choco upgrade -y golang nodejs markdownlint-cli mkcert mkdocs || true
    if ! command -v mutagen >/dev/null ; then
      MUTAGEN_VERSION=v0.12.0-beta3
      mkdir -p ~/tmp/mutagen ~/bin && curl -sSL -o ~/tmp/mutagen.tgz https://github.com/mutagen-io/mutagen/releases/download/${MUTAGEN_VERSION}/mutagen_windows_amd64_${MUTAGEN_VERSION}.tar.gz
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

yes | ddev delete images

# Remove any -built images, as we want to make sure tests do the building.
docker rmi -f $(docker images --filter "dangling=true" -q --no-trunc) >/dev/null || true
docker rmi -f $(docker images | awk '/drud.*-built/ {print $3}' ) >/dev/null || true

# Make sure the global internet detection timeout is not set to 0 (broken)
perl -pi -e 's/^internet_detection_timeout_ms:.*$/internet_detection_timeout_ms: 750/g' ~/.ddev/global_config.yaml
