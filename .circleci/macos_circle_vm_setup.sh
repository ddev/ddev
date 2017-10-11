#!/usr/bin/env bash

set -o errexit
set -x

docker version || echo "No docker yet"

# Basic tools
brew update && brew install mariadb coreutils golang && brew cask install docker

# macOS version
sw_vers


# Install, start docker, wait for daemon
sudo /Applications/Docker.app/Contents/MacOS/Docker --quit-after-install --unattended

# Open and wait from https://stackoverflow.com/a/35979292/215713
open -g -a /Applications/Docker.app || exit 2

# Wait for the server to start up, if applicable.
i=0
while ! docker system info &>/dev/null; do
  (( i++ == 0 )) && printf %s '-- Waiting for Docker to finish starting up...' || printf '.'
  sleep 1
done
(( i )) && printf '\n'

echo "-- Docker is ready."
docker version
