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
  run -0 mkdir -p ${PROJNAME}
  cd ${PROJNAME}

  run -0 ddev config --project-type=craftcms --docroot=web

  run -0 ddev start -y

  _extra_info

  # Username: [admin] admin
  # Email: admin@example.com
  # Password: Password123
  # Confirm: Password123
  # Site name: CraftCMS
  # Site URL: [https://my-craft-site.ddev.site]
  # Site language: [en]
  run -0 bats_pipe printf "admin\nadmin@example.com\nPassword123\nPassword123\nCraftCMS\n\n\n" \| ddev composer create-project craftcms/craft

  # validate ddev launch
  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"
  DDEV_DEBUG=true run -0 ddev launch /admin/login
  assert_output "FULLURL ${PRIMARY_URL}/admin/login"

  ## validate running project
  run -0 curl -sSfI ${PRIMARY_URL}
  assert_output --partial "server: nginx"
  assert_output --partial "HTTP/2 200"
  assert_output --partial "x-powered-by: Craft CMS"
  _assert_page_contains ${PRIMARY_URL} \
    "<title>Welcome to Craft CMS</title>" \
    "<h2>Popular Resources</h2>"
  _assert_page_contains ${PRIMARY_URL}/admin/login "<title>Sign In - CraftCMS</title>"
}

@test "Craft CMS Existing Projects quickstart with $(ddev --version)" {
  skip "Does not have a test yet"
}
