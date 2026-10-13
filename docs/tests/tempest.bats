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
  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --project-type=php --docroot=public --php-version=8.5 --omit-containers=db

  run -0 ddev start -y

  _extra_info

  run -0 ddev composer create-project tempest/app

  DDEV_DEBUG=true run -0 ddev launch
  assert_output --partial "FULLURL ${PRIMARY_URL}"

  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "HTTP/2 200"

  _assert_page_contains ${PRIMARY_URL} \
    "Tempest" \
    "The PHP framework that gets out of your way."

  # check used database
  run -0 ddev exec tempest about
  assert_output --partial "SQLite"
}
