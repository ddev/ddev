#!/usr/bin/env bash

# Install Docker + Docker buildx
mkdir ~/.docker
brew install docker docker-machine docker-compose
mkdir ~/.docker/cli-plugins
curl -sSL https://github.com/docker/buildx/releases/download/v0.4.2/buildx-v0.4.2.darwin-amd64 -o ~/.docker/cli-plugins/docker-buildx
chmod a+x ~/.docker/cli-plugins/docker-buildx
docker buildx version
docker-machine create --driver virtualbox default
docker-machine env default

# Install other dependencies
brew install make mysql-client mkcert
