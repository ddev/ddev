#!/usr/bin/env bash

set -o errexit
set -x


# Basic tools
brew update >/dev/null 2>/dev/null

# Get docker in first so we can install it and work on other things
brew cask install docker
sudo /Applications/Docker.app/Contents/MacOS/Docker --quit-after-install --unattended
nohup /Applications/Docker.app/Contents/MacOS/Docker --unattended &

brew tap drud/ddev
brew install mysql-client zip nsis jq expect coreutils golang ddev

curl -sSL -o /tmp/gotestsum.tgz https://github.com/gotestyourself/gotestsum/releases/download/v0.3.2/gotestsum_0.3.2_darwin_amd64.tar.gz && tar -C /usr/local/bin -zxf /tmp/gotestsum.tgz gotestsum

# gotestsum
GOTESTSUM_VERSION=0.3.2
curl -sSL "https://github.com/gotestyourself/gotestsum/releases/download/v$GOTESTSUM_VERSION/gotestsum_${GOTESTSUM_VERSION}_linux_amd64.tar.gz" | sudo tar -xz -C /usr/local/bin gotestsum && sudo chmod +x /usr/local/bin/gotestsum
curl -sSL -o /tmp/gotestsum.tgz https://github.com/gotestyourself/gotestsum/releases/download/${GOTESTSUM_VERSION}/gotestsum_${GOTESTSUM_VERSION}_darwin_amd64.tar.gz && tar -C /usr/local/bin -zxf /tmp/gotestsum.tgz gotestsum


while ! docker ps 2>/dev/null ; do
  sleep 5
  echo "Waiting for docker to come up: $(date)"
done
