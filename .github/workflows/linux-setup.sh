#!/bin/bash

# Install misc dependencies
sudo apt-get update -qq
sudo apt-get install build-essential libnss3-tools
brew install make mysql-client mkcert
