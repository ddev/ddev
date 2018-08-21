#!/usr/bin/env bash

set -o errexit
set -x

# Basic tools

sudo apt-get update -qq
sudo apt-get install -qq mysql-client realpath zip nsis

# golang of the version we want
sudo apt-get remove -qq golang && sudo rm -rf /usr/local/go &&
wget -q -O /tmp/golang.tgz https://dl.google.com/go/go1.10.3.linux-amd64.tar.gz &&
sudo tar -C /usr/local -xzf /tmp/golang.tgz


# docker-compose
sudo curl -s -L "https://github.com/docker/compose/releases/download/1.22.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Remove existing docker. This section should not be required after switch to
# circleci/classic:201808-01 image
#sudo apt-get remove docker docker-engine docker.io
#sudo apt-get install \
#    apt-transport-https \
#    ca-certificates \
#    curl \
#    software-properties-common
#curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
#sudo add-apt-repository \
#   "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
#   $(lsb_release -cs) \
#   stable"
#sudo apt-get update -qq
#sudo apt-get install -qq docker-ce
#
