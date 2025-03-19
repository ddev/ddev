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
  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  # ddev config --project-type=laravel --docroot=public
  run ddev config --project-type=laravel --docroot=public
  assert_success

  # ddev composer create --prefer-dist statamic/statamic
  run ddev composer create --prefer-dist statamic/statamic
  assert_success

  # fill out the interactive form
  run ddev php please make:user admin@example.com --password=admin1234 --super --no-interaction
  ddev mutagen sync
  assert_file_exist users/admin@example.com.yaml


  # validate ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  run bash -c "DDEV_DEBUG=true ddev launch /cp"
  assert_output "FULLURL https://${PROJNAME}.ddev.site/cp"
  assert_success

    # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_output --partial "x-powered-by: Statamic"
  run curl -sfI https://${PROJNAME}.ddev.site/cp/auth/login
  assert_success
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_output --partial "x-powered-by: Statamic"
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "<title>Home</title>"
  assert_output --partial "<li><a href=\"https://statamic.dev\">Head to the docs</a> and learn how Statamic&nbsp;works.</li>"
  run curl -sf https://${PROJNAME}.ddev.site/cp/auth/login
  assert_success
  assert_output --partial "<title>Log in ‹ Statamic</title>"
}


@test "Statamic Git Clone quickstart with $(ddev --version)" {
  skip "Does not have a test yet"
}
