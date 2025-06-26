#!/bin/bash

bats_require_minimum_version 1.11.0
set -eu -o pipefail
brew_prefix=$(brew --prefix)
load "${brew_prefix}/lib/bats-support/load.bash"
load "${brew_prefix}/lib/bats-assert/load.bash"
load "${brew_prefix}/lib/bats-file/load.bash"
