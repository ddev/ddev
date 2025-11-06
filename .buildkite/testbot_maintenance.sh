#!/usr/bin/env bash

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
    esac
fi

# Install cloudflared if it's not there.
if ! command -v cloudflared >/dev/null; then
    case $os in
    darwin)
        brew install cloudflared
        ;;
    esac
fi

# Upgrade various items on various operating systems
case $os in
darwin)
    brew pin buildkite-agent
    brew upgrade
    brew uninstall -f mysql-client || true
    for item in coreutils curl ddev/ddev/ddev golang golangci-lint libpq mkcert mkdocs mysql-client@8.0; do
        brew install $item || true
    done
    brew link --force libpq
    brew link mysql-client@8.0
    ;;
windows)
    (yes | choco upgrade -y ddev golang nodejs markdownlint-cli mkcert mkdocs postgresql) || true
    ;;
esac

echo "Deleting unused images with ddev delete images"
ddev delete images -y || true

# Remove any -built images, as we want to make sure tests do the building.
docker rmi -f $(docker images --filter "dangling=true" -q --no-trunc) >/dev/null 2>&1 || true
docker rmi -f $(docker images | awk '/ddev.*-built/ {print $3}' ) >/dev/null 2>&1 || true

# Clean the docker build cache
docker buildx prune -f -a >/dev/null
# Remove any images with name '-built'
docker rm -f $(docker ps -aq) >/dev/null 2>&1 || true
docker rmi -f $(docker images | awk '/[-]built/ { print $3 }')  >/dev/null 2>&1 || true

echo "--- cleaning up docker and Test directories"
echo "Warning: deleting all docker containers and deleting ~/.ddev/Test*"
ddev poweroff
if [ "$(docker ps -aq | wc -l )" -gt 0 ] ; then
	docker rm -f $(docker ps -aq) >/dev/null 2>&1
fi

docker system prune --volumes --force
docker volume prune -a -f

# Update all images that could have changed
( docker images | awk '/ddev|traefik|postgres/ {print $1":"$2 }' | xargs -L1 docker pull ) || true

# homebrew sometimes removes /usr/local/etc/my.cnf.d
if command -v brew >/dev/null; then
  mkdir -p "$(brew --prefix)/etc/my.cnf.d"
fi
