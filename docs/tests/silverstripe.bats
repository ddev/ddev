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

  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --project-type=silverstripe --docroot=public

  run -0 ddev start -y

  _extra_info

  run -0 _retry ddev composer create-project --prefer-dist silverstripe/installer

  run -0 ddev sake dev/build flush=all

  DDEV_DEBUG=true run -0 ddev launch /admin
  assert_output "FULLURL ${PRIMARY_URL}/admin"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}/Security/login
  assert_output --partial "server: Apache"
  assert_output --partial "HTTP/2 200"
  _assert_page_contains ${PRIMARY_URL} "Welcome to Silverstripe"
  _assert_page_contains ${PRIMARY_URL}/Security/login \
    "<title>Your Site Name: Log in</title>" \
    "id=\"MemberLoginForm_LoginForm\""
}

@test "Silverstripe CMS Git Clone  quickstart with $(ddev --version)" {

  skip "Does not have a test yet"
}
