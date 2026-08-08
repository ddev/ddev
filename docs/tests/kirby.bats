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
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --omit-containers=db --webserver-type=apache-fpm
  assert_success

  run ddev start -y
  assert_success

  _extra_info

  run ddev composer create-project getkirby/starterkit
  assert_success

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"
  assert_success

  run curl -sfIv ${PRIMARY_URL}
  assert_output --partial "server: Apache"
  assert_output --partial "HTTP/2 200"
  assert_success
  run curl -sfv ${PRIMARY_URL}
  assert_output --partial "<h2><a href=\"https://getkirby.com\">Made with Kirby</a></h2>"
  assert_output --partial "the file-based CMS that adapts to any project"
  assert_success
}
