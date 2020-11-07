#!/bin/bash

# Install misc dependencies
sudo apt-get update -qq
sudo apt-get install build-essential libnss3-tools libcurl4-gnutls-dev
brew install make mysql-client mkcert
