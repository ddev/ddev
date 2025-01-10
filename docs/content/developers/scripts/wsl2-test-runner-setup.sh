#!/usr/bin/env bash

# This sets up an Ubuntu distro to be a test runner for WSL2
# Run this inside a working WSL2 Ubuntu Distro

set -eu -o pipefail

sudo apt update && sudo apt upgrade -y
sudo apt install -y apt-transport-https autojump build-essential ca-certificates curl dirmngr etckeeper expect git gnupg icinga2 jq libcurl4-gnutls-dev libnss3-tools lsb-release mariadb-client nagios-plugins postgresql-client unzip vim zip

sudo snap install --classic go

# ngrok
curl -sSL https://ngrok-agent.s3.amazonaws.com/ngrok.asc \
  | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null \
  && echo "deb https://ngrok-agent.s3.amazonaws.com buster main" \
  | sudo tee /etc/apt/sources.list.d/ngrok.list \
  && sudo apt update \
  && sudo apt install ngrok

# Buildkite-agent
curl -fsSL https://keys.openpgp.org/vks/v1/by-fingerprint/32A37959C2FA5C3C99EFBC32A79206696452D198 | sudo gpg --dearmor -o /usr/share/keyrings/buildkite-agent-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/buildkite-agent-archive-keyring.gpg] https://apt.buildkite.com/buildkite-agent stable main" | sudo tee /etc/apt/sources.list.d/buildkite-agent.list
sudo apt-get update && sudo apt-get install -y buildkite-agent
sudo usermod -d /var/lib/buildkite-agent buildkite-agent


