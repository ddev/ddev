#!/usr/bin/env bash

# This script runs as root inside WSL2 to set up the environment for DDEV testing.
# It installs Docker CE, Go, build dependencies, and creates a testuser.

set -eu -o pipefail
set -x

# Accept GO_VERSION from the environment, default to a known good version
GO_VERSION="${GO_VERSION:-1.26.0}"

export DEBIAN_FRONTEND=noninteractive

echo "=== Updating apt packages ==="
apt-get update -qq >/dev/null
apt-get upgrade -qq -y >/dev/null
apt-get install -qq -y \
  apt-transport-https \
  build-essential \
  ca-certificates \
  curl \
  expect \
  git \
  gnupg \
  jq \
  libcurl4-gnutls-dev \
  libnss3-tools \
  lsb-release \
  make \
  mariadb-client \
  postgresql-client \
  software-properties-common \
  unzip \
  >/dev/null

echo "=== Installing Docker CE ==="
install -m 0755 -d /etc/apt/keyrings
rm -f /etc/apt/keyrings/docker.gpg
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
  | tee /etc/apt/sources.list.d/docker.list >/dev/null
apt-get update -qq >/dev/null
apt-get install -qq -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin >/dev/null

echo "=== Starting Docker daemon ==="
systemctl enable --now docker

echo "=== Installing Go ${GO_VERSION} ==="
curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf /tmp/go.tar.gz
rm /tmp/go.tar.gz
ln -sf /usr/local/go/bin/go /usr/local/bin/go
ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

echo "=== Installing mkcert ==="
MKCERT_VERSION="v1.4.4"
curl -fsSL "https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-linux-amd64" -o /usr/local/bin/mkcert
chmod +x /usr/local/bin/mkcert

echo "=== Installing ngrok ==="
curl -sSL https://ngrok-agent.s3.amazonaws.com/ngrok.asc \
  | tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null
echo "deb https://ngrok-agent.s3.amazonaws.com bookworm main" \
  | tee /etc/apt/sources.list.d/ngrok.list >/dev/null
apt-get update -qq >/dev/null
apt-get install -qq -y ngrok >/dev/null

echo "=== Installing cloudflared ==="
mkdir -p --mode=0755 /usr/share/keyrings
curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg \
  | tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null
echo 'deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main' \
  | tee /etc/apt/sources.list.d/cloudflared.list >/dev/null
apt-get update -qq >/dev/null
apt-get install -qq -y cloudflared >/dev/null

echo "=== Configuring testuser ==="
# testuser and /etc/wsl.conf are already created by the workflow install step
usermod -aG docker testuser
# Passwordless sudo for testuser
echo "testuser ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/testuser
chmod 440 /etc/sudoers.d/testuser

echo "=== Configuring git safe directory ==="
git config --global --add safe.directory '*'
# Also set for testuser
su - testuser -c "git config --global --add safe.directory '*'"

echo "=== Configuring curlrc for mkcert ==="
su - testuser -c 'echo "capath=/etc/ssl/certs/" >> ~/.curlrc'

echo "=== Installing mkcert CA for testuser ==="
su - testuser -c "mkcert -install"

echo "=== Verifying installations ==="
go version
docker version
mkcert --version || true
git --version

echo "=== WSL2 setup complete ==="
