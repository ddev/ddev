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
  _skip_if_embargoed "asterios-composer"

  run mkdir -p my-asterios-site && cd my-asterios-site
  assert_success

  run ddev config --project-type=asterios --docroot=public
  assert_success

  run ddev start -y
  assert_success

  run ddev composer create-project asterios/app
  assert_success

  DDEV_DEBUG=true run ddev launch
  assert_output --partial "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  # validate running project - check status code
  run curl -sf -o /dev/null -w "%{http_code}" https://${PROJNAME}.ddev.site
  assert_success
  assert_output "200"

  # validate running project - check content
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "AsteriosPHP Framework"
}
