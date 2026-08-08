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

  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=laravel --docroot=public
  assert_success

  run ddev composer create-project --prefer-dist statamic/statamic
  assert_success

  _extra_info

  # fill out the interactive form
  run ddev php please make:user admin@example.com --password=admin1234 --super --no-interaction
  ddev mutagen sync
  assert_file_exist users/admin@example.com.yaml


  # validate ddev launch
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"
  assert_success
  DDEV_DEBUG=true run ddev launch /cp
  assert_output "FULLURL ${PRIMARY_URL}/cp"
  assert_success

    # validate running project
  run curl -sfIv ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_output --partial "x-powered-by: Statamic"
  assert_success
  run curl -sfIv ${PRIMARY_URL}/cp/auth/login
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_output --partial "x-powered-by: Statamic"
  assert_success
  run curl -sfv ${PRIMARY_URL}
  assert_output --partial "<title>Home</title>"
  assert_output --partial "Welcome to your new Statamic site"
  assert_success
  run curl -sfv ${PRIMARY_URL}/cp/auth/login
  assert_output --regexp 'component.*auth.*Login'
  assert_success
}


@test "Statamic Git Clone quickstart with $(ddev --version)" {
  skip "Does not have a test yet"
}
