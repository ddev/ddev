#!/usr/bin/env bats

setup() {
  PROJNAME=my-kirby-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Kirby new project quickstart with $(ddev --version)" {

  # mkdir ${PROJNAME} && cd ${PROJNAME}
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  # ddev config --omit-containers=db --webserver-type=apache-fpm
  run ddev config --omit-containers=db --webserver-type=apache-fpm
  assert_success

  # ddev start -y
  run ddev start -y
  assert_success

  # ddev composer create-project getkirby/starterkit
  run ddev composer create-project getkirby/starterkit
  assert_success

  # validate ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success

  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "server: Apache"
  assert_output --partial "HTTP/2 200"
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "<h2><a href=\"https://getkirby.com\">Made with Kirby</a></h2>"
  assert_output --partial "the file-based CMS that adapts to any project"
}
