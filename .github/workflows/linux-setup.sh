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

# Install cloudflared
sudo mkdir -p --mode=0755 /usr/share/keyrings
curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg \
  | sudo tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null \
  && echo 'deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main' \
  | sudo tee /etc/apt/sources.list.d/cloudflared.list \
  && sudo apt-get update -qq >/dev/null \
  && sudo apt-get install -y -qq cloudflared

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
