#!/usr/bin/env bats

setup() {
  PROJNAME=my-cakephp-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "CakePHP Composer quickstart with $(ddev --version)" {
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  # ddev config --project-type=cakephp --docroot=webroot
  run ddev config --project-type=cakephp --docroot=webroot
  assert_success

  # ddev start -y
  run ddev start -y
  assert_success

  # ddev composer create --prefer-dist --no-interaction cakephp/app:~5.0
  run ddev composer create --prefer-dist --no-interaction cakephp/app:~5.0
  assert_success

  # validate ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "CakePHP: the rapid development PHP framework:"
  assert_output --partial "Welcome to CakePHP"
}
