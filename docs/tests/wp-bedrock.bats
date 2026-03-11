#!/usr/bin/env bats

setup() {
  PROJNAME=my-wp-bedrock-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "WordPress Bedrock quickstart with $(ddev --version)" {
  _skip_if_embargoed "wp-bedrock"
  PROJNAME=my-wp-bedrock-site

  run mkdir -p my-wp-bedrock-site && cd my-wp-bedrock-site
  assert_success
  run ddev config --project-type=wp-bedrock
  assert_success
  run ddev start -y
  assert_success
  run ddev composer create-project roots/bedrock
  assert_success
  run ddev wp core install --url='https://${PROJNAME}.ddev.site' --title='My Bedrock Site' --admin_user=admin --admin_password=admin --admin_email=admin@example.com
  assert_success
  DDEV_DEBUG=true run ddev launch
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_line --regexp "link:.*${PROJNAME}\.ddev\.site.*rel=\"https://api\.w\.org/\""
  assert_output --partial "HTTP/2 200"
  assert_success
  # validate running project /wp-admin/
  # Some environments return 302 redirect to /wp/wp-admin/, others return 200
  run curl -sfI https://${PROJNAME}.ddev.site/wp-admin/
  assert_line --regexp "link:.*${PROJNAME}\.ddev\.site.*rel=\"https://api\.w\.org/\""
  assert_line --regexp 'HTTP/2 (200|302)'
  assert_success
}
