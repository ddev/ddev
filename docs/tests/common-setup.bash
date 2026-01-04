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
    export DDEV_NONINTERACTIVE=true
    mkdir -p ${tmpdir} && cd ${tmpdir} || exit 1
    ddev delete -Oy ${PROJNAME:-notset} >/dev/null
#    echo "# Starting test at $(date)" >&3
}

_extra_info() {
  HOST_HTTP_URL=$(ddev describe -j ${PROJNAME} | jq -r .raw.services.web.host_http_url)
  HOST_HTTPS_URL=$(ddev describe -j ${PROJNAME} | jq -r .raw.services.web.host_https_url)
  PRIMARY_HTTP_URL=$(ddev describe -j ${PROJNAME} | jq -r .raw.httpurl)
  PRIMARY_HTTPS_URL=$(ddev describe -j ${PROJNAME} | jq -r .raw.httpsurl)
}

_common_teardown() {
#  echo "# Ending test at $(date)" >&3
  ddev delete -Oy ${PROJNAME} >/dev/null
  rm -rf ${tmpdir}
}
