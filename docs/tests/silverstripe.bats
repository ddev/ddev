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
  PROJNAME=my-silverstripe-composer-site

  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=silverstripe --docroot=public
  assert_success

  run ddev start -y
  assert_success

  _extra_info

  run ddev composer create-project --prefer-dist silverstripe/installer
  assert_success

  run ddev sake dev/build flush=all
  assert_success

  DDEV_DEBUG=true run ddev launch /admin
  assert_output "FULLURL ${PRIMARY_URL}/admin"
  assert_success

  # validate running project
  run curl -sfIv ${PRIMARY_URL}/Security/login
  assert_output --partial "server: Apache"
  assert_output --partial "HTTP/2 200"
  assert_success
  run curl -sfv ${PRIMARY_URL}
  assert_output --partial "Welcome to Silverstripe"
  assert_success
  run curl -sfv ${PRIMARY_URL}/Security/login
  assert_output --partial "<title>Your Site Name: Log in</title>"
  assert_output --partial "id=\"MemberLoginForm_LoginForm\""
  assert_success
}

@test "Silverstripe CMS Git Clone  quickstart with $(ddev --version)" {

  skip "Does not have a test yet"
}
