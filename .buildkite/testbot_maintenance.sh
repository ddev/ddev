#!/bin/bash

set -eu -o pipefail

os=$(go env GOOS)

rm -rf ~/.ddev/Test* ~/.ddev/global_config.yaml ~/.ddev/project_list.yaml ~/.ddev/homeadditions ~/.ddev/commands ~/.ddev/bin/docker-compose* ~/tmp/ddevtest

# Latest git won't let you do much in a non-safe directory
git config --global --add safe.directory '*' || true

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
    brew pin buildkite-agent
    brew upgrade
    brew uninstall -f mysql-client || true
    for item in ddev/ddev/ddev golang golangci-lint libpq mkcert mkdocs mysql-client@8.0; do
        brew install $item || true
    done
    brew link --force libpq
    brew link mysql-client@8.0
    ;;
windows)
    (yes | choco upgrade -y golang nodejs markdownlint-cli mkcert mkdocs postgresql) || true
    ;;
# linux is currently WSL2 only
linux)
    # homebrew is only on amd64
    if [ "$(arch)" = "x86_64" ]; then
      for item in libpq mkdocs; do
        brew upgrade $item || brew install $item || true
      done
      brew link --force libpq
    fi
    ;;

esac

echo "Deleting unused images with ddev delete images"
ddev delete images -y || true

# Remove any -built images, as we want to make sure tests do the building.
docker rmi -f $(docker images --filter "dangling=true" -q --no-trunc) >/dev/null 2>&1 || true
docker rmi -f $(docker images | awk '/ddev.*-built/ {print $3}' ) >/dev/null 2>&1 || true

# Make sure there aren't any dangling NFS volumes
if docker volume ls | grep '[Tt]est.*_nfsmount'; then
  docker volume rm -f $(docker volume ls | awk '/[Tt]est.*_nfsmount/ { print $2; }') || true
fi

# Clean the docker build cache
docker buildx prune -f -a || true
# Remove any images with name '-built'
docker rm -f $(docker ps -aq) >/dev/null 2>&1 || true
docker rmi -f $(docker images | awk '/[-]built/ { print $3 }')  >/dev/null 2>&1 || true
