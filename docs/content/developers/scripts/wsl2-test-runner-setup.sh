#!/usr/bin/env bash

# This sets up an Ubuntu distro to be a test runner for WSL2
# Run this inside a working WSL2 Ubuntu Distro

# REQUIRED environment variable: BUILDKITE_AGENT_TOKEN
# REQUIRED environment variable: BUILDKITE_DOCKER_TYPE (dockerforwindows or wsl2)
# OPTIONAL environment variable: NGROK_TOKEN if ngrok will be run

set -eu -o pipefail

# BUILDKITE_AGENT_TOKEN must be set
if [ "${BUILDKITE_AGENT_TOKEN:-}" = "" ]; then
  echo "BUILDKITE_AGENT_TOKEN must be set, export BUILDKITE_AGENT_TOKEN=token" && exit 1
fi

# BUILDKITE_DOCKER_TYPE must be set
if [ "${BUILDKITE_DOCKER_TYPE:-}" = "" ]; then
  echo "BUILDKITE_DOCKER_TYPE must be set to dockerforwindows or wsl2, export BUILDKITE_DOCKER_TYPE=dockerforwindows" && exit 2
fi

sudo apt-get update -qq >/dev/null && sudo apt-get upgrade -qq -y >/dev/null
sudo apt-get install -qq -y apt-transport-https autojump bats build-essential ca-certificates ccache clang curl dirmngr etckeeper expect git gnupg htop icinga2 jq libcurl4-gnutls-dev libnss3-tools lsb-release mariadb-client mkcert nagios-plugins postgresql-client unzip vim wslu xdg-utils zip >/dev/null

# golang
sudo snap install --classic go

# ngrok
curl -sSL https://ngrok-agent.s3.amazonaws.com/ngrok.asc \
  | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null \
  && echo "deb https://ngrok-agent.s3.amazonaws.com buster main" \
  | sudo tee /etc/apt/sources.list.d/ngrok.list \
  && sudo apt-get update >/dev/null \
  && sudo apt-get install -y ngrok >/dev/null

# ddev
sudo rm -f /etc/apt/keyrings/ddev.gpg
sudo bash -c "rm -f /etc/apt/keyrings/ddev.gpg && curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null"
sudo bash -c 'echo deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ \* \* > /etc/apt/sources.list.d/ddev.list'
sudo apt-get -qq update >/dev/null && sudo apt-get install -qq -y ddev >/dev/null

# Buildkite-agent
curl -fsSL https://keys.openpgp.org/vks/v1/by-fingerprint/32A37959C2FA5C3C99EFBC32A79206696452D198 | sudo gpg --dearmor -o /usr/share/keyrings/buildkite-agent-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/buildkite-agent-archive-keyring.gpg] https://apt.buildkite.com/buildkite-agent stable main" | sudo tee /etc/apt/sources.list.d/buildkite-agent.list
sudo apt-get update >/dev/null && sudo apt-get install -y buildkite-agent >/dev/null

# Edit the config file. Does not need sudo because buildkite-agent owns the file
sed -i "s/^token=.*/token=\"$BUILDKITE_AGENT_TOKEN\"/" /etc/buildkite-agent/buildkite-agent.cfg
echo "tags=\"os=wsl2,architecture=amd64,dockertype=${BUILDKITE_DOCKER_TYPE:-}\"" >> /etc/buildkite-agent/buildkite-agent.cfg

sudo systemctl enable buildkite-agent

# Check out ddev
mkdir ~/workspace && pushd ~/workspace && git clone -o upstream https://github.com/ddev/ddev && popd

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

echo "Now reboot the distro with 'wsl.exe -t Ubuntu' and restart it"
