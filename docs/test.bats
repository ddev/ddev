#!/usr/bin/env bats

# Test DDEV Quickstarts. This must be maintaintained along with the docs.
# bats_assert is required.

setup() {
  TEST_BREW_PREFIX="$(brew --prefix)"
  load "${TEST_BREW_PREFIX}/lib/bats-support/load.bash"
  load "${TEST_BREW_PREFIX}/lib/bats-assert/load.bash"
  tmpdir=~/tmp/quickstart-test
  mkdir -p ${tmpdir} && cd ${tmpdir} || exit
  }

cleanup() {
  ddev delete -Oy my-backdrop-site
}
# Each test:
# walk through the tings quickstart has to do
# Each of these should be in separate file

#mkdir my-backdrop-site && cd my-backdrop-site
#curl -LJO https://github.com/backdrop/backdrop/releases/latest/download/backdrop.zip
#unzip ./backdrop.zip && rm -f backdrop.zip && mv -f ./backdrop/{.,}* . ; rm -rf backdrop
#ddev config --project-type=backdrop
#ddev start
#ddev launch

@test "backdrop quickstart" {
    run mkdir -p my-backdrop-site && cd my-backdrop-site
    assert_success
    curl -LJO https://github.com/backdrop/backdrop/releases/latest/download/backdrop.zip
    assert_success
    unzip ./backdrop.zip && rm -f backdrop.zip && mv -f ./backdrop/{.,}* . ; rm -rf backdrop
    assert_success
    ddev config --project-type=backdrop
    assert_success
    ddev start
    assert_success
    DDEV_DEBUG=true ddev launch
    assert_output "FULLURL https://backdrop.ddev.site"
    assert_success
}
