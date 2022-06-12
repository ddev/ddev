#!/usr/bin/env bash

set -eu -o pipefail

# Install Docker + Docker buildx
mkdir -p ~/.docker/machine/cache
curl -Lo ~/.docker/machine/cache/boot2docker.iso https://github.com/boot2docker/boot2docker/releases/download/v19.03.12/boot2docker.iso
brew install docker docker-machine docker-compose
mkdir ~/.docker/cli-plugins
curl -sSL https://github.com/docker/buildx/releases/download/v0.5.1/buildx-v0.5.1.darwin-amd64 -o ~/.docker/cli-plugins/docker-buildx
chmod a+x ~/.docker/cli-plugins/docker-buildx
docker buildx version
docker-machine create --driver virtualbox default
docker-machine env default

# Install other dependencies
brew install make mysql mkcert

mkcert -install

set +eu
