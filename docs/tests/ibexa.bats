#!/usr/bin/env bats

setup() {
  PROJNAME=my-ibexa-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "Ibexa DXP quickstart with $(ddev --version)" {
  # mkdir -p ${PROJNAME} && cd ${PROJNAME}
  run -0 mkdir -p ${PROJNAME} && cd ${PROJNAME}
  run -0 ddev config --project-type=symfony --docroot=public
  run -0 ddev start -y
  run -0 ddev composer create-project ibexa/oss-skeleton
  run -0 ddev console ibexa:install --no-interaction
  run -0 ddev console ibexa:graphql:generate-schema

  _extra_info

  DDEV_DEBUG=true run -0 ddev launch /admin/login
  assert_output "FULLURL ${PRIMARY_URL}/admin/login"
  # validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "x-powered-by: Ibexa Open Source v5"
  assert_output --partial "HTTP/2 200"

  _assert_page_contains ${PRIMARY_URL} \
    "Open-source solution for building custom, scalable websites." \
    "Powered by Ibexa DXP"

  _assert_page_contains ${PRIMARY_URL}/admin/login \
    "Welcome to<br/> Ibexa DXP" \
    "<h3 class=\"ibexa-login__support-headline\">Get to know Ibexa DXP</h3>"
}
