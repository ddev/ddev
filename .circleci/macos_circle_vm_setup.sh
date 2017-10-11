#!/usr/bin/env bash

set -o errexit
set -x

# Basic tools

brew update && brew install mariadb coreutils golang && brew cask install docker

# macOS version
sw_vers
