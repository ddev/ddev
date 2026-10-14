#!/usr/bin/env bats

setup() {
  PROJNAME=my-statamic-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Statamic Composer quickstart with $(ddev --version)" {
  PROJNAME=my-statamic-composer-site

  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --project-type=laravel --docroot=public

  run -0 ddev composer create-project --prefer-dist statamic/statamic

  _extra_info

  # fill out the interactive form
  run ddev php please make:user admin@example.com --password=admin1234 --super --no-interaction
  ddev mutagen sync
  assert_file_exist users/admin@example.com.yaml


  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"
  DDEV_DEBUG=true run -0 ddev launch /cp
  assert_output "FULLURL ${PRIMARY_URL}/cp"

    # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_output --partial "x-powered-by: Statamic"
  run -0 curl -sSfI ${PRIMARY_URL}/cp/auth/login
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_output --partial "x-powered-by: Statamic"
  _assert_page_contains ${PRIMARY_URL} \
    "<title>Home</title>" \
    "Welcome to your new Statamic site"
  run -0 curl -sSf ${PRIMARY_URL}/cp/auth/login
  assert_output --regexp 'component.*auth.*Login'
}


@test "Statamic Git Clone quickstart with $(ddev --version)" {
  skip "Does not have a test yet"
}
