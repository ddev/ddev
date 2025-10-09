#!/usr/bin/env bash

set -o errexit

# Basic tools

set -x

if [ ! -z "${DOCKERHUB_PULL_USERNAME:-}" ]; then
  set +x
  echo "${DOCKERHUB_PULL_PASSWORD}" | docker login --username "${DOCKERHUB_PULL_USERNAME}" --password-stdin
  set -x
fi

sudo apt-get update -qq >/dev/null
sudo apt-get install -y -qq build-essential expect libnss3-tools libcurl4-gnutls-dev postgresql-client >/dev/null

curl -sSL https://ngrok-agent.s3.amazonaws.com/ngrok.asc \
  | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null \
  && echo "deb https://ngrok-agent.s3.amazonaws.com bookworm main" \
  | sudo tee /etc/apt/sources.list.d/ngrok.list \
  && sudo apt-get update -qq >/dev/null \
  && sudo apt-get install -y -qq ngrok

if [[ ${DDEV_TEST_PODMAN_ROOTLESS:-} == "true" ]]; then
  echo "Setting up podman-rootless"
  sudo systemctl disable --now docker.service docker.socket
  sudo rm -f /var/run/docker.sock
  sudo apt-get install -y -qq podman >/dev/null
  systemctl --user enable --now podman.socket
  docker context create podman --docker host="unix://$(podman info --format '{{.Host.RemoteSocket.Path}}')"
  docker context use podman
  sudo sysctl net.ipv4.ip_unprivileged_port_start=80
elif [[ "${DDEV_TEST_DOCKER_ROOTLESS:-}" == "true" ]]; then
  echo "Setting up docker-rootless"
  sudo systemctl disable --now docker.service docker.socket
  sudo rm -f /var/run/docker.sock
  # Configure AppArmor for rootlesskit
  # Source: https://github.com/ScribeMD/rootless-docker/pull/402
  abi4_version="$(find /etc/apparmor.d/abi -maxdepth 1 -name '4.*' -printf '%f\n' | sort -nr | head -1)"
  filename=$(echo $HOME/bin/rootlesskit | sed -e s@^/@@ -e s@/@.@g)
  sudo tee /etc/apparmor.d/${filename} > /dev/null <<EOF
abi <abi/${abi4_version}>,

include <tunables/global>

"$HOME/bin/rootlesskit" flags=(unconfined) {
userns,

include if exists <local/${filename}>
}
EOF
  sudo systemctl restart apparmor.service
  # Install rootless docker
  curl -fsSL https://get.docker.com/rootless | sh
  sudo sysctl net.ipv4.ip_unprivileged_port_start=80
fi

# Without this .curlrc CircleCI linux image doesn't respect mkcert certs
echo "capath=/etc/ssl/certs/" >>~/.curlrc

source ~/.bashrc

brew tap bats-core/bats-core >/dev/null
brew tap ddev/ddev >/dev/null
for item in bats-core ddev docker-compose ghr golangci-lint bats-assert bats-file bats-support; do
    brew install $item >/dev/null || brew upgrade $item >/dev/null
done

mkcert -install

# Show info to simplify debugging
docker info
docker version
lsb_release -a
