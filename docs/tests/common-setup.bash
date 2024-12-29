#!/usr/bin/env bash

_common_setup() {
    TEST_BREW_PREFIX="$(brew --prefix)"
    load "${TEST_BREW_PREFIX}/lib/bats-support/load.bash"
    load "${TEST_BREW_PREFIX}/lib/bats-assert/load.bash"
    tmpdir=$(mktemp -d ~/tmp/${PROJNAME}.XXXXXX)
    export DDEV_NO_INSTRUMENTATION=true
    mkdir -p ${tmpdir} && cd ${tmpdir} || exit
    ddev delete -Oy ${PROJNAME:-notset}
}

_common_teardown() {
  ddev delete -Oy ${PROJNAME}
  rm -rf ${tmpdir}/${PROJNAME}
}
