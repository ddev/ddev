#!/bin/bash

# Install misc dependencies
sudo apt-get update -qq
sudo apt-get install build-essential
brew install make mysql-client mkcert
