#!/usr/bin/env bats

setup() {
  PROJNAME=my-codeigniter-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "CodeIgniter composer based quickstart with $(ddev --version)" {
  run -0 mkdir -p my-codeigniter-site && cd my-codeigniter-site

  run -0 ddev config --project-type=codeigniter --docroot=public

  run -0 ddev start -y

  _extra_info

  run -0 _retry ddev composer create-project codeigniter4/appstarter

  DDEV_DEBUG=true run -0 ddev launch
  assert_output --partial "FULLURL ${PRIMARY_URL}"

  # validate running project - check status code
  run -0 curl -sf -o /dev/null -w "%{http_code}" ${PRIMARY_URL}
  assert_output "200"

  # validate running project - check content
  run -0 curl -sf ${PRIMARY_URL}
  assert_output --partial "Welcome to CodeIgniter 4"
}
