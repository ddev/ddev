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
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  # ddev config --project-type=silverstripe --docroot=public
  run ddev config --project-type=silverstripe --docroot=public
  assert_success

  # ddev config --project-type=silverstripe --docroot=public
  run ddev start -y
  assert_success

  # ddev sake dev/build flush=all
  run ddev composer create-project --prefer-dist silverstripe/installer
  assert_success

  # ddev sake dev/build flush=all
  run ddev sake dev/build flush=all
  assert_success

  # validate ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch /admin"
  assert_output "FULLURL https://${PROJNAME}.ddev.site/admin"
  assert_success

  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site/Security/login
  assert_success
  assert_output --partial "server: Apache"
  assert_output --partial "HTTP/2 200"
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "Powered by <a href=\"http://silverstripe.org\">SilverStripe</a>"
  run curl -sf https://${PROJNAME}.ddev.site/Security/login
  assert_success
  assert_output --partial "<title>Your Site Name: Log in</title>"
  assert_output --partial "id=\"MemberLoginForm_LoginForm\""
}

@test "Silverstripe CMS Git Clone  quickstart with $(ddev --version)" {
  skip "Does not have a test yet"
}
