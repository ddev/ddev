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
linux)
    sudo apt update && sudo apt upgrade -y
    # linux no longer needs homebrew
esac

echo "Deleting unused images with ddev delete images"
ddev delete images -y || true

# Remove any -built images, as we want to make sure tests do the building.
docker rmi -f $(docker images --filter "dangling=true" -q --no-trunc) >/dev/null 2>&1 || true
docker rmi -f $(docker images | awk '/ddev.*-built/ {print $3}' ) >/dev/null 2>&1 || true

# Clean the docker build cache
docker buildx prune -f -a || true
# Remove any images with name '-built'
docker rm -f $(docker ps -aq) >/dev/null 2>&1 || true
docker rmi -f $(docker images | awk '/[-]built/ { print $3 }')  >/dev/null 2>&1 || true
