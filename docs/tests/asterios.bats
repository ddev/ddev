#!/usr/bin/env bats

setup() {
  PROJNAME=my-asterios-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Asterios composer based quickstart with $(ddev --version)" {
  run mkdir -p my-asterios-site && cd my-asterios-site
  assert_success

  run ddev config --project-type=asterios --docroot=public
  assert_success

  run ddev start -y
  assert_success

  _extra_info

  run ddev composer create-project asterios/app
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output --partial "FULLURL ${PRIMARY_URL}"
  assert_success

  # validate running project - check status code
  run curl -sf -o /dev/null -w "%{http_code}" ${PRIMARY_URL}
  assert_success
  assert_output "200"

  # validate running project - check content
  run curl -sf ${PRIMARY_URL}
  assert_success
  assert_output --partial "Build your next project with de.asteriosphp.app"
}
