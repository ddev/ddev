#!/usr/bin/env bash

set -o errexit
set -x

# Basic tools

sudo apt-get update -qq
sudo apt-get install -qq mysql-client realpath zip

go version

# golang of the version we want
sudo apt-get remove -qq golang &&
wget -q -O /tmp/golang.tgz https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz &&
cd /tmp && tar -xf golang.tgz &&
sudo rm -rf /usr/local/go && sudo mv go /usr/local

# docker-compose
sudo curl -s -L "https://github.com/docker/compose/releases/download/1.16.1/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
