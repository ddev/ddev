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
  _skip_if_embargoed "ibexa-composer"

  # mkdir -p ${PROJNAME} && cd ${PROJNAME}
  run mkdir -p ${PROJNAME} && cd ${PROJNAME}
  assert_success
  run ddev config --project-type=php --docroot=public --web-environment-add DATABASE_URL=mysql://db:db@db:3306/db
  assert_success
  run ddev start -y
  assert_success
  run ddev composer create-project ibexa/oss-skeleton
  assert_success
  run ddev exec console ibexa:install --no-interaction
  assert_success
  DDEV_DEBUG=true run ddev launch /admin/login
  assert_output "FULLURL https://${PROJNAME}.ddev.site/admin/login"
  assert_success
  # validate running project
  run curl -sfIv https://${PROJNAME}.ddev.site
  assert_output --partial "x-powered-by: Ibexa Open Source v5"
  assert_output --partial "HTTP/2 200"
  assert_success

  run curl -sfv https://${PROJNAME}.ddev.site
  assert_output --partial "Open-source solution for building custom, scalable websites."
  assert_output --partial "Powered by Ibexa DXP"
  assert_success

  run curl -sfv https://${PROJNAME}.ddev.site/admin/login
  assert_output --partial "Welcome to<br/> Ibexa DXP"
  assert_output --partial "<h3 class=\"ibexa-login__support-headline\">Get to know Ibexa DXP</h3>"
  assert_success
}
