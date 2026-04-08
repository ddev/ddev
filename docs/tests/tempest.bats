#!/usr/bin/env bats

setup() {
  PROJNAME=my-tempest-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Tempest composer based quickstart with $(ddev --version)" {
  _skip_if_embargoed "tempest-composer"

  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=php --docroot=public --php-version=8.5 --omit-containers=db
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project tempest/app
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "HTTP/2 200"
  assert_success

  run curl -sfv https://${PROJNAME}.ddev.site
  assert_output --partial "Tempest"
  assert_output --partial "The PHP framework that gets out of your way."
  assert_success

  # check used database
  run ddev exec tempest about
  assert_success
  assert_output --partial "SQLite"
}
