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
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=laravel --webserver-type=apache-fpm
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project october/october
  assert_success

  run ddev artisan october:install --no-interaction
  assert_success

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "HTTP/2 200"
  assert_success

  # check October CMS is serving pages
  run curl -sfv https://${PROJNAME}.ddev.site
  assert_output --partial "Welcome to October CMS!"
  assert_success

  # check admin is accessible (follows redirect to login page)
  run curl -sfv -L https://${PROJNAME}.ddev.site/admin
  assert_output --partial "Administration Area | October CMS"
  assert_success

  # check admin setup page is accessible
  run curl -sfIv -L https://${PROJNAME}.ddev.site/admin/backend/auth/setup
  assert_output --partial "HTTP/2 200"
  assert_success

  # check used database
  run ddev artisan about
  assert_success
  assert_output --partial "mariadb"
}

@test "October CMS existing project quickstart with $(ddev --version)" {
  skip "Does not have a test yet"
}
