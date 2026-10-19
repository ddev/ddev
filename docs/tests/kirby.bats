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
  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}

  run -0 ddev config --omit-containers=db --webserver-type=apache-fpm

  run -0 ddev start -y

  _extra_info

  run -0 _retry ddev composer create-project getkirby/starterkit

  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"

  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: Apache"
  assert_output --partial "HTTP/2 200"
  _assert_page_contains ${PRIMARY_URL} \
    "<h2><a href=\"https://getkirby.com\">Made with Kirby</a></h2>" \
    "the file-based CMS that adapts to any project"
}
