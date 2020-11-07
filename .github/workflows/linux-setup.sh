#!/bin/bash

# Install misc dependencies
sudo apt-get update -qq
sudo apt-get install mysql-client zip jq expect nfs-kernel-server build-essential curl git libnss3-tools libcurl4-gnutls-dev
brew install make mysql-client mkcert

mkcert -install
