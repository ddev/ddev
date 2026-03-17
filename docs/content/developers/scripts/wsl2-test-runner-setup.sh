#!/usr/bin/env bash

# This sets up an Ubuntu distro to be a test runner for WSL2
# Run this inside a working WSL2 Ubuntu Distro

# REQUIRED environment variable: BUILDKITE_AGENT_TOKEN
# REQUIRED environment variable: BUILDKITE_DOCKER_TYPE (dockerforwindows or wsl2)
# OPTIONAL environment variable: NGROK_TOKEN if ngrok will be run

set -eu -o pipefail

echo "Please enter your sudo password if requested" && sudo ls

# BUILDKITE_AGENT_TOKEN must be set
if [ "${BUILDKITE_AGENT_TOKEN:-}" = "" ]; then
  echo "BUILDKITE_AGENT_TOKEN must be set, export BUILDKITE_AGENT_TOKEN=token" && exit 1
fi

# BUILDKITE_DOCKER_TYPE must be set
if [ "${BUILDKITE_DOCKER_TYPE:-}" = "" ]; then
  echo "BUILDKITE_DOCKER_TYPE must be set to dockerforwindows or wsl2, export BUILDKITE_DOCKER_TYPE=dockerforwindows" && exit 2
fi

set -x
sudo apt-get update -qq >/dev/null && sudo apt-get upgrade -qq -y >/dev/null
sudo apt-get install -qq -y apt-transport-https autojump bats build-essential ca-certificates ccache clang curl dirmngr etckeeper expect git gnupg htop icinga2 jq libcurl4-gnutls-dev libnss3-tools lsb-release mariadb-client mkcert monitoring-plugins-contrib nagios-plugins postgresql-client unzip vim wslu xdg-utils zip >/dev/null

# docker-ce if required
if [ "${BUILDKITE_DOCKER_TYPE:-}" = "wsl2" ]; then
  sudo mkdir -p /etc/apt/keyrings
  sudo rm -f /etc/apt/keyrings/docker.gpg /etc/apt/sources.list.d/docker.list && curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo tee /etc/apt/keyrings/docker.asc > /dev/null && sudo chmod a+r /etc/apt/keyrings/docker.asc
  printf "Types: deb\nURIs: https://download.docker.com/linux/ubuntu\nSuites: %s\nComponents: stable\nSigned-By: /etc/apt/keyrings/docker.asc\n" "$(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}")" | sudo tee /etc/apt/sources.list.d/docker.sources > /dev/null
  sudo apt-get -qq update >/dev/null && sudo apt-get install -qq -y docker-ce docker-ce-cli etckeeper containerd.io docker-compose-plugin >/dev/null
  sudo usermod -aG docker $USER
fi

# golang
sudo snap install --classic go

# ngrok
curl -sSL https://ngrok-agent.s3.amazonaws.com/ngrok.asc \
  | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null \
  && echo "deb https://ngrok-agent.s3.amazonaws.com bookworm main" \
  | sudo tee /etc/apt/sources.list.d/ngrok.list \
  && sudo apt-get update >/dev/null \
  && sudo apt-get install -y ngrok >/dev/null

# ddev
sudo rm -f /etc/apt/keyrings/ddev.gpg /etc/apt/sources.list.d/ddev.list
sudo bash -c "curl -fsSL https://pkg.ddev.com/apt/gpg.key | tee /etc/apt/keyrings/ddev.asc > /dev/null && chmod a+r /etc/apt/keyrings/ddev.asc"
sudo bash -c "printf 'Types: deb\nURIs: https://pkg.ddev.com/apt/\nSuites: *\nComponents: *\nSigned-By: /etc/apt/keyrings/ddev.asc\n' > /etc/apt/sources.list.d/ddev.sources"
sudo apt-get -qq update >/dev/null && sudo apt-get install -qq -y ddev ddev-wsl2 >/dev/null

# Buildkite-agent
curl -fsSL https://keys.openpgp.org/vks/v1/by-fingerprint/32A37959C2FA5C3C99EFBC32A79206696452D198 | sudo gpg --dearmor -o /usr/share/keyrings/buildkite-agent-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/buildkite-agent-archive-keyring.gpg] https://apt.buildkite.com/buildkite-agent stable main" | sudo tee /etc/apt/sources.list.d/buildkite-agent.list
sudo apt-get update >/dev/null && sudo apt-get install -y buildkite-agent >/dev/null

# Edit the config file. Does not need sudo because buildkite-agent owns the file
sed -i "s/^token=.*/token=\"$BUILDKITE_AGENT_TOKEN\"/" /etc/buildkite-agent/buildkite-agent.cfg
echo "tags=\"os=wsl2,architecture=amd64,dockertype=${BUILDKITE_DOCKER_TYPE:-}\"" >> /etc/buildkite-agent/buildkite-agent.cfg

sudo systemctl enable buildkite-agent

# Setup sudo
(echo "ALL ALL=NOPASSWD: ALL" | sudo tee /etc/sudoers.d/all) && sudo chmod 440 /etc/sudoers.d/all

# git global confnig (safe.directory)
git config --global --add safe.directory '*'

# curl setup to respect mkcert
echo "capath=/etc/ssl/certs/" >>~/.curlrc

# Optionally set up ngrok token
if [ "${NGROK_TOKEN:-}" != "" ]; then
  ngrok authtoken ${NGROK_TOKEN:-}
else
  echo "NGROK_TOKEN not set so not doing ngrok authtoken"
fi

mkcert -install

echo "In the editor, change the home directory of buildkite-agent to /var/lib/buildkite-agent"
echo "Press any key to continue..."
read x
sudo vipw || true
sudo systemctl start buildkite-agent

# nagios for icinga2 needs to be in docker group
if ! getent group docker > /dev/null; then
    sudo groupadd docker
fi
sudo usermod -aG docker nagios
sudo usermod -aG docker buildkite-agent

# Check out ddev code for later use
mkdir -p /var/lib/buildkite-agent/workspace && pushd /var/lib/buildkite-agent/workspace && git clone -o upstream https://github.com/ddev/ddev && popd

echo "Now reboot the distro with 'wsl.exe -t Ubuntu' and restart it"
