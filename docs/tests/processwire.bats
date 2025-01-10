#!/usr/bin/env bats

setup() {
  PROJNAME=my-processwire-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "processwire composer with $(ddev --version)" {
  # mkdir my-processwire-site && cd my-processwire-site
  run mkdir -p my-processwire-site && cd my-processwire-site
  assert_success
  # ddev config --project-type=php --webserver-type=apache-fpm
  run ddev config --project-type=php --webserver-type=apache-fpm
  assert_success
  # ddev start
  run ddev start -y
  assert_success
  # ddev composer create "processwire/processwire:^3"
  run ddev composer create "processwire/processwire:^3"
  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "server: Apache"
}
