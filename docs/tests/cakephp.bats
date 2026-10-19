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
  PROJNAME=my-cakephp-composer-site

  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --project-type=cakephp --docroot=webroot

  run -0 ddev start -y

  run -0 _retry ddev composer create-project --prefer-dist --no-interaction cakephp/app:~5.0

  _extra_info

  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"
  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  _assert_page_contains ${PRIMARY_URL} \
    "CakePHP: the rapid development PHP framework:" \
    "Welcome to CakePHP"
}
