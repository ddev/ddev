#!/usr/bin/env bats

setup() {
  PROJNAME=my-silverstripe-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Silverstripe CMS Composer quickstart with $(ddev --version)" {
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=silverstripe --docroot=public
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project --prefer-dist silverstripe/installer
  assert_success

  run ddev sake dev/build flush=all
  assert_success

  DDEV_DEBUG=true run ddev launch /admin
  assert_output "FULLURL https://${PROJNAME}.ddev.site/admin"
  assert_success

  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/Security/login
  assert_success
  assert_output --partial "server: Apache"
  assert_output --partial "HTTP/2 200"
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "Welcome to Silverstripe"
  run curl -sf https://${PROJNAME}.ddev.site/Security/login
  assert_success
  assert_output --partial "<title>Your Site Name: Log in</title>"
  assert_output --partial "id=\"MemberLoginForm_LoginForm\""
}

@test "Silverstripe CMS Git Clone  quickstart with $(ddev --version)" {
  skip "Does not have a test yet"
}
