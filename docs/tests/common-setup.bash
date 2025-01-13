#!/usr/bin/env bash

_common_setup() {
    TEST_BREW_PREFIX="$(brew --prefix 2>/dev/null || true)"
    export BATS_LIB_PATH="${BATS_LIB_PATH}:${TEST_BREW_PREFIX}/lib:/usr/lib/bats"
    bats_load_library bats-support
    bats_load_library bats-assert
    bats_load_library bats-file
    mkdir -p ~/tmp
    tmpdir=$(mktemp -d ~/tmp/${PROJNAME}.XXXXXX)
    export DDEV_NO_INSTRUMENTATION=true
    mkdir -p ${tmpdir} && cd ${tmpdir} || exit 1
    ddev delete -Oy ${PROJNAME:-notset}
}

_common_teardown() {
  ddev delete -Oy ${PROJNAME}
  rm -rf ${tmpdir}
}
