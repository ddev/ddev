#!/usr/bin/env bash

set -o errexit

# Basic tools

set -x

DDEV_REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

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

# Install cloudflared
sudo mkdir -p --mode=0755 /usr/share/keyrings
curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg \
  | sudo tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null \
  && echo 'deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main' \
  | sudo tee /etc/apt/sources.list.d/cloudflared.list \
  && sudo apt-get update -qq >/dev/null \
  && sudo apt-get install -y -qq cloudflared

if [[ ${DDEV_TEST_PODMAN_ROOTLESS:-} == "true" ]]; then

  echo "Setting up podman-rootless"
  "${DDEV_REPO_ROOT}/scripts/linux-homebrew-podman-rootless.sh"
  # brew put podman at a new path; refresh this shell's command hash table.
  hash -r
  echo "Verifying podman-rootless setup"
  cat /etc/subuid
  cat /etc/subgid
  podman info
  podman version

elif [[ "${DDEV_TEST_DOCKER_ROOTLESS:-}" == "true" ]]; then

  echo "Setting up docker-rootless"
  sudo systemctl disable --now docker.service docker.socket
  sudo rm -f /var/run/docker.sock
  # Enable ports below 1024
  sudo mkdir -p /etc/sysctl.d
  echo 'net.ipv4.ip_unprivileged_port_start=0' | sudo tee -a /etc/sysctl.d/60-rootless.conf
  sudo sysctl -p /etc/sysctl.d/60-rootless.conf
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
  # Set the default network driver to "gvisor-tap-vsock" (see https://github.com/moby/moby/releases/tag/docker-v29.5.0)
  # Allow loopback https://github.com/moby/moby/issues/47684#issuecomment-2166149845
  mkdir -p ~/.config/systemd/user/docker.service.d
  cat << 'EOF' > ~/.config/systemd/user/docker.service.d/override.conf
[Service]
Environment="DOCKERD_ROOTLESS_ROOTLESSKIT_NET=gvisor-tap-vsock"
Environment="DOCKERD_ROOTLESS_ROOTLESSKIT_DISABLE_HOST_LOOPBACK=false"
EOF
  # Install rootless docker
  # Download the rootless installer script
  curl -fsSL https://get.docker.com/rootless -o /tmp/docker-rootless-install.sh

  # Get Docker rootful version from docker --version (format: "Docker version 29.1.3, build f52814d454")
  DOCKER_VERSION=$(docker --version | sed -E 's/Docker version ([0-9]+\.[0-9]+\.[0-9]+).*/\1/')

  # Install the latest Docker rootless for now (because of using gvisor-tap-vsock)
  DOCKER_VERSION=""

  if [ "${DOCKER_VERSION}" != "" ]; then
    # Replace STABLE_LATEST with the current Docker version to match rootful installation
    sed -i "s/STABLE_LATEST=\"[0-9.]*\"/STABLE_LATEST=\"${DOCKER_VERSION}\"/" /tmp/docker-rootless-install.sh
  fi

  # Execute the modified script
  sh /tmp/docker-rootless-install.sh
  cat /etc/subuid
  cat /etc/subgid

fi

# Without this .curlrc CircleCI linux image doesn't respect mkcert certs
echo "capath=/etc/ssl/certs/" >>~/.curlrc

source ~/.bashrc

brew trust bats-core/bats-core

for item in bats-core bats-core/bats-core/bats-assert bats-core/bats-core/bats-file bats-core/bats-core/bats-support ddev/ddev/ddev golangci-lint; do
  brew install $item >/dev/null || brew upgrade -y $item >/dev/null
done

mkcert -install

# Show info to simplify debugging
docker info
docker version
lsb_release -a
