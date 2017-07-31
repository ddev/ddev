#!/usr/bin/env bash

set -o errexit

# Basic tools

# mysql - old key is expired, so get the new one
sudo apt-key adv --keyserver pgp.mit.edu --recv-keys 5072E1F5 >/dev/null 2>&1
sudo apt-get update -qq
sudo apt-get install -qq mysql-client realpath zip

# golang of the version we want
sudo apt-get remove -qq golang &&
wget -q -O /tmp/golang.tgz https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz &&
cd /tmp && tar -xf golang.tgz &&
sudo rm -rf /usr/local/go && sudo mv go /usr/local

# Docker setup
sudo apt-get remove -qq docker docker-engine
sudo apt-get update -qq
sudo apt-get install -qq apt-transport-https ca-certificates  curl software-properties-common
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
# Per docker docs, you always need the stable repository, even if you want to install edge builds as well.
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
sudo apt-get update -qq
sudo apt-get install -qq docker-ce=17.06.0~ce-0~ubuntu

# docker-compose
sudo curl -s -L "https://github.com/docker/compose/releases/download/1.14.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
