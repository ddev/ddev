#!/bin/bash

# brew install bats-core bats-core/bats-core/bats-assert bats-core/bats-core/bats-file bats-core/bats-core/bats-support

bats_require_minimum_version 1.11.0
set -eu -o pipefail
TEST_BREW_PREFIX="$(brew --prefix 2>/dev/null || true)"
export BATS_LIB_PATH="${BATS_LIB_PATH:-}:${TEST_BREW_PREFIX}/lib:/usr/lib/bats"
bats_load_library bats-support
bats_load_library bats-assert
bats_load_library bats-file
