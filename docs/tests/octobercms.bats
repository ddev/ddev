#!/usr/bin/env bats

setup() {
  PROJNAME=my-october-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "October CMS new project quickstart with $(ddev --version)" {
  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --project-type=laravel --webserver-type=apache-fpm

  run -0 ddev start -y

  _extra_info

  run -0 ddev composer create-project october/october

  run -0 ddev artisan october:install --no-interaction

  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch
  assert_output --partial "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "HTTP/2 200"

  # check October CMS is serving pages
  _assert_page_contains ${PRIMARY_URL} "Welcome to October CMS!"

  # check admin is accessible (follows redirect to login page)
  _assert_page_contains -L ${PRIMARY_URL}/admin "Administration Area | October CMS"

  # check admin setup page is accessible
  run -0 curl -sSfI -L ${PRIMARY_URL}/admin/backend/auth/setup
  assert_output --partial "HTTP/2 200"

  # check used database
  run -0 ddev artisan about
  assert_output --partial "mariadb"
}

@test "October CMS existing project quickstart with $(ddev --version)" {
  skip "Does not have a test yet"
}
