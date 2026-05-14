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

  DDEV_DEBUG=true run ddev launch /wp/wp-admin/
  assert_output "FULLURL https://${PROJNAME}.ddev.site/wp/wp-admin/"
  assert_success

  # validate running project
  run curl -sfI https://${PROJNAME}.ddev.site
  assert_line --regexp "link:.*${PROJNAME}\.ddev\.site.*rel=\"https://api\.w\.org/\""
  assert_output --partial "HTTP/2 200"
  assert_success

  # validate /wp/wp-admin/ redirects to login when unauthenticated
  run curl -sfI https://${PROJNAME}.ddev.site/wp/wp-admin/
  assert_output --partial "location: https://${PROJNAME}.ddev.site/wp/wp-login.php"
  assert_output --partial "HTTP/2 302"
  assert_success

  # validate actual login: POST credentials, follow redirect to wp-admin, check for Dashboard
  COOKIE_JAR=$(mktemp)
  curl -sf -c "${COOKIE_JAR}" https://${PROJNAME}.ddev.site/wp/wp-login.php > /dev/null
  run curl -sf -c "${COOKIE_JAR}" -b "${COOKIE_JAR}" \
    --data "log=admin&pwd=admin&wp-submit=Log+In&redirect_to=%2Fwp%2Fwp-admin%2F&testcookie=1" \
    --location https://${PROJNAME}.ddev.site/wp/wp-login.php
  assert_output --partial "Dashboard"
  assert_success
  rm -f "${COOKIE_JAR}"
}
