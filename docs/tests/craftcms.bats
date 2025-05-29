#!/usr/bin/env bats

setup() {
  PROJNAME=my-craft-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}
@test "Craft CMS New Projects quickstart with $(ddev --version)" {
  run mkdir ${PROJNAME} && cd ${PROJNAME}
  assert_success

  run ddev config --project-type=craftcms --docroot=web
  assert_success

  run ddev start -y
  assert_success

  # Username: [admin] admin
  # Email: admin@example.com
  # Password: Password123
  # Confirm: Password123
  # Site name: CraftCMS
  # Site URL: [https://my-craft-site.ddev.site]
  # Site language: [en]
  run bats_pipe printf "admin\nadmin@example.com\nPassword123\nPassword123\nCraftCMS\n\n\n" \| ddev composer create-project craftcms/craft
  assert_success

  # validate ddev launch
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  DDEV_DEBUG=true run ddev launch /admin/login
  assert_output "FULLURL https://${PROJNAME}.ddev.site/admin/login"
  assert_success

  ## validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_output --partial "x-powered-by: Craft CMS"
  run curl -sf https://${PROJNAME}.ddev.site
  assert_success
  assert_output --partial "<title>Welcome to Craft CMS</title>"
  assert_output --partial "<h2>Popular Resources</h2>"
  run curl -sf https://${PROJNAME}.ddev.site/admin/login
  assert_success
  assert_output --partial "<title>Sign In - CraftCMS</title>"
}

@test "Craft CMS Existing Projects quickstart with $(ddev --version)" {
  skip "Does not have a test yet"
}
